package dev

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// ErrlogPath builds the full pathname for errlog file.
func ErrlogPath(pathPrefix, id string) string {
	dir := filepath.Dir(pathPrefix)
	path := filepath.Join(dir, id) + ".errlog"
	return path
}

func errlog(logger hasPrintf, result FetchResult, pathPrefix string, debug bool, histSize int) {

	now := time.Now()

	path := ErrlogPath(pathPrefix, result.DevID)

	f, openErr := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0640)
	if openErr != nil {
		logger.Printf("errlog: could not open dev log: '%s': %v", path, openErr)
		return
	}

	defer f.Close()

	// load lines
	lines, lineErr := loadLines(bufio.NewReader(f), histSize-1)
	if lineErr != nil {
		logger.Printf("errlog: could not load lines: '%s': %v", path, lineErr)
		return
	}

	if debug {
		logger.Printf("errlog debug: '%s': %d lines", path, len(lines))
		//logger.Printf("errlog debug: '%s': last line: [%s]", path, lines[len(lines)-1])
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
	msg := fmt.Sprintf("%s success=%v elapsed=%v model=%s dev=%s host=%s transport=%s code=%d message=[%s]",
		now.String(),
		result.Code == fetchErrNone,
		result.End.Sub(result.Begin),
		result.Model, result.DevID, result.DevHostPort, result.Transport, result.Code, result.Msg)

	if debug {
		logger.Printf("errlog debug: push: '%s': [%s]", path, msg)
	}

	_, pushErr := w.WriteString(msg + "\n")
	if pushErr != nil {
		logger.Printf("errlog: push error: '%s': %v", path, pushErr)
		return
	}

	// write lines back to file
	for _, line := range lines {
		_, writeErr := w.Write(line)
		if writeErr != nil {
			logger.Printf("errlog: write error: '%s': %v", path, writeErr)
			break
		}
	}

	if flushErr := w.Flush(); flushErr != nil {
		logger.Printf("errlog: flush: '%s': %v", path, flushErr)
	}

	if syncErr := f.Sync(); syncErr != nil {
		logger.Printf("errlog: sync: '%s': %v", path, syncErr)
	}
}

func loadLines(r *bufio.Reader, max int) ([][]byte, error) {
	var lines [][]byte

LOOP:
	for lineCount := 0; lineCount < max; lineCount++ {
		line, readErr := r.ReadBytes(LF)
		if len(line) > 0 {
			lines = append(lines, line)
		}
		switch readErr {
		case io.EOF:
			break LOOP
		case nil:
			continue
		default:
			return lines, readErr
		}
	}

	return lines, nil
}
