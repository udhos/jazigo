package dev

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
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

type telnetOptions struct {
	supressGoAhead bool
	linemode       bool
}

func (s *transpTelnet) Read(b []byte) (int, error) {
	n1, err1 := s.Conn.Read(b)
	if err1 != nil {
		return n1, err1
	}
	n2, err2 := telnetNegotiation(b, n1, s, s.logger, false)
	return n2, err2
}

type transpPipe struct {
	proc     *exec.Cmd
	stdout   io.ReadCloser
	stderr   io.ReadCloser
	reader   io.Reader
	writer   io.WriteCloser
	logger   hasPrintf
	devLabel string
	debug    bool
	ctx      context.Context
	cancel   context.CancelFunc
}

func (s *transpPipe) result(n int, err error) (int, error) {
	var waitErr error
	if err != nil {
		waitErr = s.proc.Wait()
	}

	ctxErr := s.ctx.Err()

	if ctxErr != nil || waitErr != nil {
		return n, fmt.Errorf("transPipe result error: error=[%v] context=[%v] wait=[%v]", err, ctxErr, waitErr)
	}

	return n, err
}

func (s *transpPipe) Read(b []byte) (int, error) {
	n, err := s.reader.Read(b)
	return s.result(n, err)
}

func (s *transpPipe) Write(b []byte) (int, error) {
	n, err := s.writer.Write(b)
	return s.result(n, err)
}

func (s *transpPipe) SetDeadline(t time.Time) error {
	s.logger.Printf("transpPipe.SetDeadline: FIXME WRITEME")
	return nil
}

func (s *transpPipe) SetWriteDeadline(t time.Time) error {
	s.logger.Printf("transpPipe.SetWriteDeadline: FIXME WRITEME")
	return nil
}

func (s *transpPipe) Close() error {

	if s.debug {
		s.logger.Printf("transpPipe.Close: %s contextErr=[%v]", s.devLabel, s.ctx.Err())
	}

	s.cancel()

	err1 := s.stdout.Close()
	err2 := s.stderr.Close()
	err3 := s.writer.Close()

	if err1 != nil || err2 != nil || err3 != nil {
		return fmt.Errorf("transpPipe: close error: out=[%v] err=[%v] writer=[%v]", err1, err2, err3)
	}

	return nil
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

func openTransportPipe(logger hasPrintf, modelName, devId, hostPort, transports, user, pass string, args []string, debug bool, timeout time.Duration) (transp, string, bool, error) {
	s, err := openPipe(logger, modelName, devId, hostPort, transports, user, pass, args, debug, timeout)
	return s, "pipe", true, err
}

func openPipe(logger hasPrintf, modelName, devId, hostPort, transports, user, pass string, args []string, debug bool, timeout time.Duration) (transp, error) {

	devLabel := fmt.Sprintf("%s %s %s", modelName, devId, hostPort)

	logger.Printf("openPipe: %s - opening", devLabel)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	c := exec.CommandContext(ctx, args[0], args[1:]...)

	pipeOut, outErr := c.StdoutPipe()
	if outErr != nil {
		cancel()
		return nil, fmt.Errorf("openPipe: StdoutPipe: %s - %v", devLabel, outErr)
	}

	pipeErr, errErr := c.StderrPipe()
	if errErr != nil {
		cancel()
		return nil, fmt.Errorf("openPipe: StderrPipe: %s - %v", devLabel, errErr)
	}

	writer, wrErr := c.StdinPipe()
	if wrErr != nil {
		cancel()
		return nil, fmt.Errorf("openPipe: StdinPipe: %s - %v", devLabel, wrErr)
	}

	s := &transpPipe{proc: c, logger: logger, devLabel: devLabel, debug: debug}
	s.reader = io.MultiReader(pipeOut, pipeErr)
	s.stdout = pipeOut
	s.stderr = pipeErr
	s.writer = writer
	s.ctx = ctx
	s.cancel = cancel

	logger.Printf("openPipe: %s - starting", devLabel)

	os.Setenv("JAZIGO_DEV_ID", devId)
	os.Setenv("JAZIGO_DEV_HOSTPORT", hostPort)
	os.Setenv("JAZIGO_DEV_USER", user)
	os.Setenv("JAZIGO_DEV_PASS", pass)

	if startErr := s.proc.Start(); startErr != nil {
		s.Close()
		return nil, fmt.Errorf("openPipe: error: %v", startErr)
	}

	logger.Printf("openPipe: %s - started", devLabel)

	return s, nil
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
			hp := forceHostPort(hostPort, "22")
			s, err := openSSH(logger, modelName, devId, hp, timeout, user, pass)
			if err == nil {
				return s, t, true, nil
			}
			logger.Printf("openTransport: %v", err)
		case "telnet":
			hp := forceHostPort(hostPort, "23")
			s, err := openTelnet(logger, modelName, devId, hp, timeout)
			if err == nil {
				return s, t, false, nil
			}
			logger.Printf("openTransport: %v", err)
		default:
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

	conf := &ssh.Config{}
	conf.SetDefaults()
	conf.Ciphers = append(conf.Ciphers, "3des-cbc") // 3des-cbc is needed for IOS XR

	config := &ssh.ClientConfig{
		Config: *conf,
		User:   user,
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

	s := &transpSSH{conn: conn, client: cli, devLabel: fmt.Sprintf("%s %s %s", modelName, devId, hostPort) /*, logger: logger*/}

	ses, sessionErr := s.client.NewSession()
	if sessionErr != nil {
		return nil, fmt.Errorf("openSSH: NewSession: %s - %v", s.devLabel, sessionErr)
	}

	s.session = ses

	modes := ssh.TerminalModes{
		ssh.ECHO: 0, // disable echoing
	}

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
