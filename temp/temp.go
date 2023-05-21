// Package temp creates temporary repository.
package temp

import (
	"fmt"
	"os"
)

const tempRepoPrefix = "/tmp/tmp-jazigo-repo"

// MakeTempRepo creates the temporary repository path, for testing.
func MakeTempRepo() string {
	path := tempRepoPrefix
	if err := os.MkdirAll(path, 0700); err != nil {
		panic(fmt.Sprintf("MakeTempRepo: '%s': %v", path, err))
	}
	return path
}

// CleanupTempRepo erases the temporary repository path.
func CleanupTempRepo() string {
	path := tempRepoPrefix
	if err := os.RemoveAll(path); err != nil {
		panic(fmt.Sprintf("CleanupTempRepo: '%s': %v", path, err))
	}
	return path
}
