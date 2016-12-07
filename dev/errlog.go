package dev

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

func ErrlogPath(pathPrefix, id string) string {
	dir := filepath.Dir(pathPrefix)
	path := filepath.Join(dir, id) + ".errlog"
	return path
}

func errlog(logger hasPrintf, result FetchResult, pathPrefix string, debug bool) {

	now := time.Now()

	path := ErrlogPath(pathPrefix, result.DevId)

	f, openErr := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0640)
	if openErr != nil {
		logger.Printf("errlog: could not open dev log: '%s': %v", path, openErr)
		return
	}

	defer f.Close()

	// load lines
	lines, lineErr := loadLines(f, 9)
	if openErr != nil {
		logger.Printf("errlog: could not load lines: '%s': %v", path, lineErr)
		return
	}

	if debug {
		logger.Printf("errlog debug: '%s': %d lines", path, len(lines))
	}

	// truncate file
	if truncErr := f.Truncate(0); truncErr != nil {
		logger.Printf("errlog: truncate error: %v", truncErr)
		return
	}

	if _, seekErr := f.Seek(0, 0); seekErr != nil {
		logger.Printf("errlog: seek error: %v", seekErr)
		return
	}

	// push result
	w := bufio.NewWriter(f)
	msg := fmt.Sprintf("%s success=%v model=%s dev=%s host=%s transport=%s code=%d message=[%s]",
		now.String(),
		result.Code == fetchErrNone, result.Model, result.DevId, result.DevHostPort, result.Transport, result.Code, result.Msg)

	if debug {
		logger.Printf("errlog debug: push: '%s': %v", path, msg)
	}

	_, pushErr := w.WriteString(msg + "\n")
	if pushErr != nil {
		logger.Printf("errlog: push error: '%s': %v", path, pushErr)
		return
	}

	// write lines back to file
	for i, line := range lines {
		l := append(line, LF)
		_, writeErr := w.Write(l)
		if writeErr != nil {
			logger.Printf("errlog: write error: '%s': %v", path, writeErr)
			break
		}
		if debug {
			logger.Printf("errlog debug: wrote line=%d: '%s': %v", i, path, string(line))
		}
	}

	if flushErr := w.Flush(); flushErr != nil {
		logger.Printf("errlog: flush: '%s': %v", path, flushErr)
	}

	if syncErr := f.Sync(); syncErr != nil {
		logger.Printf("errlog: sync: '%s': %v", path, syncErr)
	}
}

func loadLines(r io.Reader, max int) ([][]byte, error) {
	scanner := bufio.NewScanner(r)
	var lines [][]byte
	for lineCount := 0; lineCount < max && scanner.Scan(); lineCount++ {
		lines = append(lines, scanner.Bytes())
	}
	return lines, scanner.Err()
}
