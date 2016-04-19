package temp

import (
	"fmt"
	"os"
)

const TEMP_REPO = "/tmp/tmp-jazigo-repo"

func TempRepo() string {
	path := TEMP_REPO
	if err := os.MkdirAll(path, 0700); err != nil {
		panic(fmt.Sprintf("TempRepo: '%s': %v", path, err))
	}
	return path
}

func CleanupTempRepo() string {
	path := TEMP_REPO
	if err := os.RemoveAll(path); err != nil {
		panic(fmt.Sprintf("CleanupTempRepo: '%s': %v", path, err))
	}
	return path
}
