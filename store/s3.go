package store

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/udhos/equalfile"
)

var awsSession *session.Session
var s3SvcTable = map[string]*s3.S3{} // region => session
var s3logger hasPrintf
var s3region string // default region

func s3session() *session.Session {
	if awsSession == nil {
		var err error
		awsSession, err = session.NewSession()
		if err != nil {
			s3log("s3client: could not create session: %v", err)
			return nil
		}
		s3log("s3session: new session created")
	}
	return awsSession
}

func s3client(region string) *s3.S3 {

	if region == "" {
		region = s3region // fallback to default region
		if region == "" {
			s3log("s3client: could not find region")
			return nil
		}
	}

	svc, ok := s3SvcTable[region]
	if !ok {
		sess := s3session()
		if sess == nil {
			return nil
		}

		svc = s3.New(sess, aws.NewConfig().WithRegion(region))
		s3SvcTable[region] = svc
		s3log("s3client: client created: region=[%s]", region)
	}

	return svc
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
	s3log("initialized: default region=[%s]", s3region)
}

func s3log(format string, v ...interface{}) {
	if s3logger == nil {
		log.Printf("s3 store (uninitialized): "+format, v...)
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

//  Input: "arn:aws:s3:region::bucket/folder/file.xxx"
// Output: "region", "bucket", "folder/file.xxx"
func s3parse(path string) (string, string, string) {
	s := strings.Split(path, ":")
	if len(s) < 6 {
		return "", "", ""
	}
	region := s[3]
	file := s[5]
	slash := strings.IndexByte(file, '/')
	if slash < 1 {
		return "", "", ""
	}
	bucket := file[:slash]
	key := file[slash+1:]
	return region, bucket, key
}

func s3fileExists(path string) bool {

	region, bucket, key := s3parse(path)

	svc := s3client(region)
	if svc == nil {
		s3log("s3fileExists: missing s3 client: ugh")
		return false // ugh
	}

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

func s3fileput(path string, buf []byte, contentType string) error {

	region, bucket, key := s3parse(path)

	svc := s3client(region)
	if svc == nil {
		return fmt.Errorf("s3fileput: missing s3 client")
	}

	params := &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(buf),
	}

	switch contentType {
	case "": // none
	case "detect": // detect
		params.ContentType = aws.String(http.DetectContentType(buf))
	default: // use literal
		params.ContentType = aws.String(contentType)
	}

	_, err := svc.PutObject(params)

	//s3log("s3fileput: [%s] upload: error: %v", path, err)

	return err
}

func s3fileRemove(path string) error {

	region, bucket, key := s3parse(path)

	svc := s3client(region)
	if svc == nil {
		return fmt.Errorf("s3fileRemove: missing s3 client")
	}

	params := &s3.DeleteObjectInput{
		Bucket: aws.String(bucket), // Required
		Key:    aws.String(key),    // Required
	}
	_, err := svc.DeleteObject(params)

	//s3log("s3fileRemove: [%s] delete: error: %v", path, err)

	return err
}

func s3fileRename(p1, p2 string) error {

	region, bucket1, key1 := s3parse(p1)

	svc := s3client(region)
	if svc == nil {
		return fmt.Errorf("s3fileRename: missing s3 client")
	}

	_, bucket2, key2 := s3parse(p2)

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

func s3fileReader(path string) (io.ReadCloser, error) {

	region, bucket, key := s3parse(path)

	svc := s3client(region)
	if svc == nil {
		return nil, fmt.Errorf("s3fileRead: missing s3 client")
	}

	params := &s3.GetObjectInput{
		Bucket: aws.String(bucket), // Required
		Key:    aws.String(key),    // Required
	}

	resp, err := svc.GetObject(params)

	return resp.Body, err
}

func s3fileRead(path string, maxSize int64) ([]byte, error) {

	r, err := s3fileReader(path)
	if err != nil {
		return nil, err
	}

	defer r.Close()

	l := &io.LimitedReader{R: r, N: maxSize}

	buf, readErr := ioutil.ReadAll(r)
	if readErr != nil {
		return buf, readErr
	}

	if l.N < 1 {
		return buf, fmt.Errorf("s3fileRead: reached max=%d: remaining bytes: %d", maxSize, l.N)
	}

	return buf, nil
}

func s3fileFirstLine(path string) (string, error) {

	region, bucket, key := s3parse(path)

	svc := s3client(region)
	if svc == nil {
		return "", fmt.Errorf("s3fileFirstLine: missing s3 client")
	}

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

	region, bucket, prefix := s3parse(path)

	svc := s3client(region)
	if svc == nil {
		return dirname, names, fmt.Errorf("s3dirList: missing s3 client")
	}

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

	region, bucket, prefix := s3parse(path)

	svc := s3client(region)
	if svc == nil {
		return fmt.Errorf("s3dirClean: missing s3 client")
	}

	// build object list
	folder := filepath.Dir(prefix)
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

	region, bucket, key := s3parse(path)

	svc := s3client(region)
	if svc == nil {
		return time.Time{}, 0, fmt.Errorf("s3fileInfo: missing s3 client")
	}

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

func s3fileCompare(p1, p2 string, maxSize int64) (bool, error) {
	r1, err1 := s3fileReader(p1)
	if err1 != nil {
		return false, err1
	}
	defer r1.Close()

	r2, err2 := s3fileReader(p2)
	if err2 != nil {
		return false, err2
	}
	defer r2.Close()

	buf := make([]byte, 100000)
	cmp := equalfile.New(buf, equalfile.Options{MaxSize: maxSize})

	return cmp.CompareReader(r1, r2)
}

// S3URL builds the URL for an S3 bucket.
// The path is an ARN: "arn:aws:s3:region::bucket/folder/file.xxx"
func S3URL(path string) string {
	region, bucket, key := s3parse(path)

	if region == "" {
		region = s3region // fallback to default region
	}
	if region == "" {
		s3log("S3URL: could not find region: [%s]", path)
		return ""
	}

	return fmt.Sprintf("https://s3-%s.amazonaws.com/%s/%s", region, bucket, key)
}
