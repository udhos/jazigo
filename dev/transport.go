package dev

import (
	"bytes"
	"fmt"
	"golang.org/x/crypto/ssh"
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
	conn    net.Conn
	session *ssh.Session
	buf     bytes.Buffer
}

func (s *transpSSH) Read(b []byte) (int, error) {
	n := copy(b, s.buf.Bytes())
	return n, nil

}

func (s *transpSSH) Write(b []byte) (int, error) {

	s.session.Stdout = &s.buf

	str := string(b)

	if err := s.session.Run(str); err != nil {
		return -1, fmt.Errorf("ssh.Write: %v", err)
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

func openTransport(logger hasPrintf, modelName, devId, hostPort, transports, user, pass string) (transp, bool, error) {
	tList := strings.Split(transports, ",")
	if len(tList) < 1 {
		return nil, false, fmt.Errorf("openTransport: missing transports: [%s]", transports)
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
				return s, true, nil
			}
			logger.Printf("openTransport: %v", err)
		default:
			//logger.Printf("openTransport: %s %s %s - trying TELNET", modelName, devId, hostPort)
			hp := forceHostPort(hostPort, "23")
			s, err := openTelnet(modelName, devId, hp, timeout)
			if err == nil {
				// tcp connected
				return s, false, nil
			}
			logger.Printf("openTransport: %v", err)
		}
	}

	return nil, false, fmt.Errorf("openTransport: %s %s %s %s - unable to open transport", modelName, devId, hostPort, transports)
}

func forceHostPort(hostPort, defaultPort string) string {
	i := strings.Index(hostPort, ":")
	if i < 0 {
		return fmt.Sprintf("%s:%s", hostPort, defaultPort)
	}
	return hostPort
}

func openSSH(modelName, devId, hostPort string, timeout time.Duration, user, pass string) (transp, error) {

	conn, err := net.DialTimeout("tcp", hostPort, timeout)
	if err != nil {
		return nil, fmt.Errorf("openSSH: Dial: %s %s %s - %v", modelName, devId, hostPort, err)
	}

	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(pass),
		},
		Timeout: timeout,
	}

	c, chans, reqs, err := ssh.NewClientConn(conn, hostPort, config)
	if err != nil {
		return nil, fmt.Errorf("openSSH: NewClientConn: %s %s %s - %v", modelName, devId, hostPort, err)
	}

	cli := ssh.NewClient(c, chans, reqs)

	ses, err := cli.NewSession()
	if err != nil {
		return nil, fmt.Errorf("openSSH: NewSession: %s %s %s - %v", modelName, devId, hostPort, err)
	}

	s := &transpSSH{conn: conn, session: ses}

	return s, nil
}

func openTelnet(modelName, devId, hostPort string, timeout time.Duration) (transp, error) {

	conn, err := net.DialTimeout("tcp", hostPort, timeout)
	if err != nil {
		return nil, fmt.Errorf("openTelnet: %s %s %s - %v", modelName, devId, hostPort, err)
	}

	return conn, nil
}
