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

type optionsFortiOS struct {
	requestPassword bool
	breakConn       bool
}

func TestFortiOS1(t *testing.T) {

	// launch bogus test server
	addr := ":2001"
	s, listenErr := spawnServerFortiOS(t, addr, optionsFortiOS{requestPassword: true})
	if listenErr != nil {
		t.Errorf("could not spawn bogus FortiOS server: %v", listenErr)
	}
	t.Logf("TestFortiOS: server running on %s", addr)

	// run client test
	logger := &testLogger{t}
	tab := NewDeviceTable()
	opt := conf.NewOptions()
	opt.Set(&conf.AppConfig{MaxConcurrency: 3, MaxConfigFiles: 10})
	RegisterModels(logger, tab)
	CreateDevice(tab, logger, "fortios", "lab1", "localhost"+addr, "telnet", "lab", "pass", "en", false, nil)

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

func TestFortiOS2(t *testing.T) {

	// launch bogus test server
	addr := ":2002"
	s, listenErr := spawnServerFortiOS(t, addr, optionsFortiOS{})
	if listenErr != nil {
		t.Errorf("could not spawn bogus FortiOS server: %v", listenErr)
	}
	t.Logf("TestFortiOS: server running on %s", addr)

	// run client test
	logger := &testLogger{t}
	tab := NewDeviceTable()
	opt := conf.NewOptions()
	opt.Set(&conf.AppConfig{MaxConcurrency: 3, MaxConfigFiles: 10})
	RegisterModels(logger, tab)
	CreateDevice(tab, logger, "fortios", "lab1", "localhost"+addr, "telnet", "lab", "pass", "en", false, nil)

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

func TestFortiOS3(t *testing.T) {

	// launch bogus test server
	addr := ":2003"
	s, listenErr := spawnServerFortiOS(t, addr, optionsFortiOS{breakConn: true})
	if listenErr != nil {
		t.Errorf("could not spawn bogus FortiOS server: %v", listenErr)
	}

	// run client test
	logger := &testLogger{t}
	tab := NewDeviceTable()
	opt := conf.NewOptions()
	opt.Set(&conf.AppConfig{MaxConcurrency: 3, MaxConfigFiles: 10})
	RegisterModels(logger, tab)
	CreateDevice(tab, logger, "fortios", "lab1", "localhost"+addr, "telnet", "lab", "pass", "en", false, nil)

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

func spawnServerFortiOS(t *testing.T, addr string, options optionsFortiOS) (*testServer, error) {

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	s := &testServer{listener: ln, done: make(chan int)}

	go acceptLoopFortiOS(t, s, handleConnectionFortiOS, options)

	return s, nil
}

func acceptLoopFortiOS(t *testing.T, s *testServer, handler func(*testing.T, net.Conn, optionsFortiOS), options optionsFortiOS) {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			t.Logf("acceptLoopFortiOS: accept failure, exiting: %v", err)
			break
		}
		go handler(t, conn, options)
	}

	close(s.done)
}

func handleConnectionFortiOS(t *testing.T, c net.Conn, options optionsFortiOS) {
	defer c.Close()

	buf := make([]byte, 1000)

	// send username prompt
	if _, err := c.Write([]byte("Bogus FortiOS server\n\nUser Access Verification\n\nlogin: ")); err != nil {
		t.Logf("handleConnectionFortiOS: send username prompt error: %v", err)
		return
	}

	// consume username
	if _, err := c.Read(buf); err != nil {
		t.Logf("handleConnectionFortiOS: read username error: %v", err)
		return
	}

	if options.requestPassword {
		// send password prompt
		if _, err := c.Write([]byte("\nPassword: ")); err != nil {
			t.Logf("handleConnectionFortiOS: send password prompt error: %v", err)
			return
		}

		// consume password
		if _, err := c.Read(buf); err != nil {
			t.Logf("handleConnectionFortiOS: read password error: %v", err)
			return
		}
	}

	config := false

LOOP:
	for {

		prompt := "hostname # "
		if config {
			prompt = "hostname (console) # "
		}

		// send command prompt
		if _, err := c.Write([]byte(fmt.Sprintf("\n%s", prompt))); err != nil {
			t.Logf("handleConnectionFortiOS: send command prompt error: %v", err)
			return
		}

		// consume command
		if _, err := c.Read(buf); err != nil {
			if err == io.EOF {
				return // peer closed connection
			}
			t.Logf("handleConnectionFortiOS: read command error: %v", err)
			return
		}

		str := string(buf)

		switch {
		case strings.HasPrefix(str, "q"): //quit
			break LOOP
		case strings.HasPrefix(str, "ex"): //exit
			break LOOP
		case strings.HasPrefix(str, "config system console"):
			config = true
		case strings.HasPrefix(str, "end"):
			config = false
		case config && strings.HasPrefix(str, "set output"):
			// paging
		case strings.HasPrefix(str, "show"): //show

			if options.breakConn {
				// break connection (on defer/exit)
				return
			}

			if _, err := c.Write([]byte("\nshow:\nthis is the full FortiOS config\nenjoy! ;-)\n")); err != nil {
				t.Logf("handleConnectionFortiOS: send sh run error: %v", err)
				return
			}
		default:
			if _, err := c.Write([]byte("\nIgnoring unknown command")); err != nil {
				t.Logf("handleConnectionFortiOS: send unknown command error: %v", err)
				return
			}
		}

	}

	// send bye
	if _, err := c.Write([]byte("\nbye\n")); err != nil {
		t.Logf("handleConnectionFortiOS: send bye error: %v", err)
		return
	}

}
