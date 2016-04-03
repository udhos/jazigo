package conf

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type hasPrintf interface {
	Printf(fmt string, v ...interface{})
}

type sortByCommitId struct {
	data   []string
	logger hasPrintf
}

func (s sortByCommitId) Len() int {
	return len(s.data)
}
func (s sortByCommitId) Swap(i, j int) {
	s.data[i], s.data[j] = s.data[j], s.data[i]
}
func (s sortByCommitId) Less(i, j int) bool {
	s1 := s.data[i]
	id1, err1 := ExtractCommitIdFromFilename(s1)
	if err1 != nil {
		s.logger.Printf("sortByCommitId.Less: error parsing config file path: '%s': %v", s1, err1)
	}
	s2 := s.data[j]
	id2, err2 := ExtractCommitIdFromFilename(s2)
	if err2 != nil {
		s.logger.Printf("sortByCommitId.Less: error parsing config file path: '%s': %v", s2, err2)
	}
	return id1 < id2
}

func ExtractCommitIdFromFilename(filename string) (int, error) {
	lastDot := strings.LastIndexByte(filename, '.')
	commitId := filename[lastDot+1:]
	id, err := strconv.Atoi(commitId)
	if err != nil {
		return -1, fmt.Errorf("extractCommitIdFromFilename: error parsing filename [%s]: %v", filename, err)
	}

	return id, nil
}

func FindLastConfig(configPathPrefix string, logger hasPrintf) (string, error) {

	dirname, matches, err := ListConfig(configPathPrefix, false, logger)
	if err != nil {
		return "", err
	}

	m := len(matches)

	logger.Printf("FindLastConfig: found %d matching files: %v", m, matches)

	if m < 1 {
		return "", fmt.Errorf("FindLastConfig: no config file found for prefix: %s", configPathPrefix)
	}

	lastConfig := filepath.Join(dirname, matches[m-1])

	return lastConfig, nil
}

func ListConfig(configPathPrefix string, reverse bool, logger hasPrintf) (string, []string, error) {

	dirname := filepath.Dir(configPathPrefix)

	dir, err := os.Open(dirname)
	if err != nil {
		return "", nil, fmt.Errorf("FindLastConfig: error opening dir '%s': %v", dirname, err)
	}

	names, e := dir.Readdirnames(0)
	if e != nil {
		return "", nil, fmt.Errorf("FindLastConfig: error reading dir '%s': %v", dirname, e)
	}

	dir.Close()

	basename := filepath.Base(configPathPrefix)

	// filter prefix
	matches := names[:0]
	for _, x := range names {
		if strings.HasPrefix(x, basename) {
			matches = append(matches, x)
		}
	}

	if reverse {
		sort.Sort(sort.Reverse(sortByCommitId{data: matches, logger: logger}))
	} else {
		sort.Sort(sortByCommitId{data: matches, logger: logger})
	}

	return dirname, matches, nil
}

func SaveNewConfig(configPathPrefix string, maxFiles int, logger hasPrintf) (string, error) {

	lastConfig, err1 := FindLastConfig(configPathPrefix, logger)
	if err1 != nil {
		logger.Printf("SaveNewConfig: error reading config: [%s]: %v", configPathPrefix, err1)
	}

	id, err2 := ExtractCommitIdFromFilename(lastConfig)
	if err2 != nil {
		logger.Printf("SaveNewConfig: error parsing config path: [%s]: %v", lastConfig, err2)
	}

	newCommitId := id + 1

	newFilepath := fmt.Sprintf("%s%d", configPathPrefix, newCommitId)

	logger.Printf("SaveNewConfig: newPath=[%s]", newFilepath)

	if _, err := os.Stat(newFilepath); err == nil {
		return "", fmt.Errorf("SaveNewConfig: new file exists: [%s]", newFilepath)
	}

	f, err3 := os.Create(newFilepath)
	if err3 != nil {
		return "", fmt.Errorf("SaveNewConfig: error creating file: [%s]: %v", newFilepath, err3)
	}

	w := bufio.NewWriter(f)

	/*
		cw := configLineWriter{w}

		if err := WriteConfig(root, &cw); err != nil {
			return "", fmt.Errorf("SaveNewConfig: error writing file: [%s]: %v", newFilepath, err)
		}
	*/

	if err := w.Flush(); err != nil {
		return "", fmt.Errorf("SaveNewConfig: error flushing file: [%s]: %v", newFilepath, err)
	}

	if err := f.Close(); err != nil {
		return "", fmt.Errorf("SaveNewConfig: error closing file: [%s]: %v", newFilepath, err)
	}

	eraseOldFiles(configPathPrefix, maxFiles, logger)

	return newFilepath, nil
}

func eraseOldFiles(configPathPrefix string, maxFiles int, logger hasPrintf) {

	if maxFiles < 1 {
		return
	}

	dirname, matches, err := ListConfig(configPathPrefix, false, logger)
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
		if err := os.Remove(path); err != nil {
			logger.Printf("eraseOldFiles: delete: error: [%s]: %v", path, err)
		}
	}
}
