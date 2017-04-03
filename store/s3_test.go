package store

import (
	"testing"
)

func TestS3Parse(t *testing.T) {
	testS3Parse(t, "", "", "", "")
	testS3Parse(t, "arn:aws:s3:region::bucket/folder/file.xxx", "region", "bucket", "folder/file.xxx")
}

func testS3Parse(t *testing.T, input, region, bucket, key string) {
	r, b, k := s3parse(input)
	if r != region {
		t.Errorf("testS3Parse: input=[%s] region expected=[%s] got=[%s]", input, region, r)
	}
	if b != bucket {
		t.Errorf("testS3Parse: input=[%s] bucket expected=[%s] got=[%s]", input, bucket, b)
	}
	if k != key {
		t.Errorf("testS3Parse: input=[%s] key expected=[%s] got=[%s]", input, key, k)
	}
}
