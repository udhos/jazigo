package dev

import (
	"bytes"
	"fmt"
	"golang.org/x/crypto/ssh"
	"io"
	"net"
	"strings"
	"time"
)

type transp interface {
	Read(b []byte) (n int, err error)
	Write(b []byte) (n int, err error)
	SetDeadline(t time.Time) error
	Close() error
}

type transpSSH struct {
	conn     net.Conn
	session  *ssh.Session
	out      bytes.Buffer
	err      bytes.Buffer
	writeErr error
}

func (s *transpSSH) Read(b []byte) (int, error) {
	n := copy(b, s.out.Bytes())
	return n, io.EOF
}

func (s *transpSSH) Write(b []byte) (int, error) {

	str := string(b)

	if err := s.session.Run(str); err != nil {

		switch err.(type) {
		//case *ssh.ExitMissingError:
		//	s.writeErr = io.EOF
		default:
			return -1, fmt.Errorf("session.Run: %v out=[%s] err=[%s]", err, s.out.String(), s.err.String())
		}

	}

	return len(str), nil
}

func (s *transpSSH) SetDeadline(t time.Time) error {
	return s.conn.SetDeadline(t)
}

func (s *transpSSH) Close() error {
	err1 := s.session.Close()
	err2 := s.conn.Close()
	if err1 != nil || err2 != nil {
		return fmt.Errorf("close error: session=[%v] conn=[%v]", err1, err2)
	}
	return nil
}

func openTransport(logger hasPrintf, modelName, devId, hostPort, transports, user, pass string) (transp, string, bool, error) {
	tList := strings.Split(transports, ",")
	if len(tList) < 1 {
		return nil, transports, false, fmt.Errorf("openTransport: missing transports: [%s]", transports)
	}

	timeout := 10 * time.Second

	for _, t := range tList {
		switch t {
		case "ssh":
			//logger.Printf("openTransport: %s %s %s - trying SSH", modelName, devId, hostPort)
			hp := forceHostPort(hostPort, "22")
			s, err := openSSH(modelName, devId, hp, timeout, user, pass)
			if err == nil {
				// ssh connected
				return s, t, true, nil
			}
			logger.Printf("openTransport: %v", err)
		default:
			//logger.Printf("openTransport: %s %s %s - trying TELNET", modelName, devId, hostPort)
			hp := forceHostPort(hostPort, "23")
			s, err := openTelnet(modelName, devId, hp, timeout)
			if err == nil {
				// tcp connected
				return s, t, false, nil
			}
			logger.Printf("openTransport: %v", err)
		}
	}

	return nil, transports, false, fmt.Errorf("openTransport: %s %s %s %s - unable to open transport", modelName, devId, hostPort, transports)
}

func forceHostPort(hostPort, defaultPort string) string {
	i := strings.Index(hostPort, ":")
	if i < 0 {
		return fmt.Sprintf("%s:%s", hostPort, defaultPort)
	}
	return hostPort
}

func openSSH(modelName, devId, hostPort string, timeout time.Duration, user, pass string) (transp, error) {

	conn, dialErr := net.DialTimeout("tcp", hostPort, timeout)
	if dialErr != nil {
		return nil, fmt.Errorf("openSSH: Dial: %s %s %s - %v", modelName, devId, hostPort, dialErr)
	}

	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(pass),
		},
		Timeout: timeout,
	}

	c, chans, reqs, connErr := ssh.NewClientConn(conn, hostPort, config)
	if connErr != nil {
		return nil, fmt.Errorf("openSSH: NewClientConn: %s %s %s - %v", modelName, devId, hostPort, connErr)
	}

	cli := ssh.NewClient(c, chans, reqs)

	ses, sessionErr := cli.NewSession()
	if sessionErr != nil {
		return nil, fmt.Errorf("openSSH: NewSession: %s %s %s - %v", modelName, devId, hostPort, sessionErr)
	}

	modes := ssh.TerminalModes{}

	if ptyErr := ses.RequestPty("xterm", 80, 40, modes); ptyErr != nil {
		return nil, fmt.Errorf("openSSH: Pty: %s %s %s - %v", modelName, devId, hostPort, ptyErr)
	}

	s := &transpSSH{conn: conn, session: ses}

	ses.Stdout = &s.out
	ses.Stderr = &s.err

	return s, nil
}

func openTelnet(modelName, devId, hostPort string, timeout time.Duration) (transp, error) {

	conn, err := net.DialTimeout("tcp", hostPort, timeout)
	if err != nil {
		return nil, fmt.Errorf("openTelnet: %s %s %s - %v", modelName, devId, hostPort, err)
	}

	return conn, nil
}
