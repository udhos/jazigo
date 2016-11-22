package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/udhos/jazigo/store"
)

type logfile struct {
	logPathPrefix     string
	maxFiles          int
	maxFileSize       int64
	sizeCheckInterval time.Duration
	lastSizeCheck     time.Time
	output            *os.File
	logger            *log.Logger
}

// NewLogfile creates a new log stream capable of automatically saving to filesystem.
func NewLogfile(prefix string, maxFiles int, maxSize int64, checkInterval time.Duration) *logfile {
	l := &logfile{
		logPathPrefix:     prefix,
		maxFiles:          maxFiles,
		maxFileSize:       maxSize,
		sizeCheckInterval: checkInterval,
		logger:            log.New(os.Stderr, "logfile stderr: ", log.LstdFlags),
	}

	outputPath, lastErr := store.FindLastConfig(l.logPathPrefix, l.logger)
	if lastErr != nil {
		return l
	}

	l.output, _ = openAppend(outputPath)

	return l
}

func openAppend(path string) (*os.File, error) {
	output, openErr := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0640)
	if openErr != nil {
		if output != nil {
			output.Close()
			output = nil
		}
	}
	return output, openErr
}

func touchFunc(w store.HasWrite) error {
	header := time.Now().String() + " - new log file\n"
	_, wrErr := w.Write([]byte(header))
	return wrErr
}

func (l *logfile) rotate() {
	if l.output != nil {
		l.output.Close()
		l.output = nil
	}

	outputPath, newErr := store.SaveNewConfig(l.logPathPrefix, l.maxFiles, l.logger, touchFunc, false)
	if newErr != nil {
		if l.output != nil {
			l.output.Close()
			l.output = nil
		}
		l.logger.Printf("logfile.rotate: could not find log path: %v", newErr)
		return
	}

	var openErr error
	l.output, openErr = openAppend(outputPath)
	if openErr != nil {
		l.logger.Printf("logfile.rotate: could not open log: %v", openErr)
	}
}

// Write implements io.Writer in order to be attached to log.New().
func (l *logfile) Write(b []byte) (int, error) {

	if l.output == nil {
		l.rotate()
		if l.output == nil {
			msg := "log: missing output - could not create output file"
			l.logger.Printf(msg)
			return 0, fmt.Errorf(msg)
		}
	}

	if time.Since(l.lastSizeCheck) > l.sizeCheckInterval {
		l.logger.Printf("log: checking file size")
		l.lastSizeCheck = time.Now()
		info, statErr := l.output.Stat()
		if statErr == nil {
			if info.Size() > l.maxFileSize {
				l.logger.Printf("log: max file size reached")
				l.rotate()
				if l.output == nil {
					msg := "log: rotate failure - could not create output file"
					l.logger.Printf(msg)
					return 0, fmt.Errorf(msg)
				}
			}
		}
	}

	return l.output.Write(b)
}
