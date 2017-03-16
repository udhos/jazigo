package store

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/udhos/equalfile"
)

type hasPrintf interface {
	Printf(fmt string, v ...interface{})
}

type sortByCommitID struct {
	data   []string
	logger hasPrintf
}

func (s sortByCommitID) Len() int {
	return len(s.data)
}
func (s sortByCommitID) Swap(i, j int) {
	s.data[i], s.data[j] = s.data[j], s.data[i]
}
func (s sortByCommitID) Less(i, j int) bool {
	s1 := s.data[i]
	id1, err1 := ExtractCommitIDFromFilename(s1)
	if err1 != nil {
		s.logger.Printf("sortByCommitID.Less: error parsing config file path: '%s': %v", s1, err1)
	}
	s2 := s.data[j]
	id2, err2 := ExtractCommitIDFromFilename(s2)
	if err2 != nil {
		s.logger.Printf("sortByCommitID.Less: error parsing config file path: '%s': %v", s2, err2)
	}
	return id1 < id2
}

// Init starts the store by providing a logger and default S3 region.
func Init(logger hasPrintf, region string) {
	if logger == nil {
		panic("store.Init: nil logger")
	}
	s3init(logger, region)
}

// ExtractCommitIDFromFilename gets the commit from a filename: "aaa.1" => 1
func ExtractCommitIDFromFilename(filename string) (int, error) {
	lastDot := strings.LastIndexByte(filename, '.')
	commitID := filename[lastDot+1:]
	id, err := strconv.Atoi(commitID)
	if err != nil {
		return -1, fmt.Errorf("extractCommitIDFromFilename: error parsing filename [%s]: %v", filename, err)
	}

	return id, nil
}

func fileFirstLine(path string) (string, error) {

	if s3path(path) {
		return s3fileFirstLine(path)
	}

	f, openErr := os.Open(path)
	if openErr != nil {
		return "", openErr
	}
	defer f.Close()

	r := bufio.NewReader(f)
	line, _, readErr := r.ReadLine()

	return string(line[:]), readErr
}

func tryShortcut(configPathPrefix string, logger hasPrintf) string {

	lastIDPath := getLastIDPath(configPathPrefix)
	id, err := fileFirstLine(lastIDPath)
	if err != nil {
		//logger.Printf("tryShortcut: [%s] error: %v", lastIDPath, err)
		return "" // not found
	}

	path := getConfigPath(configPathPrefix, id)
	if fileExists(path) {
		return path // found
	}

	return "" // not found
}

// FindLastConfig finds the last file under a path prefix.
func FindLastConfig(configPathPrefix string, logger hasPrintf) (string, error) {

	if path := tryShortcut(configPathPrefix, logger); path != "" {
		return path, nil // found
	}
	logger.Printf("FindLastConfig: not found from shortcut: [%s]", configPathPrefix)

	// search filesystem directory

	dirname, matches, err := ListConfig(configPathPrefix, logger)
	if err != nil {
		return "", err
	}

	size := len(matches)

	logger.Printf("FindLastConfig: found %d matching files: %v", size, matches)

	if size < 1 {
		return "", fmt.Errorf("FindLastConfig: no config file found for prefix: %s", configPathPrefix)
	}

	maxID := -1
	last := ""
	for _, m := range matches {
		id, idErr := ExtractCommitIDFromFilename(m)
		if idErr != nil {
			return "", fmt.Errorf("FindLastConfig: bad commit id: %s: %v", m, idErr)
		}
		if id >= maxID {
			maxID = id
			last = m
		}
	}

	lastPath := filepath.Join(dirname, last)

	logger.Printf("FindLastConfig: found: %s", lastPath)

	return lastPath, nil
}

// ListConfigSorted retrieves the sorted list of files under a path prefix.
func ListConfigSorted(configPathPrefix string, reverse bool, logger hasPrintf) (string, []string, error) {

	dirname, matches, err := ListConfig(configPathPrefix, logger)
	if err != nil {
		return dirname, matches, err
	}

	if reverse {
		sort.Sort(sort.Reverse(sortByCommitID{data: matches, logger: logger}))
	} else {
		sort.Sort(sortByCommitID{data: matches, logger: logger})
	}

	return dirname, matches, nil
}

func dirList(path string) (string, []string, error) {

	if s3path(path) {
		return s3dirList(path)
	}

	dirname := filepath.Dir(path)

	dir, err := os.Open(dirname)
	if err != nil {
		return dirname, nil, fmt.Errorf("ListConfig: error opening dir '%s': %v", dirname, err)
	}

	defer dir.Close()

	names, err2 := dir.Readdirnames(0)
	if err2 != nil {
		return dirname, nil, fmt.Errorf("ListConfig: error reading dir '%s': %v", dirname, err2)
	}

	return dirname, names, nil
}

// ListConfig retrieves files under a path prefix.
func ListConfig(configPathPrefix string, logger hasPrintf) (string, []string, error) {

	var dirname string
	var names []string
	var dirErr error
	dirname, names, dirErr = dirList(configPathPrefix)
	if dirErr != nil {
		return dirname, nil, dirErr
	}

	logger.Printf("ListConfig: prefix=[%s] names=%d", configPathPrefix, len(names))

	basename := filepath.Base(configPathPrefix)

	// filter prefix
	matches := names[:0] // slice trick: Filtering without allocating
	for _, x := range names {
		lastByte := rune(x[len(x)-1])
		if unicode.IsDigit(lastByte) && strings.HasPrefix(x, basename) {
			matches = append(matches, x)
		}
	}

	return dirname, matches, nil
}

// HasWrite is a helper interface for types providing the method Write().
type HasWrite interface {
	Write(p []byte) (int, error)
}

func getLastIDPath(configPathPrefix string) string {
	return fmt.Sprintf("%slast", configPathPrefix)
}

func getConfigPath(configPathPrefix, id string) string {
	return fmt.Sprintf("%s%s", configPathPrefix, id)
}

func fileExists(path string) bool {

	if s3path(path) {
		return s3fileExists(path)
	}

	if _, err := os.Stat(path); err == nil {
		return true
	}

	return false
}

func fileRemove(path string) error {

	if s3path(path) {
		return s3fileRemove(path)
	}

	return os.Remove(path)
}

func fileRename(p1, p2 string) error {

	if s3path(p1) {
		return s3fileRename(p1, p2)
	}

	return os.Rename(p1, p2)
}

// FileRead reads bytes from file.
func FileRead(path string, maxSize int64) ([]byte, error) {

	var r *io.LimitedReader

	if s3path(path) {
		r1, readErr := s3fileReader(path)
		if readErr != nil {
			return nil, readErr
		}
		defer r1.Close()
		r = &io.LimitedReader{R: r1, N: maxSize}
	} else {
		f, openErr := os.Open(path)
		if openErr != nil {
			return nil, openErr
		}
		defer f.Close()
		r = &io.LimitedReader{R: f, N: maxSize}
	}

	buf, readErr := ioutil.ReadAll(r)
	if readErr != nil {
		return buf, readErr
	}

	if r.N < 1 {
		return buf, fmt.Errorf("FileRead: reached max=%d: remaining=%d", maxSize, r.N)
	}

	return buf, nil
}

func writeFileBuf(path string, buf []byte, contentType string) error {

	if s3path(path) {
		return s3fileput(path, buf, contentType)
	}

	return ioutil.WriteFile(path, buf, 0640)
}

func writeFile(path string, writeFunc func(HasWrite) error, contentType string) error {

	if s3path(path) {
		w := &bytes.Buffer{}

		if err := writeFunc(w); err != nil {
			return fmt.Errorf("SaveNewConfig: writeFunc error: [%s]: %v", path, err)
		}

		return s3fileput(path, w.Bytes(), contentType)
	}

	f, createErr := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0640)
	if createErr != nil {
		return fmt.Errorf("SaveNewConfig: error creating file: [%s]: %v", path, createErr)
	}

	w := bufio.NewWriter(f)

	if err := writeFunc(w); err != nil {
		return fmt.Errorf("SaveNewConfig: writeFunc error: [%s]: %v", path, err)
	}

	if err := w.Flush(); err != nil {
		return fmt.Errorf("SaveNewConfig: error flushing file: [%s]: %v", path, err)
	}

	if err := f.Close(); err != nil {
		return fmt.Errorf("SaveNewConfig: error closing file: [%s]: %v", path, err)
	}

	return nil
}

// SaveNewConfig saves data to a new file. The function writeFunc must be provided to issue the actual data.
func SaveNewConfig(configPathPrefix string, maxFiles int, logger hasPrintf, writeFunc func(HasWrite) error, changesOnly bool, contentType string) (string, error) {

	// get tmp file

	tmpPath := getConfigPath(configPathPrefix, "tmp")
	if fileExists(tmpPath) {
		return "", fmt.Errorf("SaveNewConfig: tmp file exists: [%s]", tmpPath)
	}

	// write to tmp file

	creatErr := writeFile(tmpPath, writeFunc, contentType)
	if creatErr != nil {
		return "", fmt.Errorf("SaveNewConfig: error creating tmp file: [%s]: %v", tmpPath, creatErr)
	}

	defer fileRemove(tmpPath)

	// get previous file

	previousFound := true
	lastConfig, err1 := FindLastConfig(configPathPrefix, logger)
	if err1 != nil {
		logger.Printf("SaveNewConfig: error reading config: [%s]: %v", configPathPrefix, err1)
		previousFound = false
	}

	id, err2 := ExtractCommitIDFromFilename(lastConfig)
	if err2 != nil {
		logger.Printf("SaveNewConfig: error parsing config path: [%s]: %v", lastConfig, err2)
	}

	if changesOnly && previousFound {
		equal, equalErr := fileCompare(lastConfig, tmpPath)
		if equalErr == nil {
			if equal {
				logger.Printf("SaveNewConfig: refusing to create identical new file: [%s]", tmpPath)
				if removeErr := fileRemove(tmpPath); removeErr != nil {
					logger.Printf("SaveNewConfig: error removing temp file=[%s]: %v", tmpPath, removeErr)
				}
				return lastConfig, nil // success
			}
			// unequal
			logger.Printf("SaveNewConfig: files differ previous=[%s] new=[%s]", lastConfig, tmpPath)
		} else {
			// unable to compare
			logger.Printf("SaveNewConfig: error comparing previous=[%s] to new=[%s]: %v", lastConfig, tmpPath, equalErr)
		}
	}

	// get new file

	newCommitID := id + 1
	newFilepath := getConfigPath(configPathPrefix, strconv.Itoa(newCommitID))

	logger.Printf("SaveNewConfig: newPath=[%s]", newFilepath)

	if fileExists(newFilepath) {
		return "", fmt.Errorf("SaveNewConfig: new file exists: [%s]", newFilepath)
	}

	// rename tmp to new file

	if renameErr := fileRename(tmpPath, newFilepath); renameErr != nil {
		return "", fmt.Errorf("SaveNewConfig: could not rename '%s' to '%s'; %v", tmpPath, newFilepath, renameErr)
	}

	// write shortcut file

	// write last id into shortcut file
	lastIDPath := getLastIDPath(configPathPrefix)
	if err := writeFileBuf(lastIDPath, []byte(strconv.Itoa(newCommitID)), contentType); err != nil {
		logger.Printf("SaveNewConfig: error writing last id file '%s': %v", lastIDPath, err)

		// since we failed to update the shortcut file,
		// it might be pointing to old backup.
		// then it's safer to simply remove it.
		fileRemove(lastIDPath)
	}

	// erase old file

	eraseOldFiles(configPathPrefix, maxFiles, logger)

	return newFilepath, nil
}

func eraseOldFiles(configPathPrefix string, maxFiles int, logger hasPrintf) {

	if maxFiles < 1 {
		return
	}

	dirname, matches, err := ListConfigSorted(configPathPrefix, false, logger)
	if err != nil {
		logger.Printf("eraseOldFiles: %v", err)
		return
	}

	totalFiles := len(matches)

	toDelete := totalFiles - maxFiles
	if toDelete < 1 {
		logger.Printf("eraseOldFiles: nothing to delete existing=%d <= max=%d", totalFiles, maxFiles)
		return
	}

	for i := 0; i < toDelete; i++ {
		path := filepath.Join(dirname, matches[i])
		logger.Printf("eraseOldFiles: delete: [%s]", path)
		if err := fileRemove(path); err != nil {
			logger.Printf("eraseOldFiles: delete: error: [%s]: %v", path, err)
		}
	}
}

// FileInfo returns file modification time and size.
func FileInfo(path string) (time.Time, int64, error) {

	if s3path(path) {
		return s3fileInfo(path)
	}

	info, statErr := os.Stat(path)
	if statErr != nil {
		return time.Time{}, 0, statErr
	}

	return info.ModTime(), info.Size(), nil
}

func fileCompare(p1, p2 string) (bool, error) {

	if s3path(p1) {
		maxSize := int64(10000000) // 10M FIXME??
		return s3fileCompare(p1, p2, maxSize)
	}

	cmp := equalfile.New(nil, equalfile.Options{})
	return cmp.CompareFile(p1, p2)
}

// MkDir creates a new directory.
func MkDir(path string) error {

	if s3path(path) {
		s3log("store.MkDir: silently refusing to create unneeded dir path on S3: [%s]", path)
		return nil
	}

	return os.MkdirAll(path, 0750)
}
