package dev

import (
	"fmt"
	"testing"
	"time"

	"github.com/udhos/jazigo/conf"
	"github.com/udhos/jazigo/temp"
)

func TestOldCiscoIOS1(t *testing.T) {

	// launch bogus test server
	addr := ":2001"
	s, listenErr := spawnServerCiscoIOS(t, addr, optionsCiscoIOS{sendUsername: true, sendDisable: true, requestEnablePass: true})
	if listenErr != nil {
		t.Errorf("could not spawn bogus CiscoIOS server: %v", listenErr)
	}
	t.Logf("TestOldCiscoIOS: server running on %s", addr)

	// run client test
	logger := &testLogger{t}
	tab := NewDeviceTable()
	opt := &conf.AppConfig{MaxConcurrency: 3, MaxConfigFiles: 10}
	RegisterModels(logger, tab)
	CreateDevice(tab, logger, "cisco-ios", "lab1", "localhost"+addr, "telnet", "lab", "pass", "en", false, nil)

	repo := temp.TempRepo()
	defer temp.CleanupTempRepo()

	good, bad, skip := ScanDevices(tab, tab.ListDevices(), logger, 100*time.Millisecond, 200*time.Millisecond, repo, opt, NewFilterTable(logger))
	if good != 1 || bad != 0 || skip != 0 {
		t.Errorf("good=%d bad=%d skip=%d", good, bad, skip)
	}

	//time.Sleep(time.Hour)

	// shutdown server
	s.close()

	<-s.done // wait termination of accept loop goroutine
}

func TestOldCiscoIOS2(t *testing.T) {

	// launch bogus test server
	addr := ":2002"
	s, listenErr := spawnServerCiscoIOS(t, addr, optionsCiscoIOS{sendUsername: false})
	if listenErr != nil {
		t.Errorf("could not spawn bogus CiscoIOS server: %v", listenErr)
	}
	t.Logf("TestOldCiscoIOS: server running on %s", addr)

	// run client test
	logger := &testLogger{t}
	tab := NewDeviceTable()
	opt := &conf.AppConfig{MaxConcurrency: 3, MaxConfigFiles: 10}
	RegisterModels(logger, tab)
	CreateDevice(tab, logger, "cisco-ios", "lab1", "localhost"+addr, "telnet", "lab", "pass", "en", false, nil)

	repo := temp.TempRepo()
	defer temp.CleanupTempRepo()

	good, bad, skip := ScanDevices(tab, tab.ListDevices(), logger, 100*time.Millisecond, 200*time.Millisecond, repo, opt, NewFilterTable(logger))
	if good != 1 || bad != 0 || skip != 0 {
		t.Errorf("good=%d bad=%d skip=%d", good, bad, skip)
	}

	//time.Sleep(time.Hour)

	// shutdown server
	s.close()

	<-s.done // wait termination of accept loop goroutine
}

func TestOldCiscoIOS3(t *testing.T) {

	// launch bogus test server
	addr := ":2003"
	s, listenErr := spawnServerCiscoIOS(t, addr, optionsCiscoIOS{sendUsername: true, sendDisable: true, requestEnablePass: true, breakConn: true})
	if listenErr != nil {
		t.Errorf("could not spawn bogus CiscoIOS server: %v", listenErr)
	}

	// run client test
	logger := &testLogger{t}
	tab := NewDeviceTable()
	opt := &conf.AppConfig{MaxConcurrency: 3, MaxConfigFiles: 10}
	RegisterModels(logger, tab)
	CreateDevice(tab, logger, "cisco-ios", "lab1", "localhost"+addr, "telnet", "lab", "pass", "en", false, nil)

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

func TestOldCiscoIOS4(t *testing.T) {

	// launch bogus test server
	addr := ":2004"
	s, listenErr := spawnServerCiscoIOS(t, addr, optionsCiscoIOS{sendUsername: false})
	if listenErr != nil {
		t.Errorf("could not spawn bogus CiscoIOS server: %v", listenErr)
	}
	t.Logf("TestOldCiscoIOS: server running on %s", addr)

	jobs := 100
	devices := 10 * jobs

	// run client test
	logger := &testLogger{t}
	tab := NewDeviceTable()
	opt := &conf.AppConfig{MaxConcurrency: jobs, MaxConfigFiles: 10}
	RegisterModels(logger, tab)
	for i := 0; i < devices; i++ {
		CreateDevice(tab, logger, "cisco-ios", fmt.Sprintf("lab%02d", i), "localhost"+addr, "telnet", "lab", "pass", "en", false, nil)
	}

	repo := temp.TempRepo()
	defer temp.CleanupTempRepo()

	good, bad, skip := ScanDevices(tab, tab.ListDevices(), logger, 0*time.Millisecond, 0*time.Millisecond, repo, opt, NewFilterTable(logger))
	if good != 1000 || bad != 0 || skip != 0 {
		t.Errorf("good=%d bad=%d", good, bad)
	}

	//time.Sleep(time.Hour)

	// shutdown server
	s.close()

	<-s.done // wait termination of accept loop goroutine
}
