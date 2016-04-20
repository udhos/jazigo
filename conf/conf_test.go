package conf

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/udhos/jazigo/temp"
)

// testLogger: wrap Printf interface around *testing.T
type testLogger struct {
	*testing.T
}

func (t *testLogger) Printf(format string, v ...interface{}) {
	t.Logf("client: "+format, v...)
}

func confWrite(t *testing.T, prefix, content, expected string, maxFiles int) error {

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

	logger := &testLogger{t}

	path, writeErr := SaveNewConfig(prefix, maxFiles, logger, writeFunc)
	if writeErr != nil {
		return fmt.Errorf("confWrite: error: %v", writeErr)
	}

	if path != expected {
		return fmt.Errorf("confWrite: got=%s wanted=%s", path, expected)
	}

	found, findErr := FindLastConfig(prefix, logger)
	if findErr != nil {
		return fmt.Errorf("confWrite: FindLastConfig: error: %v", findErr)
	}

	if found != expected {
		return fmt.Errorf("confWrite: FindLastConfig: found=%s wanted=%s", found, expected)
	}

	return nil
}

func TestConf1(t *testing.T) {

	repo := temp.TempRepo()
	defer temp.CleanupTempRepo()

	prefix := filepath.Join(repo, "conf-test.")
	maxFiles := 2

	if err := confWrite(t, prefix, "a", fmt.Sprintf("%s0", prefix), maxFiles); err != nil {
		t.Errorf("TestConf1: %v", err)
	}

	if err := confWrite(t, prefix, "b", fmt.Sprintf("%s1", prefix), maxFiles); err != nil {
		t.Errorf("TestConf1: %v", err)
	}

	if err := confWrite(t, prefix, "c", fmt.Sprintf("%s2", prefix), maxFiles); err != nil {
		t.Errorf("TestConf1: %v", err)
	}

	if err := confWrite(t, prefix, "d", fmt.Sprintf("%s3", prefix), maxFiles); err != nil {
		t.Errorf("TestConf1: %v", err)
	}

}
