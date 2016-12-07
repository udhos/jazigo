package dev

import (
	"fmt"
	"io"
	"net"
	"strings"
	"testing"

	"github.com/udhos/jazigo/conf"
	"github.com/udhos/jazigo/temp"
)

type optionsCiscoIOSXR struct {
	sendUsername      bool
	sendDisable       bool
	requestEnablePass bool
	breakConn         bool
}

func TestCiscoIOSXR1(t *testing.T) {

	// launch bogus test server
	addr := ":2001"
	s, listenErr := spawnServerCiscoIOSXR(t, addr, optionsCiscoIOSXR{sendUsername: true, sendDisable: true, requestEnablePass: true})
	if listenErr != nil {
		t.Errorf("could not spawn bogus CiscoIOSXR server: %v", listenErr)
	}
	t.Logf("TestCiscoIOSXR: server running on %s", addr)

	// run client test
	logger := &testLogger{t}
	tab := NewDeviceTable()
	opt := conf.NewOptions()
	opt.Set(&conf.AppConfig{MaxConcurrency: 3, MaxConfigFiles: 10})
	RegisterModels(logger, tab)
	CreateDevice(tab, logger, "cisco-iosxr", "lab1", "localhost"+addr, "telnet", "lab", "pass", "en", false, nil)

	repo := temp.MakeTempRepo()
	defer temp.CleanupTempRepo()

	requestCh := make(chan FetchRequest)
	go Spawner(tab, logger, requestCh, repo, repo, opt, NewFilterTable(logger))
	good, bad, skip := Scan(tab, tab.ListDevices(), logger, opt.Get(), requestCh)
	if good != 1 || bad != 0 || skip != 0 {
		t.Errorf("good=%d bad=%d skip=%d", good, bad, skip)
	}

	close(requestCh) // shutdown Spawner - we might exit first though

	s.close() // shutdown server

	<-s.done // wait termination of accept loop goroutine
}

func TestCiscoIOSXR2(t *testing.T) {

	// launch bogus test server
	addr := ":2002"
	s, listenErr := spawnServerCiscoIOSXR(t, addr, optionsCiscoIOSXR{sendUsername: false})
	if listenErr != nil {
		t.Errorf("could not spawn bogus CiscoIOSXR server: %v", listenErr)
	}
	t.Logf("TestCiscoIOSXR: server running on %s", addr)

	// run client test
	logger := &testLogger{t}
	tab := NewDeviceTable()
	opt := conf.NewOptions()
	opt.Set(&conf.AppConfig{MaxConcurrency: 3, MaxConfigFiles: 10})
	RegisterModels(logger, tab)
	CreateDevice(tab, logger, "cisco-iosxr", "lab1", "localhost"+addr, "telnet", "lab", "pass", "en", false, nil)

	repo := temp.MakeTempRepo()
	defer temp.CleanupTempRepo()

	requestCh := make(chan FetchRequest)
	go Spawner(tab, logger, requestCh, repo, repo, opt, NewFilterTable(logger))
	good, bad, skip := Scan(tab, tab.ListDevices(), logger, opt.Get(), requestCh)
	if good != 1 || bad != 0 || skip != 0 {
		t.Errorf("good=%d bad=%d skip=%d", good, bad, skip)
	}

	close(requestCh) // shutdown Spawner - we might exit first though

	s.close() // shutdown server

	<-s.done // wait termination of accept loop goroutine
}

func TestCiscoIOSXR3(t *testing.T) {

	// launch bogus test server
	addr := ":2003"
	s, listenErr := spawnServerCiscoIOSXR(t, addr, optionsCiscoIOSXR{sendUsername: true, sendDisable: true, requestEnablePass: true, breakConn: true})
	if listenErr != nil {
		t.Errorf("could not spawn bogus CiscoIOSXR server: %v", listenErr)
	}

	// run client test
	logger := &testLogger{t}
	tab := NewDeviceTable()
	opt := conf.NewOptions()
	opt.Set(&conf.AppConfig{MaxConcurrency: 3, MaxConfigFiles: 10})
	RegisterModels(logger, tab)
	CreateDevice(tab, logger, "cisco-iosxr", "lab1", "localhost"+addr, "telnet", "lab", "pass", "en", false, nil)

	repo := temp.MakeTempRepo()
	defer temp.CleanupTempRepo()

	requestCh := make(chan FetchRequest)
	go Spawner(tab, logger, requestCh, repo, repo, opt, NewFilterTable(logger))
	good, bad, skip := Scan(tab, tab.ListDevices(), logger, opt.Get(), requestCh)
	if good != 0 || bad != 1 || skip != 0 {
		t.Errorf("good=%d bad=%d skip=%d", good, bad, skip)
	}

	close(requestCh) // shutdown Spawner - we might exit first though

	s.close() // shutdown server

	<-s.done // wait termination of accept loop goroutine
}

func spawnServerCiscoIOSXR(t *testing.T, addr string, options optionsCiscoIOSXR) (*testServer, error) {

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	s := &testServer{listener: ln, done: make(chan int)}

	go acceptLoopIOSXR(t, s, handleConnectionCiscoIOSXR, options)

	return s, nil
}

func acceptLoopIOSXR(t *testing.T, s *testServer, handler func(*testing.T, net.Conn, optionsCiscoIOSXR), options optionsCiscoIOSXR) {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			t.Logf("acceptLoopIOSXR: accept failure, exiting: %v", err)
			break
		}
		go handler(t, conn, options)
	}

	close(s.done)
}

func handleConnectionCiscoIOSXR(t *testing.T, c net.Conn, options optionsCiscoIOSXR) {
	defer c.Close()

	buf := make([]byte, 1000)

	if options.sendUsername {
		// send username prompt
		if _, err := c.Write([]byte("Bogus CiscoIOSXR server\n\nUser Access Verification\n\nUsername: ")); err != nil {
			t.Logf("handleConnectionCiscoIOSXR: send username prompt error: %v", err)
			return
		}

		// consume username
		if _, err := c.Read(buf); err != nil {
			t.Logf("handleConnectionCiscoIOSXR: read username error: %v", err)
			return
		}
	}

	// send password prompt
	if _, err := c.Write([]byte("\nPassword: ")); err != nil {
		t.Logf("handleConnectionCiscoIOSXR: send password prompt error: %v", err)
		return
	}

	// consume password
	if _, err := c.Read(buf); err != nil {
		t.Logf("handleConnectionCiscoIOSXR: read password error: %v", err)
		return
	}

	enabled := !options.sendDisable

LOOP:
	for {

		prompt := ">"
		if enabled {
			prompt = "#"
		}

		// send command prompt
		if _, err := c.Write([]byte(fmt.Sprintf("\nRP/0/RSP0/CPU0:asr9k%s", prompt))); err != nil {
			t.Logf("handleConnectionCiscoIOSXR: send command prompt error: %v", err)
			return
		}

		// consume command
		if _, err := c.Read(buf); err != nil {
			if err == io.EOF {
				return // peer closed connection
			}
			t.Logf("handleConnectionCiscoIOSXR: read command error: %v", err)
			return
		}

		str := string(buf)

		switch {
		case strings.HasPrefix(str, "q"): //quit
			break LOOP
		case strings.HasPrefix(str, "ex"): //exit
			break LOOP
		case strings.HasPrefix(str, "term"): //term len 0
		case strings.HasPrefix(str, "sh"): //sh run

			if options.breakConn {
				// break connection (on defer/exit)
				return
			}

			if _, err := c.Write([]byte("\nshow running-configuration\nthis is the IOS XR config\n")); err != nil {
				t.Logf("handleConnectionCiscoIOSXR: send sh run error: %v", err)
				return
			}
		case strings.HasPrefix(str, "en"): //enable
			if !enabled {
				// send password prompt
				if _, err := c.Write([]byte("\nPassword: ")); err != nil {
					t.Logf("handleConnectionCiscoIOSXR: send enable password prompt error: %v", err)
					return
				}

				// consume password
				if _, err := c.Read(buf); err != nil {
					t.Logf("handleConnectionCiscoIOSXR: read enable password error: %v", err)
					return
				}

				enabled = true
			}
		default:
			if _, err := c.Write([]byte("\nIgnoring unknown command")); err != nil {
				t.Logf("handleConnectionCiscoIOSXR: send unknown command error: %v", err)
				return
			}
		}

	}

	// send bye
	if _, err := c.Write([]byte("\nbye\n")); err != nil {
		t.Logf("handleConnectionCiscoIOSXR: send bye error: %v", err)
		return
	}

}
