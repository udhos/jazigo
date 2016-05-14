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

	l.output, _ = l.open(outputPath)

	return l
}

func (l *logfile) open(path string) (*os.File, error) {
	output, openErr := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
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

	outputPath, newErr := store.SaveNewConfig(l.logPathPrefix, l.maxFiles, l.logger, touchFunc)
	if newErr != nil {
		if l.output != nil {
			l.output.Close()
			l.output = nil
		}
		l.logger.Printf("logfile.rotate: could not open log: %v", newErr)
		return
	}

	l.output, _ = l.open(outputPath)
}

func (l *logfile) Write(b []byte) (int, error) {

	if l.output == nil {
		l.rotate()
		if l.output == nil {
			return 0, fmt.Errorf("log: could not create output file")
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
			}
		}
	}

	return l.output.Write(b)
}
