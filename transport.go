package main

import (
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
	session *ssh.Session
}

func (s *transpSSH) Read(b []byte) (n int, err error) {
	return 0, fmt.Errorf("Read: FIXME WRITEME")
}

func (s *transpSSH) Write(b []byte) (n int, err error) {
	return 0, fmt.Errorf("Write: FIXME WRITEME")
}

func (s *transpSSH) SetDeadline(t time.Time) error {
	return fmt.Errorf("SetDeadline: FIXME WRITEME")
}

func (s *transpSSH) Close() error {
	return fmt.Errorf("Close: FIXME WRITEME")
}

func openTransport(modelName, devId, hostPort, transports, user, pass string) (transp, bool, error) {
	tList := strings.Split(transports, ",")
	if len(tList) < 1 {
		return nil, false, fmt.Errorf("openTransport: missing transports: [%s]", transports)
	}

	timeout := 10 * time.Second

	for _, t := range tList {
		switch t {
		case "ssh":
			logger.Printf("openTransport: %s %s %s - trying SSH", modelName, devId, hostPort)
			hp := forceHostPort(hostPort, "22")
			s, err := openSSH(modelName, devId, hp, timeout, user, pass)
			if err == nil {
				// ssh connected
				return s, true, nil
			}
			logger.Printf("openTransport: %v", err)
		default:
			logger.Printf("openTransport: %s %s %s - trying TELNET", modelName, devId, hostPort)
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

	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(pass),
		},
		Timeout: timeout,
	}
	client, err := ssh.Dial("tcp", hostPort, config)
	if err != nil {
		return nil, fmt.Errorf("openSSH: dial %s %s %s - %v", modelName, devId, hostPort, err)
	}

	ses, err := client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("openSSH: session %s %s %s - %v", modelName, devId, hostPort, err)
	}

	s := &transpSSH{session: ses}

	return s, nil
}

func openTelnet(modelName, devId, hostPort string, timeout time.Duration) (transp, error) {

	conn, err := net.DialTimeout("tcp", hostPort, timeout)
	if err != nil {
		return nil, fmt.Errorf("openTelnet: %s %s %s - %v", modelName, devId, hostPort, err)
	}

	return conn, nil
}
