package dev

import (
	"testing"
	"time"

	"github.com/udhos/jazigo/conf"
	"github.com/udhos/jazigo/temp"
)

func TestOldHTTP1(t *testing.T) {

	t.Logf("TestOldHTTP1: starting")

	// launch bogus test server
	addr := ":2001"
	s, listenErr := spawnServerHTTP(t, addr)
	if listenErr != nil {
		t.Errorf("could not spawn bogus HTTP server: %v", listenErr)
	}
	t.Logf("TestOldHTTP1: server running on %s", addr)

	// run client test
	logger := &testLogger{t}
	tab := NewDeviceTable()
	opt := &conf.AppConfig{MaxConcurrency: 3, MaxConfigFiles: 10}
	RegisterModels(logger, tab)
	CreateDevice(tab, logger, "http", "lab1", "localhost"+addr, "", "", "", "", false)

	repo := temp.TempRepo()
	defer temp.CleanupTempRepo()

	good, bad, skip := ScanDevices(tab, tab.ListDevices(), logger, 100*time.Millisecond, 200*time.Millisecond, repo, opt)
	if good != 1 || bad != 0 || skip != 0 {
		t.Errorf("good=%d bad=%d skip=%d", good, bad, skip)
	}

	s.close()

	<-s.done // wait termination of accept loop goroutine
}
