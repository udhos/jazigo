package store

import (
	"bytes"
	"fmt"
	"strings"

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
	}

	if s3Session == nil {
		s3Session = s3.New(awsSession, aws.NewConfig().WithRegion(s3region))
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
	s3log("initialized")
}

func s3log(format string, v ...interface{}) {
	if s3logger == nil {
		fmt.Printf("s3 store: "+format, v...)
		panic("s3 store uninitialized")
	}
	s3logger.Printf("s3 store: "+format, v...)
}

func s3path(path string) bool {
	s3match := strings.HasPrefix(path, "arn:aws:s3:")
	if s3match {
		s3log("s3path: [%s]", path)
	}
	return s3match
}

// "arn:aws:s3:::bucket/folder/file.xxx"
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
	s3log("s3parse: [%s] => bucket=[%s] key=[%s]", path, bucket, key)
	return bucket, key
}

func s3fileExists(path string) bool {

	s3c := s3client()
	if s3c == nil {
		s3log("s3fileExists: missing s3 client: ugh")
		return true // ugh
	}

	bucket, key := s3parse(path)

	params := &s3.HeadObjectInput{
		Bucket: aws.String(bucket), // Required
		Key:    aws.String(key),    // Required
	}
	_, err := s3c.HeadObject(params)
	if err == nil {
		s3log("s3fileExists: FOUND [%s]", path)
		return true // found
	}

	s3log("s3fileExists: [%s] error: %v", path, err)

	return false
}

func s3fileput(path string, buf []byte) error {

	s3c := s3client()
	if s3c == nil {
		return fmt.Errorf("s3fileput: missing s3 client")
	}

	bucket, key := s3parse(path)

	params := &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(buf),
	}
	_, err := s3c.PutObject(params)

	s3log("s3fileput: [%s] upload: error: %v", path, err)

	return err
}

func s3fileFirstLine(path string) (string, error) {
	return "", fmt.Errorf("s3fileFirstLine: FIXME WRITEME [%s]", path)
}

func s3fileRemove(path string) error {
	return fmt.Errorf("s3fileRemove: FIXME WRITEME [%s]", path)
}

func s3fileRename(p1, p2 string) error {
	return fmt.Errorf("s3fileRename: FIXME WRITEME [%s,%s]", p1, p2)
}

func s3dirList(path string) (string, []string, error) {
	return "", nil, fmt.Errorf("s3dirList: FIXME WRITEME [%s]", path)
}
