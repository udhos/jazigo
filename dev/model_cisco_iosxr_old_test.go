package dev

import (
	"testing"
	"time"

	"github.com/udhos/jazigo/conf"
	"github.com/udhos/jazigo/temp"
)

func TestOldCiscoIOSXR1(t *testing.T) {

	// launch bogus test server
	addr := ":2001"
	s, listenErr := spawnServerCiscoIOSXR(t, addr, optionsCiscoIOSXR{sendUsername: true, sendDisable: true, requestEnablePass: true})
	if listenErr != nil {
		t.Errorf("could not spawn bogus CiscoIOSXR server: %v", listenErr)
	}
	t.Logf("TestOldCiscoIOSXR: server running on %s", addr)

	// run client test
	logger := &testLogger{t}
	tab := NewDeviceTable()
	opt := &conf.AppConfig{MaxConcurrency: 3, MaxConfigFiles: 10}
	RegisterModels(logger, tab)
	CreateDevice(tab, logger, "cisco-iosxr", "lab1", "localhost"+addr, "telnet", "lab", "pass", "en", false, nil)

	repo := temp.TempRepo()
	defer temp.CleanupTempRepo()

	good, bad, skip := ScanDevices(tab, tab.ListDevices(), logger, 100*time.Millisecond, 200*time.Millisecond, repo, opt, NewFilterTable(logger))
	if good != 1 || bad != 0 || skip != 0 {
		t.Errorf("good=%d bad=%d skip=%d", good, bad, skip)
	}

	// shutdown server
	s.close()

	<-s.done // wait termination of accept loop goroutine
}

func TestOldCiscoIOSXR2(t *testing.T) {

	// launch bogus test server
	addr := ":2002"
	s, listenErr := spawnServerCiscoIOSXR(t, addr, optionsCiscoIOSXR{sendUsername: false})
	if listenErr != nil {
		t.Errorf("could not spawn bogus CiscoIOSXR server: %v", listenErr)
	}
	t.Logf("TestOldCiscoIOSXR: server running on %s", addr)

	// run client test
	logger := &testLogger{t}
	tab := NewDeviceTable()
	opt := &conf.AppConfig{MaxConcurrency: 3, MaxConfigFiles: 10}
	RegisterModels(logger, tab)
	CreateDevice(tab, logger, "cisco-iosxr", "lab1", "localhost"+addr, "telnet", "lab", "pass", "en", false, nil)

	repo := temp.TempRepo()
	defer temp.CleanupTempRepo()

	good, bad, skip := ScanDevices(tab, tab.ListDevices(), logger, 100*time.Millisecond, 200*time.Millisecond, repo, opt, NewFilterTable(logger))
	if good != 1 || bad != 0 || skip != 0 {
		t.Errorf("good=%d bad=%d skip=%d", good, bad, skip)
	}

	// shutdown server
	s.close()

	<-s.done // wait termination of accept loop goroutine
}

func TestOldCiscoIOSXR3(t *testing.T) {

	// launch bogus test server
	addr := ":2003"
	s, listenErr := spawnServerCiscoIOSXR(t, addr, optionsCiscoIOSXR{sendUsername: true, sendDisable: true, requestEnablePass: true, breakConn: true})
	if listenErr != nil {
		t.Errorf("could not spawn bogus CiscoIOSXR server: %v", listenErr)
	}

	// run client test
	logger := &testLogger{t}
	tab := NewDeviceTable()
	opt := &conf.AppConfig{MaxConcurrency: 3, MaxConfigFiles: 10}
	RegisterModels(logger, tab)
	CreateDevice(tab, logger, "cisco-iosxr", "lab1", "localhost"+addr, "telnet", "lab", "pass", "en", false, nil)

	repo := temp.TempRepo()
	defer temp.CleanupTempRepo()

	good, bad, skip := ScanDevices(tab, tab.ListDevices(), logger, 100*time.Millisecond, 200*time.Millisecond, repo, opt, NewFilterTable(logger))
	if good != 0 || bad != 1 || skip != 0 {
		t.Errorf("good=%d bad=%d skip=%d", good, bad, skip)
	}

	// shutdown server
	s.close()

	<-s.done // wait termination of accept loop goroutine
}
