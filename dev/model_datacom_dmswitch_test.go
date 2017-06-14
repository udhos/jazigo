package dev

import (
	"fmt"
	"io"
	"net"
	"path/filepath"
	"strings"
	"testing"

	"github.com/udhos/jazigo/conf"
	"github.com/udhos/jazigo/temp"
)

type optionsDatacomDmswitch struct {
	breakConn  bool
	refuseAuth bool
}

func TestDatacomDmswitch1(t *testing.T) {

	// launch bogus test server
	addr := ":2001"
	s, listenErr := spawnServerDatacomDmswitch(t, addr, optionsDatacomDmswitch{})
	if listenErr != nil {
		t.Errorf("could not spawn bogus DatacomDmswitch server: %v", listenErr)
	}
	t.Logf("TestDatacomDmswitch: server running on %s", addr)

	// run client test
	logger := &testLogger{t}
	tab := NewDeviceTable()
	opt := conf.NewOptions()
	opt.Set(&conf.AppConfig{MaxConcurrency: 3, MaxConfigFiles: 10})
	RegisterModels(logger, tab)
	CreateDevice(tab, logger, "dmswitch", "lab1", "localhost"+addr, "telnet", "lab", "pass", "en", false, nil)

	repo := temp.MakeTempRepo()
	defer temp.CleanupTempRepo()

	requestCh := make(chan FetchRequest)
	errlogPrefix := filepath.Join(repo, "errlog_test.")
	go Spawner(tab, logger, requestCh, repo, errlogPrefix, opt, NewFilterTable(logger))
	good, bad, skip := Scan(tab, tab.ListDevices(), logger, opt.Get(), requestCh)
	if good != 1 || bad != 0 || skip != 0 {
		t.Errorf("good=%d bad=%d skip=%d", good, bad, skip)
	}

	close(requestCh) // shutdown Spawner - we might exit first though

	s.close() // shutdown server

	<-s.done // wait termination of accept loop goroutine
}

func TestDatacomDmswitch2(t *testing.T) {

	// launch bogus test server
	addr := ":2003"
	s, listenErr := spawnServerDatacomDmswitch(t, addr, optionsDatacomDmswitch{breakConn: true})
	if listenErr != nil {
		t.Errorf("could not spawn bogus DatacomDmswitch server: %v", listenErr)
	}

	// run client test
	debug := false
	logger := &testLogger{t}
	tab := NewDeviceTable()
	opt := conf.NewOptions()
	opt.Set(&conf.AppConfig{MaxConcurrency: 3, MaxConfigFiles: 10})
	RegisterModels(logger, tab)
	CreateDevice(tab, logger, "dmswitch", "lab1", "localhost"+addr, "telnet", "lab", "pass", "en", debug, nil)

	repo := temp.MakeTempRepo()
	defer temp.CleanupTempRepo()

	requestCh := make(chan FetchRequest)
	errlogPrefix := filepath.Join(repo, "errlog_test.")
	go Spawner(tab, logger, requestCh, repo, errlogPrefix, opt, NewFilterTable(logger))
	good, bad, skip := Scan(tab, tab.ListDevices(), logger, opt.Get(), requestCh)
	if good != 0 || bad != 1 || skip != 0 {
		t.Errorf("good=%d bad=%d skip=%d", good, bad, skip)
	}

	close(requestCh) // shutdown Spawner - we might exit first though

	s.close() // shutdown server

	<-s.done // wait termination of accept loop goroutine
}

func TestDatacomDmswitch3(t *testing.T) {

	// launch bogus test server
	addr := ":2004"
	s, listenErr := spawnServerDatacomDmswitch(t, addr, optionsDatacomDmswitch{refuseAuth: true})
	if listenErr != nil {
		t.Errorf("could not spawn bogus DatacomDmswitch server: %v", listenErr)
	}

	// run client test
	debug := false
	logger := &testLogger{t}
	tab := NewDeviceTable()
	opt := conf.NewOptions()
	opt.Set(&conf.AppConfig{MaxConcurrency: 3, MaxConfigFiles: 10})
	RegisterModels(logger, tab)
	CreateDevice(tab, logger, "dmswitch", "lab1", "localhost"+addr, "telnet", "lab", "pass", "en", debug, nil)

	repo := temp.MakeTempRepo()
	defer temp.CleanupTempRepo()

	requestCh := make(chan FetchRequest)
	errlogPrefix := filepath.Join(repo, "errlog_test.")
	go Spawner(tab, logger, requestCh, repo, errlogPrefix, opt, NewFilterTable(logger))
	good, bad, skip := Scan(tab, tab.ListDevices(), logger, opt.Get(), requestCh)
	if good != 0 || bad != 1 || skip != 0 {
		t.Errorf("good=%d bad=%d skip=%d", good, bad, skip)
	}

	close(requestCh) // shutdown Spawner - we might exit first though

	s.close() // shutdown server

	<-s.done // wait termination of accept loop goroutine
}

func spawnServerDatacomDmswitch(t *testing.T, addr string, options optionsDatacomDmswitch) (*testServer, error) {

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	s := &testServer{listener: ln, done: make(chan int)}

	go acceptLoopDatacomDmswitch(t, s, handleConnectionDatacomDmswitch, options)

	return s, nil
}

func acceptLoopDatacomDmswitch(t *testing.T, s *testServer, handler func(*testing.T, net.Conn, optionsDatacomDmswitch), options optionsDatacomDmswitch) {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			t.Logf("acceptLoopDatacomDmswitch: accept failure, exiting: %v", err)
			break
		}
		go handler(t, conn, options)
	}

	close(s.done)
}

func handleConnectionDatacomDmswitch(t *testing.T, c net.Conn, options optionsDatacomDmswitch) {
	defer c.Close()

	buf := make([]byte, 1000)

	// send banner
	if _, err := c.Write([]byte("Bogus DatacomDmswitch server\n\nLogin authentication")); err != nil {
		t.Logf("handleConnectionDatacomDmswitch: send banner error: %v", err)
		return
	}

	for {
		// send username prompt
		if _, err := c.Write([]byte("\n\ndmswitch login: ")); err != nil {
			t.Logf("handleConnectionDatacomDmswitch: send username prompt error: %v", err)
			return
		}

		// consume username
		if _, err := c.Read(buf); err != nil {
			t.Logf("handleConnectionDatacomDmswitch: read username error: %v", err)
			return
		}

		requestPass := true
		if requestPass {
			// send password prompt
			if _, err := c.Write([]byte("\nPassword: ")); err != nil {
				t.Logf("handleConnectionDatacomDmswitch: send password prompt error: %v", err)
				return
			}

			// consume password
			if _, err := c.Read(buf); err != nil {
				t.Logf("handleConnectionDatacomDmswitch: read password error: %v", err)
				return
			}
		}

		if options.refuseAuth {
			if _, err := c.Write([]byte("\r\nError: Authentication failed.\r\n  Logged Fail!\r\n  Please retry after 5 seconds.\r\n")); err != nil {
				t.Logf("handleConnectionDatacomDmswitch: send auth refusal error: %v", err)
				return
			}

			continue // repeat authentication
		}

		break // accept authentication
	}

	config := false

LOOP:
	for {

		hostname := "dmswitch"
		prompt := hostname + "#"
		if config {
			prompt = hostname + "(config)#"
		}

		// send command prompt
		if _, err := c.Write([]byte(fmt.Sprintf("\n%s", prompt))); err != nil {
			t.Logf("handleConnectionDatacomDmswitch: send command prompt error: %v", err)
			return
		}

		// consume command
		if _, err := c.Read(buf); err != nil {
			if err == io.EOF {
				return // peer closed connection
			}
			t.Logf("handleConnectionDatacomDmswitch: read command error: %v", err)
			return
		}

		str := string(buf)

		switch {
		case strings.HasPrefix(str, "exit"): //quit
			switch {
			case config:
				config = false
			default:
				break LOOP
			}
		case strings.HasPrefix(str, "conf"):
			config = true
		case strings.HasPrefix(str, "term"):
			// set paging
		case strings.HasPrefix(str, "no term"):
			// set paging
		case strings.HasPrefix(str, "sh"): // accept any show command

			if options.breakConn {
				// break connection (on defer/exit)
				t.Logf("handleConnectionDatacomDmswitch: breaking connection")
				return
			}

			if _, err := c.Write([]byte("\nshow:\nthis is the full DatacomDmswitch config\nenjoy! ;-)\n")); err != nil {
				t.Logf("handleConnectionDatacomDmswitch: send sh run error: %v", err)
				return
			}
		default:
			if _, err := c.Write([]byte("\nIgnoring unknown command")); err != nil {
				t.Logf("handleConnectionDatacomDmswitch: send unknown command error: %v", err)
				return
			}
		}

	}

	// send bye
	if _, err := c.Write([]byte("\nbye\n")); err != nil {
		t.Logf("handleConnectionDatacomDmswitch: send bye error: %v", err)
		return
	}

}
