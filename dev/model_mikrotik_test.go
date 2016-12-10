package dev

import (
	"io"
	"net"
	"path/filepath"
	"strings"
	"testing"

	"github.com/udhos/jazigo/conf"
	"github.com/udhos/jazigo/temp"
)

type optionsMikrotik struct {
	sendBanner bool
	breakConn  bool
}

func TestMikrotik1(t *testing.T) {

	// launch bogus test server
	addr := ":2011"
	s, listenErr := spawnServerMikrotik(t, addr, optionsMikrotik{})
	if listenErr != nil {
		t.Errorf("could not spawn bogus Mikrotik server: %v", listenErr)
	}

	// run client test
	logger := &testLogger{t}
	tab := NewDeviceTable()
	opt := conf.NewOptions()
	opt.Set(&conf.AppConfig{MaxConcurrency: 3, MaxConfigFiles: 10})
	RegisterModels(logger, tab)
	CreateDevice(tab, logger, "mikrotik", "lab1", "localhost"+addr, "telnet", "lab", "pass", "en", false, nil)

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

func TestMikrotik2(t *testing.T) {

	// launch bogus test server
	addr := ":2012"
	s, listenErr := spawnServerMikrotik(t, addr, optionsMikrotik{sendBanner: true})
	if listenErr != nil {
		t.Errorf("could not spawn bogus Mikrotik server: %v", listenErr)
	}

	// run client test
	logger := &testLogger{t}
	tab := NewDeviceTable()
	opt := conf.NewOptions()
	opt.Set(&conf.AppConfig{MaxConcurrency: 3, MaxConfigFiles: 10})
	RegisterModels(logger, tab)
	CreateDevice(tab, logger, "mikrotik", "lab1", "localhost"+addr, "telnet", "lab", "pass", "en", false, nil)

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

func TestMikrotik3(t *testing.T) {

	// launch bogus test server
	addr := ":2013"
	s, listenErr := spawnServerMikrotik(t, addr, optionsMikrotik{breakConn: true})
	if listenErr != nil {
		t.Errorf("could not spawn bogus Mikrotik server: %v", listenErr)
	}

	// run client test
	logger := &testLogger{t}
	tab := NewDeviceTable()
	opt := conf.NewOptions()
	opt.Set(&conf.AppConfig{MaxConcurrency: 3, MaxConfigFiles: 10})
	RegisterModels(logger, tab)
	CreateDevice(tab, logger, "mikrotik", "lab1", "localhost"+addr, "telnet", "lab", "pass", "en", false, nil)

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

func TestMikrotik4(t *testing.T) {

	// launch bogus test server
	addr := ":2014"
	s, listenErr := spawnServerMikrotik(t, addr, optionsMikrotik{sendBanner: true, breakConn: true})
	if listenErr != nil {
		t.Errorf("could not spawn bogus Mikrotik server: %v", listenErr)
	}

	// run client test
	logger := &testLogger{t}
	tab := NewDeviceTable()
	opt := conf.NewOptions()
	opt.Set(&conf.AppConfig{MaxConcurrency: 3, MaxConfigFiles: 10})
	RegisterModels(logger, tab)
	CreateDevice(tab, logger, "mikrotik", "lab1", "localhost"+addr, "telnet", "lab", "pass", "en", false, nil)

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

func spawnServerMikrotik(t *testing.T, addr string, options optionsMikrotik) (*testServer, error) {

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	s := &testServer{listener: ln, done: make(chan int)}

	go acceptLoopMikrotik(t, s, handleConnectionMikrotik, options)

	return s, nil
}

func acceptLoopMikrotik(t *testing.T, s *testServer, handler func(*testing.T, net.Conn, optionsMikrotik), options optionsMikrotik) {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			t.Logf("acceptLoopMikrotik: accept failure, exiting: %v", err)
			break
		}
		go handler(t, conn, options)
	}

	close(s.done)
}

func handleConnectionMikrotik(t *testing.T, c net.Conn, options optionsMikrotik) {
	defer c.Close()

	buf := make([]byte, 1000)

	// send username prompt
	if _, err := c.Write([]byte("Login: ")); err != nil {
		t.Logf("handleConnectionMikrotik: send username prompt error: %v", err)
		return
	}

	// consume username
	if _, err := c.Read(buf); err != nil {
		t.Logf("handleConnectionMikrotik: read username error: %v", err)
		return
	}

	// send password prompt
	if _, err := c.Write([]byte("\nPassword: ")); err != nil {
		t.Logf("handleConnectionMikrotik: send password prompt error: %v", err)
		return
	}

	// consume password
	if _, err := c.Read(buf); err != nil {
		t.Logf("handleConnectionMikrotik: read password error: %v", err)
		return
	}

	// send post login prompt

	if options.sendBanner {

		banner := `
  MMM      MMM       KKK                          TTTTTTTTTTT      KKK
  MMMM    MMMM       KKK                          TTTTTTTTTTT      KKK
  MMM MMMM MMM  III  KKK  KKK  RRRRRR     OOOOOO      TTT     III  KKK  KKK
  MMM  MM  MMM  III  KKKKK     RRR  RRR  OOO  OOO     TTT     III  KKKKK
  MMM      MMM  III  KKK KKK   RRRRRR    OOO  OOO     TTT     III  KKK KKK
  MMM      MMM  III  KKK  KKK  RRR  RRR   OOOOOO      TTT     III  KKK  KKK

  MikroTik RouterOS 6.37.3 (c) 1999-2016       http://www.mikrotik.com/


ROUTER HAS NO SOFTWARE KEY
----------------------------
You have 20h19m to configure the router to be remotely accessible,
and to enter the key by pasting it in a Telnet window or in Winbox.
Turn off the device to stop the timer.
See www.mikrotik.com/key for more details.

Current installation "software ID": 6ZDH-HYFR
Please press "Enter" to continue!
`

		if _, err := c.Write([]byte("\r\n" + banner)); err != nil {
			t.Logf("handleConnectionMikrotik: send banner error: %v", err)
			return
		}

		// consume response
		if _, err := c.Read(buf); err != nil {
			t.Logf("handleConnectionMikrotik: read response error: %v", err)
			return
		}

	}

LOOP:
	for {
		// send command prompt
		if _, err := c.Write([]byte("\r\n[admin@MikroTik] > ")); err != nil {
			t.Logf("handleConnectionMikrotik: send command prompt error: %v", err)
			return
		}

		// consume command
		n, readErr := c.Read(buf)
		if readErr != nil {
			if readErr == io.EOF {
				t.Logf("handleConnectionMikrotik: peer EOF")
				return // peer closed connection
			}
			t.Logf("handleConnectionMikrotik: read command error: %v", readErr)
			return
		}

		str := strings.TrimSpace(string(buf[:n]))

		t.Logf("handleConnectionMikrotik: command: [%s]", str)

		if strings.HasPrefix(str, "/") {
			str = str[1:]
		}

		switch {
		case strings.HasPrefix(str, "quit"):
			break LOOP
		case strings.HasPrefix(str, "export"):
			if options.breakConn {
				t.Logf("handleConnectionMikrotik: export command: will BREAK conn")
				return // break connection (defer/close)
			}

			config := `# jan/08/2015 22:37:22 by RouterOS 6.37.3
# software id = PSRQ-KACL
#
/ip address
add address=192.168.56.100/24 interface=ether1 network=192.168.56.0
`
			if _, err := c.Write([]byte("\r\n" + config)); err != nil {
				t.Logf("handleConnectionMikrotik: send config error: %v", err)
				return
			}

		case strings.HasPrefix(str, "system resource print"):

			system :=
				`                   uptime: 10h9m21s
                  version: 6.37.3 (stable)
               build-time: Jan/28/2014 11:11:46
              free-memory: 231.7MiB
             total-memory: 249.8MiB
                      cpu: Intel(R)
                cpu-count: 1
            cpu-frequency: 0MHz
                 cpu-load: 100%
           free-hdd-space: 3950.3MiB
          total-hdd-space: 4032.5MiB
  write-sect-since-reboot: 5896
         write-sect-total: 5896
        architecture-name: x86
               board-name: x86
                 platform: MikroTik
`
			if _, err := c.Write([]byte("\r\n" + system)); err != nil {
				t.Logf("handleConnectionMikrotik: send system error: %v", err)
				return
			}
		default:
			if _, err := c.Write([]byte("\r\nbad command name")); err != nil {
				t.Logf("handleConnectionMikrotik: send unknown command error: %v", err)
				return
			}
		}

	}

	// send bye
	if _, err := c.Write([]byte("\r\ninterrupted\r\n")); err != nil {
		t.Logf("handleConnectionMikrotik: send bye error: %v", err)
		return
	}
}
