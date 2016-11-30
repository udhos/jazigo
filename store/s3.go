package store

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

var awsSession *session.Session
var s3Session *s3.S3
var s3logger hasPrintf
var s3region string

func s3client() *s3.S3 {
	if awsSession == nil {
		var err error
		awsSession, err = session.NewSession()
		if err != nil {
			s3log("s3client: could not create session: %v", err)
			return nil
		}
		s3log("s3client: session created")
	}

	if s3Session == nil {
		s3Session = s3.New(awsSession, aws.NewConfig().WithRegion(s3region))
		s3log("s3client: client created for region: [%s]", s3region)
	}

	return s3Session
}

func s3init(logger hasPrintf, region string) {
	if s3logger != nil {
		panic("s3 store reinitialization")
	}
	if logger == nil {
		panic("s3 store nil logger")
	}
	s3region = region
	s3logger = logger
	s3log("initialized: region: " + s3region)
}

func s3log(format string, v ...interface{}) {
	if s3logger == nil {
		log.Printf("s3 store (unitialized): "+format, v...)
		return
	}
	s3logger.Printf("s3 store: "+format, v...)
}

// S3Path checks if path is an aws s3 path.
func S3Path(path string) bool {
	return s3path(path)
}

func s3path(path string) bool {
	s3match := strings.HasPrefix(path, "arn:aws:s3:")
	if s3match {
		s3log("s3path: [%s]", path)
	}
	return s3match
}

// "arn:aws:s3:::bucket/folder/file.xxx"
// => "bucket", "folder/file.xxx"
func s3parse(path string) (string, string) {
	s := strings.Split(path, ":")
	if len(s) < 6 {
		return "", ""
	}
	file := s[5]
	slash := strings.IndexByte(file, '/')
	if slash < 1 {
		return "", ""
	}
	bucket := file[:slash]
	key := file[slash+1:]
	//s3log("s3parse: [%s] => bucket=[%s] key=[%s]", path, bucket, key)
	return bucket, key
}

func s3fileExists(path string) bool {

	svc := s3client()
	if svc == nil {
		s3log("s3fileExists: missing s3 client: ugh")
		return false // ugh
	}

	bucket, key := s3parse(path)

	params := &s3.HeadObjectInput{
		Bucket: aws.String(bucket), // Required
		Key:    aws.String(key),    // Required
	}
	if _, err := svc.HeadObject(params); err == nil {
		//s3log("s3fileExists: FOUND [%s]", path)
		return true // found
	}

	return false
}

func s3fileput(path string, buf []byte) error {

	svc := s3client()
	if svc == nil {
		return fmt.Errorf("s3fileput: missing s3 client")
	}

	bucket, key := s3parse(path)

	params := &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(buf),
	}
	_, err := svc.PutObject(params)

	//s3log("s3fileput: [%s] upload: error: %v", path, err)

	return err
}

func s3fileRemove(path string) error {

	svc := s3client()
	if svc == nil {
		return fmt.Errorf("s3fileRemove: missing s3 client")
	}

	bucket, key := s3parse(path)

	params := &s3.DeleteObjectInput{
		Bucket: aws.String(bucket), // Required
		Key:    aws.String(key),    // Required
	}
	_, err := svc.DeleteObject(params)

	//s3log("s3fileRemove: [%s] delete: error: %v", path, err)

	return err
}

func s3fileRename(p1, p2 string) error {

	svc := s3client()
	if svc == nil {
		return fmt.Errorf("s3fileRename: missing s3 client")
	}

	bucket1, key1 := s3parse(p1)
	bucket2, key2 := s3parse(p2)

	params := &s3.CopyObjectInput{
		Bucket:     aws.String(bucket2),              // Required
		CopySource: aws.String(bucket1 + "/" + key1), // Required
		Key:        aws.String(key2),                 // Required
	}
	_, copyErr := svc.CopyObject(params)
	if copyErr != nil {
		return copyErr
	}

	if removeErr := s3fileRemove(p1); removeErr != nil {
		// could not remove old file
		s3fileRemove(p2) // remove new file (clean up)
		return removeErr
	}

	return nil
}

func s3fileRead(path string) ([]byte, error) {

	svc := s3client()
	if svc == nil {
		return nil, fmt.Errorf("s3fileRead: missing s3 client")
	}

	bucket, key := s3parse(path)

	params := &s3.GetObjectInput{
		Bucket: aws.String(bucket), // Required
		Key:    aws.String(key),    // Required
	}

	resp, err := svc.GetObject(params)
	if err != nil {
		return nil, err
	}

	s3log("s3fileRead: FIXME limit number of lines read from s3 object")

	return ioutil.ReadAll(resp.Body)
}

func s3fileFirstLine(path string) (string, error) {

	svc := s3client()
	if svc == nil {
		return "", fmt.Errorf("s3fileFirstLine: missing s3 client")
	}

	bucket, key := s3parse(path)

	params := &s3.GetObjectInput{
		Bucket: aws.String(bucket), // Required
		Key:    aws.String(key),    // Required
	}

	resp, err := svc.GetObject(params)
	if err != nil {
		return "", err
	}

	r := bufio.NewReader(resp.Body)
	line, _, readErr := r.ReadLine()

	return string(line[:]), readErr
}

func s3dirList(path string) (string, []string, error) {

	dirname := filepath.Dir(path)
	var names []string

	svc := s3client()
	if svc == nil {
		return dirname, names, fmt.Errorf("s3dirList: missing s3 client")
	}

	bucket, prefix := s3parse(path)

	params := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket), // Required
		Prefix: aws.String(prefix),
	}

	for {
		resp, err := svc.ListObjectsV2(params)
		if err != nil {
			return dirname, names, err
		}

		//s3log("s3dirList: FOUND %d keys [%s]", *resp.KeyCount, path)

		for _, obj := range resp.Contents {
			key := *obj.Key
			name := filepath.Base(key)
			//s3log("s3dirList: [%s] found: dir=[%s] file=[%s]", path, dirname, name)
			names = append(names, name)
		}

		if *resp.IsTruncated {
			params.ContinuationToken = resp.NextContinuationToken
			continue
		}

		break
	}

	//s3log("s3dirList: FOUND %d total keys [%s]", len(names), path)

	return dirname, names, nil
}

func s3dirClean(path string) error {

	// retrieve object list
	_, names, listErr := s3dirList(path)
	if listErr != nil {
		return listErr
	}

	if len(names) < 1 {
		return nil
	}

	bucket, prefix := s3parse(path)
	folder := filepath.Dir(prefix)

	svc := s3client()
	if svc == nil {
		return fmt.Errorf("s3dirClean: missing s3 client")
	}

	// build object list
	list := []*s3.ObjectIdentifier{}
	for _, filename := range names {
		key := folder + "/" + filename
		s3log("s3dirClean: [%s] bucket=[%s] key=[%s]", path, bucket, key)
		obj := &s3.ObjectIdentifier{
			Key: aws.String(key), // Required
		}
		list = append(list, obj)
	}

	// query parameters
	params := &s3.DeleteObjectsInput{
		Bucket: aws.String(bucket), // Required
		Delete: &s3.Delete{ // Required
			Objects: list, // Required
		},
	}

	// send
	_, err := svc.DeleteObjects(params)

	return err
}

func s3fileInfo(path string) (time.Time, int64, error) {

	svc := s3client()
	if svc == nil {
		return time.Time{}, 0, fmt.Errorf("s3fileInfo: missing s3 client")
	}

	bucket, key := s3parse(path)

	params := &s3.HeadObjectInput{
		Bucket: aws.String(bucket), // Required
		Key:    aws.String(key),    // Required
	}
	resp, err := svc.HeadObject(params)
	if err != nil {
		return time.Time{}, 0, err
	}

	mod := *resp.LastModified
	size := *resp.ContentLength

	return mod, size, nil
}

func s3fileCompare(p1, p2 string) (bool, error) {
	return false, fmt.Errorf("s3fileCompare: FIXME WRITEME cant currently compare files on S3: [%s,%s]", p1, p2)
}
