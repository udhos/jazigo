package dev

import (
	"fmt"
	"io"
	"regexp"
	"time"
)

type Model struct {
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

type Device struct {
	devModel   *Model
	id         string
	hostPort   string
	transports string

	loginUser      string
	loginPassword  string
	enablePassword string

	attr attributes
}

type DeviceTable interface {
	ListDevices() []*Device
	GetModel(modelName string) (*Model, error)
	SetDevice(id string, d *Device)
}

func CreateDevice(tab DeviceTable, logger hasPrintf, modelName, id, hostPort, transports, user, pass, enable string) {
	logger.Printf("CreateDevice: %s %s %s %s", modelName, id, hostPort, transports)

	mod, err := tab.GetModel(modelName)
	if err != nil {
		logger.Printf("CreateDevice: could not find model '%s': %v", modelName, err)
		return
	}

	d := NewDevice(mod, id, hostPort, transports, user, pass, enable)

	tab.SetDevice(id, d)
}

func NewDevice(mod *Model, id, hostPort, transports, loginUser, loginPassword, enablePassword string) *Device {
	d := &Device{devModel: mod, id: id, hostPort: hostPort, transports: transports, loginUser: loginUser, loginPassword: loginPassword, enablePassword: enablePassword}
	d.attr = mod.defaultAttr
	return d
}

const (
	FETCH_ERR_NONE   = 0
	FETCH_ERR_TRANSP = 1
	FETCH_ERR_LOGIN  = 2
	FETCH_ERR_CHAT   = 3
	FETCH_ERR_OTHER  = 4
)

type FetchResult struct {
	Model       string
	DevId       string
	DevHostPort string
	Msg         string    // result error message
	Code        int       // result error code
	Begin       time.Time // begin timestamp
}

type hasPrintf interface {
	Printf(fmt string, v ...interface{})
}

type dialog struct {
	buf []byte
}

// fetch runs in a per-device goroutine
func (d *Device) Fetch(logger hasPrintf, resultCh chan FetchResult, delay time.Duration) {
	modelName := d.devModel.name
	logger.Printf("fetch: %s %s %s %s delay=%dms", modelName, d.id, d.hostPort, d.transports, delay/time.Millisecond)

	if delay > 0 {
		time.Sleep(delay)
	}

	begin := time.Now()

	session, logged, err := openTransport(logger, modelName, d.id, d.hostPort, d.transports, d.loginUser, d.loginPassword)
	if err != nil {
		resultCh <- FetchResult{Model: modelName, DevId: d.id, DevHostPort: d.hostPort, Msg: fmt.Sprintf("fetch transport: %v", err), Code: FETCH_ERR_TRANSP, Begin: begin}
		return
	}

	logger.Printf("fetch: %s %s %s - transport OPEN logged=%v", modelName, d.id, d.hostPort, logged)

	capture := dialog{}

	if d.attr.needLoginChat && !logged {
		err1 := d.login(logger, session, &capture)
		if err1 != nil {
			resultCh <- FetchResult{Model: modelName, DevId: d.id, DevHostPort: d.hostPort, Msg: fmt.Sprintf("fetch login: %v", err1), Code: FETCH_ERR_LOGIN, Begin: begin}
			return
		}
	}

	resultCh <- FetchResult{Model: modelName, DevId: d.id, DevHostPort: d.hostPort, Msg: "fetch: FIXME WRITEME", Code: FETCH_ERR_OTHER, Begin: begin}
}

type hasTimeout interface {
	Timeout() bool
}

// readTimeout: per-read timeout (protection against inactivity)
// matchTimeout: full match timeout (protection against slow sender -- think 1 byte per second)
func (d *Device) match(logger hasPrintf, t transp, capture *dialog, readTimeout, matchTimeout time.Duration, patterns []string) (int, error) {

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

func (d *Device) login(logger hasPrintf, t transp, capture *dialog) error {

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

func RegisterModels(logger hasPrintf, models map[string]*Model) {
	registerModelCiscoIOS(logger, models)
}

func registerModelCiscoIOS(logger hasPrintf, models map[string]*Model) {
	modelName := "cisco-ios"
	m := &Model{name: modelName}

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

func ScanDevices(tab DeviceTable, logger hasPrintf, maxConcurrency int, deviceDelay time.Duration) {

	devices := tab.ListDevices()
	deviceCount := len(devices)

	logger.Printf("ScanDevices: starting devices=%d", deviceCount)
	if deviceCount < 1 {
		logger.Printf("ScanDevices: aborting")
		return
	}

	begin := time.Now()

	resultCh := make(chan FetchResult)

	logger.Printf("scanDevices: non-hammering delay between captures: %d ms", deviceDelay/time.Millisecond)

	currDelay := time.Duration(0)
	elapMax := 0 * time.Second
	elapMin := 24 * time.Hour
	wait := 0
	nextDevice := 0

	for nextDevice < deviceCount || wait > 0 {

		// launch additional devices
		for nextDevice < deviceCount {
			// there are devices to process

			if maxConcurrency > 0 && wait >= maxConcurrency {
				break // max concurrent limit reached
			}

			// launch one additional per-device goroutine
			d := devices[nextDevice]
			go d.Fetch(logger, resultCh, currDelay) // per-device goroutine
			currDelay += deviceDelay
			nextDevice++
			wait++
		}

		// wait for one device to finish
		r := <-resultCh
		wait--
		elap := time.Since(r.Begin)
		logger.Printf("device result: %s %s %s msg=[%s] code=%d wait=%d remain=%d elap=%s", r.Model, r.DevId, r.DevHostPort, r.Msg, r.Code, wait, deviceCount-nextDevice, elap)
		if elap < elapMin {
			elapMin = elap
		}
		if elap > elapMax {
			elapMax = elap
		}
	}

	elapsed := time.Since(begin)
	average := elapsed / time.Duration(deviceCount)

	logger.Printf("scanDevices: finished elapsed=%s devices=%d average=%s min=%s max=%s", elapsed, deviceCount, average, elapMin, elapMax)
}
