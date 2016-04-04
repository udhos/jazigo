package dev

import (
	"fmt"
	"io"
	"math/rand"
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

	// readTimeout: per-read timeout (protection against inactivity)
	// matchTimeout: full match timeout (protection against slow sender -- think 1 byte per second)
	readTimeout  time.Duration // protection against inactivity
	matchTimeout time.Duration // protection against slow sender
	sendTimeout  time.Duration // protection against inactivity
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

func DeviceMapToSlice(m map[string]*Device) []*Device {
	devices := make([]*Device, len(m))
	i := 0
	for _, d := range m {
		devices[i] = d
		i++
	}
	return devices
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
	FETCH_ERR_ENABLE = 3
	//FETCH_ERR_CHAT   = 4
	FETCH_ERR_OTHER = 5
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

	enabled := false

	if d.attr.needLoginChat && !logged {
		e, loginErr := d.login(logger, session, &capture)
		if loginErr != nil {
			resultCh <- FetchResult{Model: modelName, DevId: d.id, DevHostPort: d.hostPort, Msg: fmt.Sprintf("fetch login: %v", loginErr), Code: FETCH_ERR_LOGIN, Begin: begin}
			return
		}
		if e {
			enabled = true
		}
	}

	if d.attr.needEnabledMode && !enabled {
		enableErr := d.enable(logger, session, &capture)
		if enableErr != nil {
			resultCh <- FetchResult{Model: modelName, DevId: d.id, DevHostPort: d.hostPort, Msg: fmt.Sprintf("fetch enable: %v", enableErr), Code: FETCH_ERR_ENABLE, Begin: begin}
			return
		}
	}

	logger.Printf("Fetch: %s ENABLED OK", d.id)

	resultCh <- FetchResult{Model: modelName, DevId: d.id, DevHostPort: d.hostPort, Msg: "Fetch: FIXME WRITEME", Code: FETCH_ERR_OTHER, Begin: begin}
}

type hasTimeout interface {
	Timeout() bool
}

func (d *Device) match(logger hasPrintf, t transp, capture *dialog, patterns []string) (int, error) {

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
		if now.Sub(begin) > d.attr.matchTimeout {
			return badIndex, fmt.Errorf("match: timed out: %s", d.attr.matchTimeout)
		}

		deadline := now.Add(d.attr.readTimeout)
		if err := t.SetDeadline(deadline); err != nil {
			return badIndex, fmt.Errorf("match: could not set read timeout: %v", err)
		}

		n, err1 := t.Read(buf)
		if err1 != nil {
			if te, ok := err1.(hasTimeout); ok {
				if te.Timeout() {
					return badIndex, fmt.Errorf("match: read timed out (%s): %v", d.attr.readTimeout, err1)
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

func (d *Device) send(logger hasPrintf, t transp, msg string) error {

	deadline := time.Now().Add(d.attr.sendTimeout)
	if err := t.SetDeadline(deadline); err != nil {
		return fmt.Errorf("send: could not set read timeout: %v", err)
	}

	_, wrErr := t.Write([]byte(msg))

	return wrErr
}

func (d *Device) enable(logger hasPrintf, t transp, capture *dialog) error {
	if enableErr := d.send(logger, t, d.attr.enableCommand); enableErr != nil {
		return fmt.Errorf("enable: could not send enable command '%s': %v", d.attr.enableCommand, enableErr)
	}

	m, err := d.match(logger, t, capture, []string{d.attr.enablePasswordPromptPattern, d.attr.enabledPromptPattern})
	if err != nil {
		return fmt.Errorf("enable: could not match after-enable prompt: %v", err)
	}

	if m == 1 {
		return nil // found enabled command prompt
	}

	if passErr := d.send(logger, t, d.enablePassword); passErr != nil {
		return fmt.Errorf("enable: could not send enable password: %v", passErr)
	}

	if _, mismatch := d.match(logger, t, capture, []string{d.attr.enabledPromptPattern}); mismatch != nil {
		return fmt.Errorf("enable: could not find enabled command prompt: %v", mismatch)
	}

	return nil
}

func (d *Device) login(logger hasPrintf, t transp, capture *dialog) (bool, error) {

	m1, err := d.match(logger, t, capture, []string{d.attr.usernamePromptPattern, d.attr.passwordPromptPattern})
	if err != nil {
		return false, fmt.Errorf("login: could not find username prompt: %v", err)
	}

	switch m1 {
	case 0:
		logger.Printf("login: found username prompt")

		if userErr := d.send(logger, t, d.loginUser); userErr != nil {
			return false, fmt.Errorf("login: could not send username: %v", userErr)
		}

		_, err := d.match(logger, t, capture, []string{d.attr.passwordPromptPattern})
		if err != nil {
			return false, fmt.Errorf("login: could not find password prompt: %v", err)
		}

	case 1:
		logger.Printf("login: found password prompt")
	}

	if passErr := d.send(logger, t, d.loginPassword); passErr != nil {
		return false, fmt.Errorf("login: could not send password: %v", passErr)
	}

	m, err := d.match(logger, t, capture, []string{d.attr.disabledPromptPattern, d.attr.enabledPromptPattern})
	if err != nil {
		return false, fmt.Errorf("login: could not find command prompt: %v", err)
	}

	switch m {
	case 0:
		logger.Printf("login: found disabled command prompt")
	case 1:
		logger.Printf("login: found enabled command prompt")
	}

	enabled := m == 1

	return enabled, nil
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
		usernamePromptPattern:       `Username:\s*$`,
		passwordPromptPattern:       `Password:\s*$`,
		enablePasswordPromptPattern: `Password:\s*$`,
		disabledPromptPattern:       `\S+>\s*$`,
		enabledPromptPattern:        `\S+#\s*$`,
		commandList:                 []string{"show clock det", "show ver", "show run"},
		disablePagerCommand:         "term len 0",
		readTimeout:                 10 * time.Second,
		matchTimeout:                20 * time.Second,
		sendTimeout:                 10 * time.Second,
	}

	models[modelName] = m

	logger.Printf("registerModelCiscoIOS: FIXME WRITEME program chat sequence")
}

func round(val float64) int {
	if val < 0 {
		return int(val - 0.5)
	}
	return int(val + 0.5)
}

func ScanDevices(tab DeviceTable, logger hasPrintf, maxConcurrency int, delayMin, delayMax time.Duration) {

	devices := tab.ListDevices()
	deviceCount := len(devices)

	logger.Printf("ScanDevices: starting devices=%d maxConcurrency=%d", deviceCount, maxConcurrency)
	if deviceCount < 1 {
		logger.Printf("ScanDevices: aborting")
		return
	}

	begin := time.Now()
	random := rand.New(rand.NewSource(begin.UnixNano()))

	resultCh := make(chan FetchResult)

	logger.Printf("ScanDevices: per-device delay before starting: %d-%d ms", delayMin/time.Millisecond, delayMax/time.Millisecond)

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
			r := random.Float64()
			var delay time.Duration
			if delayMax > 0 {
				delay = time.Duration(round(r*float64(delayMax-delayMin))) + delayMin
			}
			go d.Fetch(logger, resultCh, delay) // per-device goroutine
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
