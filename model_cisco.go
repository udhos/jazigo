package main

import (
	"fmt"
	"io"
	"regexp"
	"time"
)

type model struct {
	name        string
	defaultAttr attributes
}

type attributes struct {
	needLoginChat               bool     // need login chat
	needEnabledMode             bool     // need enabled mode
	enableCommand               string   // enable
	usernamePromptPattern       string   // Username:
	passwordPromptPattern       string   // Password:
	enablePasswordPromptPattern string   // Password:
	disabledPromptPattern       string   // >
	enabledPromptPattern        string   // #
	commandList                 []string // show run
	disablePagerCommand         string   // term len 0
}

type device struct {
	devModel   *model
	id         string
	hostPort   string
	transports string

	loginUser      string
	loginPassword  string
	enablePassword string

	attr attributes
}

const (
	FETCH_ERR_NONE   = 0
	FETCH_ERR_TRANSP = 1
	FETCH_ERR_LOGIN  = 2
	FETCH_ERR_CHAT   = 3
	FETCH_ERR_OTHER  = 4
)

type dialog struct {
	buf []byte
}

// fetch runs in a per-device goroutine
func (d *device) fetch(logger hasPrintf, resultCh chan fetchResult, delay time.Duration) {
	modelName := d.devModel.name
	logger.Printf("fetch: %s %s %s %s delay=%dms", modelName, d.id, d.hostPort, d.transports, delay/time.Millisecond)

	if delay > 0 {
		time.Sleep(delay)
	}

	begin := time.Now()

	session, logged, err := openTransport(logger, modelName, d.id, d.hostPort, d.transports, d.loginUser, d.loginPassword)
	if err != nil {
		resultCh <- fetchResult{model: modelName, devId: d.id, devHostPort: d.hostPort, msg: fmt.Sprintf("fetch transport: %v", err), code: FETCH_ERR_TRANSP, begin: begin}
		return
	}

	logger.Printf("fetch: %s %s %s - transport OPEN logged=%v", modelName, d.id, d.hostPort, logged)

	capture := dialog{}

	if d.attr.needLoginChat && !logged {
		err1 := d.login(logger, session, &capture)
		if err1 != nil {
			resultCh <- fetchResult{model: modelName, devId: d.id, devHostPort: d.hostPort, msg: fmt.Sprintf("fetch login: %v", err1), code: FETCH_ERR_LOGIN, begin: begin}
			return
		}
	}

	resultCh <- fetchResult{model: modelName, devId: d.id, devHostPort: d.hostPort, msg: "fetch: FIXME WRITEME", code: FETCH_ERR_OTHER, begin: begin}
}

type hasTimeout interface {
	Timeout() bool
}

// readTimeout: per-read timeout (protection against inactivity)
// matchTimeout: full match timeout (protection against slow sender -- think 1 byte per second)
func (d *device) match(logger hasPrintf, t transp, capture *dialog, readTimeout, matchTimeout time.Duration, patterns []string) (int, error) {

	const badIndex = -1

	expList := make([]*regexp.Regexp, len(patterns))
	for i, p := range patterns {
		exp, badExp := regexp.Compile(p)
		if badExp != nil {
			return badIndex, fmt.Errorf("match: bad pattern '%s': %v", p, badExp)
		}
		expList[i] = exp
	}

	begin := time.Now()

	buf := make([]byte, 1000)

	for {
		now := time.Now()
		if now.Sub(begin) > matchTimeout {
			return badIndex, fmt.Errorf("match: timed out: %s", matchTimeout)
		}

		deadline := now.Add(readTimeout)
		if err := t.SetDeadline(deadline); err != nil {
			return badIndex, fmt.Errorf("match: could not set read timeout: %v", err)
		}

		n, err1 := t.Read(buf)
		if err1 != nil {
			if te, ok := err1.(hasTimeout); ok {
				if te.Timeout() {
					return badIndex, fmt.Errorf("match: read timed out (%s): %v", readTimeout, err1)
				}
			}
			if err1 == io.EOF {
				return badIndex, fmt.Errorf("match: eof: %v", err1)
			}
			return badIndex, fmt.Errorf("match: unexpected error: %v", err1)
		}
		if n < 1 {
			return badIndex, fmt.Errorf("match: unexpected empty read")
		}

		capture.buf = append(capture.buf, buf[:n]...)

		logger.Printf("match: debug: read=%d newsize=%d", n, len(capture.buf))

		for i, exp := range expList {
			match := exp.Match(capture.buf)
			if match {
				return i, nil
			}
		}
	}
}

func (d *device) login(logger hasPrintf, t transp, capture *dialog) error {

	readTimeout := 10 * time.Second  // protection against inactivity
	matchTimeout := 20 * time.Second // protection against slow sender

	m, err := d.match(logger, t, capture, readTimeout, matchTimeout, []string{d.attr.usernamePromptPattern, d.attr.passwordPromptPattern})
	if err != nil {
		return fmt.Errorf("login: could not find username prompt: %v", err)
	}

	switch m {
	case 0:
		logger.Printf("login: found username prompt")
	case 1:
		logger.Printf("login: found password prompt")
	}

	return fmt.Errorf("login: FIXME WRITEME")
}

func registerModelCiscoIOS(logger hasPrintf, models map[string]*model) {
	modelName := "cisco-ios"
	m := &model{name: modelName}

	m.defaultAttr = attributes{
		needLoginChat:               true,
		needEnabledMode:             true,
		enableCommand:               "enable",
		usernamePromptPattern:       "Username: ",
		passwordPromptPattern:       "Password: ",
		enablePasswordPromptPattern: "Password: ",
		disabledPromptPattern:       "> ",
		enabledPromptPattern:        "# ",
		commandList:                 []string{"show run"},
		disablePagerCommand:         "term len 0",
	}

	models[modelName] = m

	logger.Printf("registerModelCiscoIOS: FIXME WRITEME program chat sequence")
}

func createDevice(jaz *app, modelName, id, hostPort, transports, user, pass, enable string) {
	jaz.logf("createDevice: %s %s %s %s", modelName, id, hostPort, transports)

	mod, ok := jaz.models[modelName]
	if !ok {
		jaz.logf("createDevice: could not find model '%s'", modelName)
	}

	dev := &device{devModel: mod, id: id, hostPort: hostPort, transports: transports, loginUser: user, loginPassword: pass, enablePassword: enable}

	dev.attr = mod.defaultAttr

	jaz.devices[id] = dev
}
