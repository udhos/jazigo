package dev

import (
	//"bytes"
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
	SetWriteDeadline(t time.Time) error
	Close() error
}

type transpTCP struct {
	net.Conn
}

type transpTelnet struct {
	net.Conn
	logger hasPrintf
}

func (s *transpTelnet) Read(b []byte) (int, error) {
	n, err := s.Conn.Read(b)
	if err != nil {
		return n, err
	}
	return telnetNegotiation(b, n, s)
}

type transpSSH struct {
	devLabel string
	conn     net.Conn
	client   *ssh.Client
	session  *ssh.Session
	writer   io.Writer
	reader   io.Reader
}

func (s *transpSSH) Read(b []byte) (int, error) {
	return s.reader.Read(b)
}

func (s *transpSSH) Write(b []byte) (int, error) {

	n, err := s.writer.Write(b)
	if err != nil {
		//return -1, fmt.Errorf("ssh write(%s): %v out=[%s] err=[%s]", b, err, s.out.Bytes(), s.err.Bytes())
		return -1, fmt.Errorf("ssh write(%s): %v", b, err)
	}

	return n, nil
}

func (s *transpSSH) SetDeadline(t time.Time) error {
	return s.conn.SetDeadline(t)
}

func (s *transpSSH) SetWriteDeadline(t time.Time) error {
	return s.conn.SetWriteDeadline(t)
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
			s, err := openSSH(logger, modelName, devId, hp, timeout, user, pass)
			if err == nil {
				return s, t, true, nil
			}
			logger.Printf("openTransport: %v", err)
		case "telnet":
			//logger.Printf("openTransport: %s %s %s - trying TELNET", modelName, devId, hostPort)
			hp := forceHostPort(hostPort, "23")
			s, err := openTelnet(logger, modelName, devId, hp, timeout)
			if err == nil {
				return s, t, false, nil
			}
			logger.Printf("openTransport: %v", err)
		default:
			//logger.Printf("openTransport: %s %s %s - trying TCP", modelName, devId, hostPort)
			s, err := openTCP(logger, modelName, devId, hostPort, timeout)
			if err == nil {
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

func openSSH(logger hasPrintf, modelName, devId, hostPort string, timeout time.Duration, user, pass string) (transp, error) {

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

	s := &transpSSH{conn: conn, client: cli, devLabel: fmt.Sprintf("%s %s %s", modelName, devId, hostPort)}

	ses, sessionErr := s.client.NewSession()
	if sessionErr != nil {
		return nil, fmt.Errorf("openSSH: NewSession: %s - %v", s.devLabel, sessionErr)
	}

	s.session = ses

	modes := ssh.TerminalModes{}

	if ptyErr := ses.RequestPty("xterm", 80, 40, modes); ptyErr != nil {
		return nil, fmt.Errorf("openSSH: Pty: %s - %v", s.devLabel, ptyErr)
	}

	pipeOut, outErr := ses.StdoutPipe()
	if outErr != nil {
		return nil, fmt.Errorf("openSSH: StdoutPipe: %s - %v", s.devLabel, outErr)
	}

	pipeErr, errErr := ses.StderrPipe()
	if errErr != nil {
		return nil, fmt.Errorf("openSSH: StderrPipe: %s - %v", s.devLabel, errErr)
	}

	s.reader = io.MultiReader(pipeOut, pipeErr)

	writer, wrErr := ses.StdinPipe()
	if wrErr != nil {
		return nil, fmt.Errorf("openSSH: StdinPipe: %s - %v", s.devLabel, wrErr)
	}

	s.writer = writer

	if shellErr := ses.Shell(); shellErr != nil {
		return nil, fmt.Errorf("openSSH: Remote shell error: %s - %v", s.devLabel, shellErr)
	}

	return s, nil
}

func openTelnet(logger hasPrintf, modelName, devId, hostPort string, timeout time.Duration) (transp, error) {

	conn, err := net.DialTimeout("tcp", hostPort, timeout)
	if err != nil {
		return nil, fmt.Errorf("openTelnet: %s %s %s - %v", modelName, devId, hostPort, err)
	}

	return &transpTelnet{conn, logger}, nil
}

func openTCP(logger hasPrintf, modelName, devId, hostPort string, timeout time.Duration) (transp, error) {

	conn, err := net.DialTimeout("tcp", hostPort, timeout)
	if err != nil {
		return nil, fmt.Errorf("openTCP: %s %s %s - %v", modelName, devId, hostPort, err)
	}

	return &transpTCP{conn}, nil
}
