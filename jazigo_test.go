package main

import (
	"fmt"
	"net"
	"strings"
	"testing"
	//"time"
)

type testServer struct {
	listener net.Listener
}

func (s *testServer) close() {
	s.listener.Close()
}

// testLogger: wrap Printf interface around *testing.T
type testLogger struct {
	*testing.T
}

func (t *testLogger) Printf(format string, v ...interface{}) {
	t.Logf("client: "+format, v...)
}

type optionsCiscoIOS struct {
	sendUsername      bool
	sendDisable       bool
	requestEnablePass bool
}

func TestCiscoIOS1(t *testing.T) {

	// launch bogus test server
	addr := ":2001"
	s, listenErr := spawnServerCiscoIOS(t, addr, optionsCiscoIOS{sendUsername: true, sendDisable: true, requestEnablePass: true})
	if listenErr != nil {
		t.Errorf("could not spawn bogus CiscoIOS server: %v", listenErr)
	}
	t.Logf("TestCiscoIOS: server running on %s", addr)

	// run client test
	jaz := newApp(&testLogger{t})
	createDevice(jaz, "cisco-ios", "lab1", "localhost"+addr, "telnet", "lab", "pass", "en")
	scanDevices(jaz)

	//time.Sleep(time.Hour)

	// shutdown server
	s.close()
}

func TestCiscoIOS2(t *testing.T) {

	// launch bogus test server
	addr := ":2002"
	s, listenErr := spawnServerCiscoIOS(t, addr, optionsCiscoIOS{sendUsername: false})
	if listenErr != nil {
		t.Errorf("could not spawn bogus CiscoIOS server: %v", listenErr)
	}
	t.Logf("TestCiscoIOS: server running on %s", addr)

	// run client test
	jaz := newApp(&testLogger{t})
	createDevice(jaz, "cisco-ios", "lab1", "localhost"+addr, "telnet", "lab", "pass", "en")
	scanDevices(jaz)

	//time.Sleep(time.Hour)

	// shutdown server
	s.close()
}

func spawnServerCiscoIOS(t *testing.T, addr string, options optionsCiscoIOS) (*testServer, error) {

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	s := &testServer{listener: ln}

	go acceptLoop(t, s, handleConnectionCiscoIOS, options)

	return s, nil
}

func acceptLoop(t *testing.T, s *testServer, handler func(*testing.T, net.Conn, optionsCiscoIOS), options optionsCiscoIOS) {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			t.Logf("accept failure, exiting: %v", err)
			return
		}
		go handler(t, conn, options)
	}
}

func handleConnectionCiscoIOS(t *testing.T, c net.Conn, options optionsCiscoIOS) {
	defer c.Close()

	buf := make([]byte, 1000)

	if options.sendUsername {
		// send username prompt
		if _, err := c.Write([]byte("Bogus CiscoIOS server\nUsername: ")); err != nil {
			t.Logf("handleConnectionCiscoIOS: send username prompt error: %v", err)
			return
		}

		// consume username
		if _, err := c.Read(buf); err != nil {
			t.Logf("handleConnectionCiscoIOS: read username error: %v", err)
			return
		}
	}

	// send password prompt
	if _, err := c.Write([]byte("\nPassword: ")); err != nil {
		t.Logf("handleConnectionCiscoIOS: send password prompt error: %v", err)
		return
	}

	// consume password
	if _, err := c.Read(buf); err != nil {
		t.Logf("handleConnectionCiscoIOS: read password error: %v", err)
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
		if _, err := c.Write([]byte(fmt.Sprintf("\nrouter%s ", prompt))); err != nil {
			t.Logf("handleConnectionCiscoIOS: send command prompt error: %v", err)
			return
		}

		// consume command
		if _, err := c.Read(buf); err != nil {
			t.Logf("handleConnectionCiscoIOS: read command error: %v", err)
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
			if _, err := c.Write([]byte("\nshow running-configuration")); err != nil {
				t.Logf("handleConnectionCiscoIOS: send sh run error: %v", err)
				return
			}
		case strings.HasPrefix(str, "en"): //enable
			if !enabled {
				// send password prompt
				if _, err := c.Write([]byte("\nPassword: ")); err != nil {
					t.Logf("handleConnectionCiscoIOS: send enable password prompt error: %v", err)
					return
				}

				// consume password
				if _, err := c.Read(buf); err != nil {
					t.Logf("handleConnectionCiscoIOS: read enable password error: %v", err)
					return
				}

				enabled = true
			}
		default:
			if _, err := c.Write([]byte("\nIgnoring unknown command")); err != nil {
				t.Logf("handleConnectionCiscoIOS: send unknown command error: %v", err)
				return
			}
		}

	}

	// send bye
	if _, err := c.Write([]byte("\nbye\n")); err != nil {
		t.Logf("handleConnectionCiscoIOS: send bye error: %v", err)
		return
	}

}
