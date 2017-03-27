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

type optionsHuaweiVRP struct {
	requestPassword bool
	breakConn       bool
}

func TestHuaweiVRP1(t *testing.T) {

	// launch bogus test server
	addr := ":2001"
	s, listenErr := spawnServerHuaweiVRP(t, addr, optionsHuaweiVRP{requestPassword: true})
	if listenErr != nil {
		t.Errorf("could not spawn bogus HuaweiVRP server: %v", listenErr)
	}
	t.Logf("TestHuaweiVRP: server running on %s", addr)

	// run client test
	logger := &testLogger{t}
	tab := NewDeviceTable()
	opt := conf.NewOptions()
	opt.Set(&conf.AppConfig{MaxConcurrency: 3, MaxConfigFiles: 10})
	RegisterModels(logger, tab)
	CreateDevice(tab, logger, "huawei-vrp", "lab1", "localhost"+addr, "telnet", "lab", "pass", "en", false, nil)

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

func TestHuaweiVRP2(t *testing.T) {

	// launch bogus test server
	addr := ":2002"
	s, listenErr := spawnServerHuaweiVRP(t, addr, optionsHuaweiVRP{})
	if listenErr != nil {
		t.Errorf("could not spawn bogus HuaweiVRP server: %v", listenErr)
	}
	t.Logf("TestHuaweiVRP: server running on %s", addr)

	// run client test
	logger := &testLogger{t}
	tab := NewDeviceTable()
	opt := conf.NewOptions()
	opt.Set(&conf.AppConfig{MaxConcurrency: 3, MaxConfigFiles: 10})
	RegisterModels(logger, tab)
	CreateDevice(tab, logger, "huawei-vrp", "lab1", "localhost"+addr, "telnet", "lab", "pass", "en", false, nil)

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

func TestHuaweiVRP3(t *testing.T) {

	// launch bogus test server
	addr := ":2003"
	s, listenErr := spawnServerHuaweiVRP(t, addr, optionsHuaweiVRP{breakConn: true})
	if listenErr != nil {
		t.Errorf("could not spawn bogus HuaweiVRP server: %v", listenErr)
	}

	// run client test
	logger := &testLogger{t}
	tab := NewDeviceTable()
	opt := conf.NewOptions()
	opt.Set(&conf.AppConfig{MaxConcurrency: 3, MaxConfigFiles: 10})
	RegisterModels(logger, tab)
	CreateDevice(tab, logger, "huawei-vrp", "lab1", "localhost"+addr, "telnet", "lab", "pass", "en", false, nil)

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

func spawnServerHuaweiVRP(t *testing.T, addr string, options optionsHuaweiVRP) (*testServer, error) {

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	s := &testServer{listener: ln, done: make(chan int)}

	go acceptLoopHuaweiVRP(t, s, handleConnectionHuaweiVRP, options)

	return s, nil
}

func acceptLoopHuaweiVRP(t *testing.T, s *testServer, handler func(*testing.T, net.Conn, optionsHuaweiVRP), options optionsHuaweiVRP) {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			t.Logf("acceptLoopHuaweiVRP: accept failure, exiting: %v", err)
			break
		}
		go handler(t, conn, options)
	}

	close(s.done)
}

func handleConnectionHuaweiVRP(t *testing.T, c net.Conn, options optionsHuaweiVRP) {
	defer c.Close()

	buf := make([]byte, 1000)

	// send username prompt
	if _, err := c.Write([]byte("Bogus HuaweiVRP server\n\nLogin authentication\n\nUsername:")); err != nil {
		t.Logf("handleConnectionHuaweiVRP: send username prompt error: %v", err)
		return
	}

	// consume username
	if _, err := c.Read(buf); err != nil {
		t.Logf("handleConnectionHuaweiVRP: read username error: %v", err)
		return
	}

	if options.requestPassword {
		// send password prompt
		if _, err := c.Write([]byte("\nPassword:")); err != nil {
			t.Logf("handleConnectionHuaweiVRP: send password prompt error: %v", err)
			return
		}

		// consume password
		if _, err := c.Read(buf); err != nil {
			t.Logf("handleConnectionHuaweiVRP: read password error: %v", err)
			return
		}
	}

	config := false
	configVty := false

LOOP:
	for {

		prompt := "<huawei-vrp-router>"
		if config {
			prompt = "[<huawei-vrp-router]"
		}

		// send command prompt
		if _, err := c.Write([]byte(fmt.Sprintf("\n%s", prompt))); err != nil {
			t.Logf("handleConnectionHuaweiVRP: send command prompt error: %v", err)
			return
		}

		// consume command
		if _, err := c.Read(buf); err != nil {
			if err == io.EOF {
				return // peer closed connection
			}
			t.Logf("handleConnectionHuaweiVRP: read command error: %v", err)
			return
		}

		str := string(buf)

		switch {
		case strings.HasPrefix(str, "q"): //quit
			switch {
			case configVty:
				configVty = false
			case config:
				config = false
			default:
				break LOOP
			}
		case strings.HasPrefix(str, "sys"):
			config = true
		case strings.HasPrefix(str, "screen-length"):
			// set paging
		case config && strings.HasPrefix(str, "user-interface"):
			configVty = true
		case strings.HasPrefix(str, "disp"): //show

			if options.breakConn {
				// break connection (on defer/exit)
				return
			}

			if _, err := c.Write([]byte("\nshow:\nthis is the full HuaweiVRP config\nenjoy! ;-)\n")); err != nil {
				t.Logf("handleConnectionHuaweiVRP: send sh run error: %v", err)
				return
			}
		default:
			if _, err := c.Write([]byte("\nIgnoring unknown command")); err != nil {
				t.Logf("handleConnectionHuaweiVRP: send unknown command error: %v", err)
				return
			}
		}

	}

	// send bye
	if _, err := c.Write([]byte("\nbye\n")); err != nil {
		t.Logf("handleConnectionHuaweiVRP: send bye error: %v", err)
		return
	}

}
