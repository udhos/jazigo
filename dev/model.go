package dev

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/udhos/jazigo/conf"
	"github.com/udhos/jazigo/store"
)

type Model struct {
	name        string
	defaultAttr conf.DevAttributes
}

type Device struct {
	conf.DevConfig

	logger      hasPrintf
	devModel    *Model
	lastStatus  bool // true=good false=bad
	lastTry     time.Time
	lastSuccess time.Time
}

func (d *Device) Printf(format string, v ...interface{}) {
	prefix := fmt.Sprintf("%s %s %s: ", d.DevConfig.Model, d.Id, d.HostPort)
	d.logger.Printf(prefix+format, v...)
}

func (d *Device) Model() string {
	return d.devModel.name
}

func (d *Device) LastStatus() bool {
	return d.lastStatus
}

func (d *Device) LastTry() time.Time {
	return d.lastTry
}

func (d *Device) LastSuccess() time.Time {
	return d.lastSuccess
}

func (d *Device) Holdtime(now time.Time, holdtime time.Duration) time.Duration {
	return holdtime - now.Sub(d.lastSuccess)
}

func RegisterModels(logger hasPrintf, t *DeviceTable) {
	registerModelCiscoIOS(logger, t)
	registerModelCiscoIOSXR(logger, t)
	registerModelLinux(logger, t)
	registerModelJunOS(logger, t)
	registerModelHTTP(logger, t)
}

func CreateDevice(tab *DeviceTable, logger hasPrintf, modelName, id, hostPort, transports, user, pass, enable string, debug bool, change *conf.Change) error {
	logger.Printf("CreateDevice: %s %s %s %s", modelName, id, hostPort, transports)

	mod, getErr := tab.GetModel(modelName)
	if getErr != nil {
		err := fmt.Errorf("CreateDevice: could not find model '%s': %v", modelName, getErr)
		logger.Printf(err.Error())
		return err
	}

	d := NewDevice(logger, mod, id, hostPort, transports, user, pass, enable, debug)

	if change != nil {
		d.LastChange = *change
	}

	if newDevErr := tab.SetDevice(d); newDevErr != nil {
		err := fmt.Errorf("CreateDevice: could not add device '%s': %v", id, newDevErr)
		logger.Printf(err.Error())
		return err
	}

	return nil
}

func NewDeviceFromConf(tab *DeviceTable, logger hasPrintf, cfg *conf.DevConfig) (*Device, error) {
	mod, getErr := tab.GetModel(cfg.Model)
	if getErr != nil {
		return nil, fmt.Errorf("NewDeviceFromConf: could not find model '%s': %v", cfg.Model, getErr)
	}
	d := &Device{logger: logger, devModel: mod, DevConfig: *cfg}
	return d, nil
}

func NewDevice(logger hasPrintf, mod *Model, id, hostPort, transports, loginUser, loginPassword, enablePassword string, debug bool) *Device {
	d := &Device{logger: logger, devModel: mod, DevConfig: conf.DevConfig{Model: mod.name, Id: id, HostPort: hostPort, Transports: transports, LoginUser: loginUser, LoginPassword: loginPassword, EnablePassword: enablePassword, Debug: debug}}
	d.Attr = mod.defaultAttr
	return d
}

const (
	fetchErrNone     = 0
	fetchErrGetDev   = 1
	fetchErrTransp   = 2
	fetchErrLogin    = 3
	fetchErrEnable   = 4
	fetchErrPager    = 5
	fetchErrCommands = 6
	fetchErrSave     = 7
)

type FetchRequest struct {
	Id        string           // fetch this device
	ReplyChan chan FetchResult // reply on this channel
}

type FetchResult struct {
	Model       string
	DevId       string
	DevHostPort string
	Transport   string
	Msg         string    // result error message
	Code        int       // result error code
	Begin       time.Time // begin timestamp
}

type hasPrintf interface {
	Printf(fmt string, v ...interface{})
}

type dialog struct {
	save [][]byte
}

// fetch runs in a per-device goroutine
func (d *Device) Fetch(tab DeviceUpdater, logger hasPrintf, resultCh chan FetchResult, delay time.Duration, repository string, opt *conf.AppConfig, ft *FilterTable) {

	result := d.fetch(logger, delay, repository, opt.MaxConfigFiles, ft)

	good := result.Code == fetchErrNone

	updateDeviceStatus(tab, d.Id, good, time.Now(), logger, opt.Holdtime)

	if resultCh != nil {
		resultCh <- result
	}
}

func (d *Device) fetch(logger hasPrintf, delay time.Duration, repository string, maxFiles int, ft *FilterTable) FetchResult {
	modelName := d.devModel.name
	d.Printf("fetch: delay=%dms", delay/time.Millisecond)

	if delay > 0 {
		time.Sleep(delay)
	}

	begin := time.Now()

	session, transport, logged, err := openTransport(logger, modelName, d.Id, d.HostPort, d.Transports, d.LoginUser, d.LoginPassword)
	if err != nil {
		return FetchResult{Model: modelName, DevId: d.Id, DevHostPort: d.HostPort, Transport: transport, Msg: fmt.Sprintf("fetch transport: %v", err), Code: fetchErrTransp, Begin: begin}
	}

	defer session.Close()

	logger.Printf("fetch: %s %s %s - transport OPEN logged=%v", modelName, d.Id, d.HostPort, logged)

	capture := dialog{}

	enabled := false

	if d.Attr.NeedLoginChat && !logged {
		e, loginErr := d.login(logger, session, &capture)
		if loginErr != nil {
			return FetchResult{Model: modelName, DevId: d.Id, DevHostPort: d.HostPort, Transport: transport, Msg: fmt.Sprintf("fetch login: %v", loginErr), Code: fetchErrLogin, Begin: begin}
		}
		if e {
			enabled = true
		}
	}

	if d.Attr.NeedEnabledMode && !enabled {
		enableErr := d.enable(logger, session, &capture)
		if enableErr != nil {
			return FetchResult{Model: modelName, DevId: d.Id, DevHostPort: d.HostPort, Transport: transport, Msg: fmt.Sprintf("fetch enable: %v", enableErr), Code: fetchErrEnable, Begin: begin}
		}
	}

	if d.Attr.NeedPagingOff {
		pagingErr := d.pagingOff(logger, session, &capture)
		if pagingErr != nil {
			return FetchResult{Model: modelName, DevId: d.Id, DevHostPort: d.HostPort, Transport: transport, Msg: fmt.Sprintf("fetch pager off: %v", pagingErr), Code: fetchErrPager, Begin: begin}
		}
	}

	if cmdErr := d.sendCommands(logger, session, &capture); cmdErr != nil {
		d.saveRollback(logger, &capture)
		return FetchResult{Model: modelName, DevId: d.Id, DevHostPort: d.HostPort, Transport: transport, Msg: fmt.Sprintf("commands: %v", cmdErr), Code: fetchErrCommands, Begin: begin}
	}

	if saveErr := d.saveCommit(logger, &capture, repository, maxFiles, ft); saveErr != nil {
		return FetchResult{Model: modelName, DevId: d.Id, DevHostPort: d.HostPort, Transport: transport, Msg: fmt.Sprintf("save commit: %v", saveErr), Code: fetchErrSave, Begin: begin}
	}

	return FetchResult{Model: modelName, DevId: d.Id, DevHostPort: d.HostPort, Transport: transport, Code: fetchErrNone, Begin: begin}
}

func (d *Device) saveRollback(logger hasPrintf, capture *dialog) {
	capture.save = nil
}

func deviceDirectory(repository, id string) string {
	return filepath.Join(repository, id)
}

func (d *Device) DeviceDir(repository string) string {
	return deviceDirectory(repository, d.Id)
}

func DeviceFullPrefix(repository, id string) string {
	return filepath.Join(deviceDirectory(repository, id), id+".")
}

func DeviceFullPath(repository, id, file string) string {
	return filepath.Join(repository, id, file)
}

func (d *Device) DevicePathPrefix(devDir string) string {
	return filepath.Join(devDir, d.Id+".")
}

func (d *Device) saveCommit(logger hasPrintf, capture *dialog, repository string, maxFiles int, ft *FilterTable) error {

	devDir := d.DeviceDir(repository)

	if mkdirErr := os.MkdirAll(devDir, 0750); mkdirErr != nil {
		return fmt.Errorf("saveCommit: mkdir: error: %v", mkdirErr)
	}

	devPathPrefix := d.DevicePathPrefix(devDir)

	// writeFunc: copy command outputs into file
	writeFunc := func(w store.HasWrite) error {

		lineFilter, filterFound := ft.table[d.Attr.LineFilter]
		if filterFound {
			d.Printf("saveCommit: filter '%s' FOUND", d.Attr.LineFilter)
		} else {
			if d.Attr.LineFilter != "" {
				d.Printf("saveCommit: filter '%s' not found", d.Attr.LineFilter)
			}
		}

		lineNum := 1

		for _, b := range capture.save {

			var lines [][]byte
			if filterFound {
				lines = bytes.Split(b, []byte{'\n'}) // split block into lines
			} else {
				lines = [][]byte{b} // use block as single line
			}

			for _, line := range lines {

				if filterFound {
					line = lineFilter(d, d.Debug, ft, line, lineNum) // apply filter
					line = append(line, '\n')                        // restore LF removed by split
				}

				n, writeErr := w.Write(line)
				if writeErr != nil {
					return fmt.Errorf("saveCommit: writeFunc: error: %v", writeErr)
				}
				if n != len(line) {
					return fmt.Errorf("saveCommit: writeFunc: partial: wrote=%d size=%d", n, len(line))
				}

				lineNum++
			}
		}
		return nil
	}

	path, writeErr := store.SaveNewConfig(devPathPrefix, maxFiles, logger, writeFunc, d.Attr.ChangesOnly)
	if writeErr != nil {
		return fmt.Errorf("saveCommit: error: %v", writeErr)
	}

	logger.Printf("saveCommit: dev '%s' saved to '%s'", d.Id, path)

	return nil
}

type hasTimeout interface {
	Timeout() bool
}

func (d *Device) match(logger hasPrintf, t transp, capture *dialog, patterns []string) (int, []byte, error) {

	const badIndex = -1
	var matchBuf []byte

	var expList []*regexp.Regexp

	// patterns[0] == "" --> look for EOF
	if patterns[0] != "" {
		expList = make([]*regexp.Regexp, len(patterns))
		for i, p := range patterns {
			exp, badExp := regexp.Compile(p)
			if badExp != nil {
				return badIndex, matchBuf, fmt.Errorf("match: bad pattern '%s': %v", p, badExp)
			}
			expList[i] = exp
		}
	}

	begin := time.Now()
	buf := make([]byte, 100000)

READ_LOOP:
	for {
		now := time.Now()
		if now.Sub(begin) > d.Attr.MatchTimeout {
			return badIndex, matchBuf, fmt.Errorf("match: timed out: %s", d.Attr.MatchTimeout)
		}

		deadline := now.Add(d.Attr.ReadTimeout)
		if err := t.SetDeadline(deadline); err != nil {
			return badIndex, matchBuf, fmt.Errorf("match: could not set read timeout: %v", err)
		}

		eof := false

		n, readErr := t.Read(buf)
		if readErr != nil {
			if te, ok := readErr.(hasTimeout); ok {
				if te.Timeout() {
					return badIndex, matchBuf, fmt.Errorf("match: read timed out (%s): %v", d.Attr.ReadTimeout, readErr)
				}
			}
			switch readErr {
			case io.EOF:
				if d.Debug {
					d.logf("debug recv: EOF")
				}
				eof = true // EOF is normal termination for SSH transport
			case telnetNegOnly:
				if d.Debug {
					d.logf("debug recv: telnetNegotiationOnly")
				}
				continue READ_LOOP
			default:
				return badIndex, matchBuf, fmt.Errorf("match: unexpected error: %v", readErr)
			}
		}
		if n < 1 && !eof {
			return badIndex, matchBuf, fmt.Errorf("match: unexpected empty read")
		}

		lastRead := buf[:n]

		if d.Debug {
			d.logf("debug recv1(%d): [%q]", len(lastRead), lastRead)
		}

		if !d.Attr.KeepControlChars {
			matchBuf, lastRead = removeControlChars(d, d.Debug, matchBuf, lastRead)
		}

		if d.Debug {
			d.logf("debug recv2(%d): [%q]", len(lastRead), lastRead)
		}

		matchBuf = append(matchBuf, lastRead...)

		lastLine := findLastLine(matchBuf)

		if expList != nil {
			for i, exp := range expList {
				if exp.Match(lastLine) {
					if d.Debug {
						d.logf("debug matched: %d/%d [%q]", i, len(expList), lastLine)
					}
					return i, matchBuf, nil // pattern found
				}
			}
		}

		if eof {
			return badIndex, matchBuf, io.EOF
		}
	}
}

const (
	BS = 'H' - '@'
	CR = '\r'
	LF = '\n'
)

func findLastLine(buf []byte) []byte {

	// remove possible trailing CR LF from end of line
	if len(buf) > 0 && buf[len(buf)-1] == '\n' {
		// found LF
		buf = buf[:len(buf)-1] // drop LF
		if len(buf) > 0 && buf[len(buf)-1] == '\r' {
			// found CR
			buf = buf[:len(buf)-1] // drop CR
		}
	}

	lastEOL := bytes.LastIndexAny(buf, "\r\n")
	lineBegin := lastEOL + 1
	lastLine := buf[lineBegin:]

	return lastLine
}

func (d *Device) logf(format string, v ...interface{}) {
	d.logger.Printf(fmt.Sprintf("device '%s': ", d.Id)+format, v...)
}

func (d *Device) send(logger hasPrintf, t transp, msg string) error {
	return d.sendBytes(logger, t, []byte(msg))
}

func (d *Device) sendln(logger hasPrintf, t transp, msg string) error {
	if d.Attr.SupressAutoLF {
		return d.send(logger, t, msg)
	}
	return d.send(logger, t, msg+"\n")
}

func (d *Device) sendBytes(logger hasPrintf, t transp, msg []byte) error {

	deadline := time.Now().Add(d.Attr.SendTimeout)
	if err := t.SetDeadline(deadline); err != nil {
		return fmt.Errorf("send: could not set read timeout: %v", err)
	}

	if d.Debug {
		d.logf("debug send: [%q]", msg)
	}

	_, wrErr := t.Write(msg)

	return wrErr
}

func (d *Device) sendCommands(logger hasPrintf, t transp, capture *dialog) error {

	// save timeouts
	saveReadTimeout := d.Attr.ReadTimeout
	saveMatchTimeout := d.Attr.MatchTimeout

	// temporarily change timeouts
	d.Attr.ReadTimeout = d.Attr.CommandReadTimeout
	d.Attr.MatchTimeout = d.Attr.CommandMatchTimeout

	// restore timeouts
	defer func() {
		d.Attr.ReadTimeout = saveReadTimeout
		d.Attr.MatchTimeout = saveMatchTimeout
	}()

	for i, c := range d.Attr.CommandList {

		if c != "" {
			if err := d.sendln(logger, t, c); err != nil {
				return fmt.Errorf("sendCommands: could not send command [%d] '%s': %v", i, c, err)
			}
		}

		pattern := d.Attr.EnabledPromptPattern

		_, matchBuf, matchErr := d.match(logger, t, capture, []string{pattern})
		switch matchErr {
		case nil: // ok
		case io.EOF:
			if pattern != "" {
				return fmt.Errorf("sendCommands: EOF could not match command prompt: %v buf=[%s]", matchErr, matchBuf)
			}
			logger.Printf("sendCommands: found wanted EOF")
		default:
			return fmt.Errorf("sendCommands: could not match command prompt: %v buf=[%s]", matchErr, matchBuf)
		}

		if saveErr := d.save(logger, capture, c, matchBuf); saveErr != nil {
			return fmt.Errorf("sendCommands: could not save command '%s' result: %v", c, saveErr)
		}
	}

	return nil
}

func (d *Device) save(logger hasPrintf, capture *dialog, command string, buf []byte) error {

	if command != "" {
		if d.Attr.QuoteSentCommandsFormat != "" {
			command = fmt.Sprintf(d.Attr.QuoteSentCommandsFormat, command)
		}
		command = "\n" + command + "\n" // prettify
	}

	capture.save = append(capture.save, []byte(command), buf)
	return nil
}

func (d *Device) pagingOff(logger hasPrintf, t transp, capture *dialog) error {
	if pagerErr := d.sendln(logger, t, d.Attr.DisablePagerCommand); pagerErr != nil {
		return fmt.Errorf("pager off: could not send pager disabling command '%s': %v", d.Attr.DisablePagerCommand, pagerErr)
	}

	if _, _, err := d.match(logger, t, capture, []string{d.Attr.EnabledPromptPattern}); err != nil {
		return fmt.Errorf("pager off: could not match command prompt: %v", err)
	}

	return nil
}

func (d *Device) enable(logger hasPrintf, t transp, capture *dialog) error {

	// test enabled prompt

	if emptyErr := d.sendln(logger, t, ""); emptyErr != nil {
		return fmt.Errorf("enable: could not send empty: %v", emptyErr)
	}

	m0, _, err0 := d.match(logger, t, capture, []string{d.Attr.DisabledPromptPattern, d.Attr.EnabledPromptPattern})
	if err0 != nil {
		return fmt.Errorf("enable: could not find command prompt: %v", err0)
	}

	switch m0 {
	case 0:
		d.Printf("enable: found disabled command prompt")
	case 1:
		d.Printf("enable: found enabled command prompt")
		return nil
	}

	// send enable

	if enableErr := d.sendln(logger, t, d.Attr.EnableCommand); enableErr != nil {
		return fmt.Errorf("enable: could not send enable command '%s': %v", d.Attr.EnableCommand, enableErr)
	}

	m, _, err := d.match(logger, t, capture, []string{d.Attr.EnablePasswordPromptPattern, d.Attr.EnabledPromptPattern})
	if err != nil {
		return fmt.Errorf("enable: could not match after-enable prompt: %v", err)
	}

	if m == 1 {
		return nil // found enabled command prompt
	}

	if passErr := d.sendln(logger, t, d.EnablePassword); passErr != nil {
		return fmt.Errorf("enable: could not send enable password: %v", passErr)
	}

	if _, _, mismatch := d.match(logger, t, capture, []string{d.Attr.EnabledPromptPattern}); mismatch != nil {
		return fmt.Errorf("enable: could not find enabled command prompt: %v", mismatch)
	}

	return nil
}

func (d *Device) login(logger hasPrintf, t transp, capture *dialog) (bool, error) {

	m1, _, err := d.match(logger, t, capture, []string{d.Attr.UsernamePromptPattern, d.Attr.PasswordPromptPattern})
	if err != nil {
		return false, fmt.Errorf("login: could not find username prompt: %v", err)
	}

	switch m1 {
	case 0:
		d.Printf("login: found username prompt")

		if userErr := d.sendln(logger, t, d.LoginUser); userErr != nil {
			return false, fmt.Errorf("login: could not send username: %v", userErr)
		}

		_, _, err := d.match(logger, t, capture, []string{d.Attr.PasswordPromptPattern})
		if err != nil {
			return false, fmt.Errorf("login: could not find password prompt: %v", err)
		}

	case 1:
		d.Printf("login: found password prompt")
	}

	if passErr := d.sendln(logger, t, d.LoginPassword); passErr != nil {
		return false, fmt.Errorf("login: could not send password: %v", passErr)
	}

	m, _, err := d.match(logger, t, capture, []string{d.Attr.DisabledPromptPattern, d.Attr.EnabledPromptPattern})
	if err != nil {
		return false, fmt.Errorf("login: could not find command prompt: %v", err)
	}

	switch m {
	case 0:
		d.Printf("login: found disabled command prompt")
	case 1:
		d.Printf("login: found enabled command prompt")
	}

	enabled := m == 1

	return enabled, nil
}

func round(val float64) int {
	if val < 0 {
		return int(val - 0.5)
	}
	return int(val + 0.5)
}

// ClearDeviceStatus: forget about last success (expire holdtime).
// Otherwise holdtime could prevent immediate backup.
func ClearDeviceStatus(tab DeviceUpdater, devId string, logger hasPrintf, holdtime time.Duration) (*Device, error) {
	d, getErr := tab.GetDevice(devId)
	if getErr != nil {
		logger.Printf("ClearDeviceStatus: '%s' not found: %v", devId, getErr)
		return nil, getErr
	}

	now := time.Now()
	h1 := d.Holdtime(now, holdtime)

	d.lastSuccess = time.Time{} // expire holdime
	tab.UpdateDevice(d)

	h2 := d.Holdtime(now, holdtime)
	logger.Printf("ClearDeviceStatus: device %s holdtime: old=%v new=%v", devId, h1, h2)

	return d, nil
}

// UpdateLastSuccess: load device last success from filesystem.
func UpdateLastSuccess(tab *DeviceTable, logger hasPrintf, repository string) {
	for _, d := range tab.ListDevices() {
		prefix := d.DevicePathPrefix(d.DeviceDir(repository))

		lastConfig, lastErr := store.FindLastConfig(prefix, logger)
		if lastErr != nil {
			logger.Printf("UpdateLastSuccess: find last: '%s': %v", prefix, lastErr)
			continue
		}

		/*
			f, openErr := os.Open(lastConfig)
			if openErr != nil {
				logger.Printf("UpdateLastSuccess: open: '%s': %v", lastConfig, openErr)
				continue
			}

			info, statErr := f.Stat()
			if statErr != nil {
				logger.Printf("UpdateLastSuccess: stat: '%s': %v", lastConfig, statErr)
			} else {
				d.lastSuccess = info.ModTime()
				tab.UpdateDevice(d)
			}

			if closeErr := f.Close(); closeErr != nil {
				logger.Printf("UpdateLastSuccess: close: '%s': %v", lastConfig, closeErr)
			}
		*/

		modTime, _, infoErr := store.FileInfo(lastConfig)
		if infoErr != nil {
			logger.Printf("UpdateLastSuccess: info error: '%s': %v", lastConfig, infoErr)
			continue
		}

		d.lastSuccess = modTime
		tab.UpdateDevice(d)
	}
}
