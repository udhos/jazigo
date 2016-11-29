package store

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/udhos/jazigo/temp"
)

// testLogger: wrap Printf interface around *testing.T
type testLogger struct {
	*testing.T
}

func (t *testLogger) Printf(format string, v ...interface{}) {
	t.Logf("store testLogger: "+format, v...)
}

func TestStore1(t *testing.T) {

	repo := temp.MakeTempRepo()
	defer temp.CleanupTempRepo()

	region := os.Getenv("JAZIGO_S3_REGION")

	maxFiles := 2
	logger := &testLogger{t}
	Init(logger, region)

	prefix := filepath.Join(repo, "store-test.")
	storeBatch(t, prefix, maxFiles, logger)

	if region == "" {
		t.Logf("TestStore1: JAZIGO_S3_REGION undefined: skipping S3 tests")
		return
	}
	s3folder := os.Getenv("JAZIGO_S3_FOLDER")
	if s3folder == "" {
		t.Logf("TestStore1: JAZIGO_S3_FOLDER undefined: skipping S3 tests")
		return
	}

	prefix = fmt.Sprintf("arn:aws:s3:::%s/store-test.", s3folder)
	storeBatch(t, prefix, maxFiles, logger)
}

func storeBatch(t *testing.T, prefix string, maxFiles int, logger hasPrintf) {
	if err := storeWrite(t, prefix, "a", fmt.Sprintf("%s0", prefix), maxFiles, logger); err != nil {
		t.Errorf("TestStore1: %v", err)
	}

	if err := storeWrite(t, prefix, "b", fmt.Sprintf("%s1", prefix), maxFiles, logger); err != nil {
		t.Errorf("TestStore1: %v", err)
	}

	if err := storeWrite(t, prefix, "c", fmt.Sprintf("%s2", prefix), maxFiles, logger); err != nil {
		t.Errorf("TestStore1: %v", err)
	}

	if err := storeWrite(t, prefix, "d", fmt.Sprintf("%s3", prefix), maxFiles, logger); err != nil {
		t.Errorf("TestStore1: %v", err)
	}
}

func storeWrite(t *testing.T, prefix, content, expected string, maxFiles int, logger hasPrintf) error {

	c := []byte(content)

	writeFunc := func(w HasWrite) error {
		n, writeErr := w.Write(c)
		if writeErr != nil {
			return fmt.Errorf("writeFunc: error: %v", writeErr)
		}
		if n != len(c) {
			return fmt.Errorf("writeFunc: partial: wrote=%d size=%d", n, len(c))
		}
		return nil
	}

	path, writeErr := SaveNewConfig(prefix, maxFiles, logger, writeFunc, false)
	if writeErr != nil {
		return fmt.Errorf("storeWrite: error: %v", writeErr)
	}

	if path != expected {
		return fmt.Errorf("storeWrite: got=%s wanted=%s", path, expected)
	}

	found, findErr := FindLastConfig(prefix, logger)
	if findErr != nil {
		return fmt.Errorf("storeWrite: FindLastConfig: error: %v", findErr)
	}

	if found != expected {
		return fmt.Errorf("storeWrite: FindLastConfig: found=%s wanted=%s", found, expected)
	}

	return nil
}
