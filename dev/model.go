package dev

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"time"

	"github.com/udhos/jazigo/conf"
	"github.com/udhos/jazigo/store"
)

// Model provides default attributes for model of devices.
type Model struct {
	name        string
	defaultAttr conf.DevAttributes
}

// Device is an specific device.
type Device struct {
	conf.DevConfig

	logger      hasPrintf
	devModel    *Model
	lastStatus  bool // true=good false=bad
	lastTry     time.Time
	lastSuccess time.Time
	lastElapsed time.Duration
}

// Username gets the username for login into a device.
func (d *Device) Username() string {
	if d.Model() == "mikrotik" {
		return d.DevConfig.LoginUser + "+cte"
	}
	return d.DevConfig.LoginUser
}

// Printf formats device-specific messages into logs.
func (d *Device) Printf(format string, v ...interface{}) {
	prefix := fmt.Sprintf("%s %s %s: ", d.DevConfig.Model, d.Id, d.HostPort)
	d.logger.Printf(prefix+format, v...)
}

// Model gets the model name.
func (d *Device) Model() string {
	return d.devModel.name
}

// LastStatus gets a status string for last configuration backup.
func (d *Device) LastStatus() bool {
	return d.lastStatus
}

// LastTry provides the timestamp for the last backup attempt.
func (d *Device) LastTry() time.Time {
	return d.lastTry
}

// LastSuccess informs the timestamp for the last successful backup.
func (d *Device) LastSuccess() time.Time {
	return d.lastSuccess
}

// LastElapsed gets the elapsed time for the last backup attempt.
func (d *Device) LastElapsed() time.Duration {
	return d.lastElapsed
}

// Holdtime informs the devices' remaining holdtime.
func (d *Device) Holdtime(now time.Time, holdtime time.Duration) time.Duration {
	return holdtime - now.Sub(d.lastSuccess)
}

// RegisterModels adds known device models.
func RegisterModels(logger hasPrintf, t *DeviceTable) {
	registerModelFortiOS(logger, t)
	registerModelCiscoNGA(logger, t)
	registerModelCiscoAPIC(logger, t)
	registerModelCiscoIOS(logger, t)
	registerModelCiscoIOSXR(logger, t)
	registerModelLinux(logger, t)
	registerModelJunOS(logger, t)
	registerModelHTTP(logger, t)
	registerModelRun(logger, t)
	registerModelMikrotik(logger, t)
}

// CreateDevice creates a new device in the device table.
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

// NewDeviceFromConf creates a new device from a DevConfig.
func NewDeviceFromConf(tab *DeviceTable, logger hasPrintf, cfg *conf.DevConfig) (*Device, error) {
	mod, getErr := tab.GetModel(cfg.Model)
	if getErr != nil {
		return nil, fmt.Errorf("NewDeviceFromConf: could not find model '%s': %v", cfg.Model, getErr)
	}
	d := &Device{logger: logger, devModel: mod, DevConfig: *cfg}
	return d, nil
}

// NewDevice creates a new device.
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

// FetchRequest is a request for fetching a device configuration.
type FetchRequest struct {
	Id        string           // fetch this device
	ReplyChan chan FetchResult // reply on this channel
}

// FetchResult reports the result for fetching a device configuration.
type FetchResult struct {
	Model       string
	DevID       string
	DevHostPort string
	Transport   string
	Msg         string    // result error message
	Code        int       // result error code
	Begin       time.Time // begin timestamp
	End         time.Time // end timestamp
}

type hasPrintf interface {
	Printf(fmt string, v ...interface{})
}

type dialog struct {
	save [][]byte
}

// Fetch captures a configuration for a device.
// Fetch runs in a per-device goroutine.
func (d *Device) Fetch(tab DeviceUpdater, logger hasPrintf, resultCh chan FetchResult, delay time.Duration, repository, logPathPrefix string, opt *conf.AppConfig, ft *FilterTable) {

	result := d.fetch(logger, delay, repository, opt.MaxConfigFiles, ft)

	result.End = time.Now()

	good := result.Code == fetchErrNone

	updateDeviceStatus(tab, d.Id, good, result.End, result.End.Sub(result.Begin), logger, opt.Holdtime)

	errlog(logger, result, logPathPrefix, d.Debug, d.Attr.ErrlogHistSize)

	if resultCh != nil {
		resultCh <- result
	}
}

func (d *Device) createTransport(logger hasPrintf) (transp, string, bool, error) {
	modelName := d.devModel.name

	if modelName == "run" {
		d.debugf("createTransport: %q", d.Attr.RunProg)
		return openTransportPipe(logger, modelName, d.Id, d.HostPort, d.Transports, d.LoginUser, d.LoginPassword, d.Attr.RunProg, d.Debug, d.Attr.RunTimeout)
	}

	return openTransport(logger, modelName, d.Id, d.HostPort, d.Transports, d.Username(), d.LoginPassword)
}

func (d *Device) fetch(logger hasPrintf, delay time.Duration, repository string, maxFiles int, ft *FilterTable) FetchResult {
	modelName := d.devModel.name

	if delay > 0 {
		time.Sleep(delay)
	}

	begin := time.Now()

	session, transport, logged, err := d.createTransport(logger)
	if err != nil {
		return FetchResult{Model: modelName, DevID: d.Id, DevHostPort: d.HostPort, Transport: transport, Msg: fmt.Sprintf("fetch transport: %v", err), Code: fetchErrTransp, Begin: begin}
	}

	defer session.Close()

	logger.Printf("fetch: %s %s %s - transport OPEN logged=%v", modelName, d.Id, d.HostPort, logged)

	capture := dialog{}

	enabled := false

	d.debugf("will login")

	if d.Attr.NeedLoginChat && !logged {
		e, loginErr := d.login(logger, session, &capture)
		if loginErr != nil {
			return FetchResult{Model: modelName, DevID: d.Id, DevHostPort: d.HostPort, Transport: transport, Msg: fmt.Sprintf("fetch login: %v", loginErr), Code: fetchErrLogin, Begin: begin}
		}
		if e {
			enabled = true
		}
	}

	d.debugf("will enable")

	if d.Attr.NeedEnabledMode && !enabled {
		enableErr := d.enable(logger, session, &capture)
		if enableErr != nil {
			return FetchResult{Model: modelName, DevID: d.Id, DevHostPort: d.HostPort, Transport: transport, Msg: fmt.Sprintf("fetch enable: %v", enableErr), Code: fetchErrEnable, Begin: begin}
		}
	}

	d.debugf("will disable paging: %v pattern=[%s]", d.Attr.NeedPagingOff, d.Attr.DisablePagerCommand)

	if d.Attr.NeedPagingOff {
		pagingErr := d.pagingOff(logger, session, &capture)
		if pagingErr != nil {
			return FetchResult{Model: modelName, DevID: d.Id, DevHostPort: d.HostPort, Transport: transport, Msg: fmt.Sprintf("fetch pager off: %v", pagingErr), Code: fetchErrPager, Begin: begin}
		}
	}

	d.debugf("will send commands")

	if cmdErr := d.sendCommands(logger, session, &capture); cmdErr != nil {
		d.saveRollback(logger, &capture)
		return FetchResult{Model: modelName, DevID: d.Id, DevHostPort: d.HostPort, Transport: transport, Msg: fmt.Sprintf("commands: %v", cmdErr), Code: fetchErrCommands, Begin: begin}
	}

	d.debugf("will save results")

	if saveErr := d.saveCommit(logger, &capture, repository, maxFiles, ft); saveErr != nil {
		return FetchResult{Model: modelName, DevID: d.Id, DevHostPort: d.HostPort, Transport: transport, Msg: fmt.Sprintf("save commit: %v", saveErr), Code: fetchErrSave, Begin: begin}
	}

	return FetchResult{Model: modelName, DevID: d.Id, DevHostPort: d.HostPort, Transport: transport, Code: fetchErrNone, Begin: begin}
}

func (d *Device) saveRollback(logger hasPrintf, capture *dialog) {
	capture.save = nil
}

func deviceDirectory(repository, id string) string {
	return filepath.Join(repository, id)
}

// DeviceDir gets the directory used as device repository.
func (d *Device) DeviceDir(repository string) string {
	return deviceDirectory(repository, d.Id)
}

// DeviceFullPrefix gets the full path prefix for a device repository.
func DeviceFullPrefix(repository, id string) string {
	return filepath.Join(deviceDirectory(repository, id), id+".")
}

// DeviceFullPath get the full file path for a device repository.
func DeviceFullPath(repository, id, file string) string {
	return filepath.Join(repository, id, file)
}

// DevicePathPrefix gets the full path prefix for a device repository.
func (d *Device) DevicePathPrefix(devDir string) string {
	return filepath.Join(devDir, d.Id+".")
}

func (d *Device) saveCommit(logger hasPrintf, capture *dialog, repository string, maxFiles int, ft *FilterTable) error {

	devDir := d.DeviceDir(repository)

	if mkdirErr := store.MkDir(devDir); mkdirErr != nil {
		return fmt.Errorf("saveCommit: mkdir: error: %v", mkdirErr)
	}

	devPathPrefix := d.DevicePathPrefix(devDir)

	// writeFunc: copy command outputs into file
	writeFunc := func(w store.HasWrite) error {

		lineFilter, filterFound := ft.table[d.Attr.LineFilter]
		if filterFound {
			d.debugf("saveCommit: filter '%s' FOUND", d.Attr.LineFilter)
		} else {
			if d.Attr.LineFilter != "" {
				d.debugf("saveCommit: filter '%s' not found", d.Attr.LineFilter)
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

	path, writeErr := store.SaveNewConfig(devPathPrefix, maxFiles, logger, writeFunc, d.Attr.ChangesOnly, d.Attr.S3ContentType)
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

	d.debugf("match: begin")

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

	d.debugf("match: entering read loop")

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

		d.debugf("match: reading")

		n, readErr := t.Read(buf)

		d.debugf("match: read: %d bytes", n)

		if readErr != nil {
			if te, ok := readErr.(hasTimeout); ok {
				if te.Timeout() {
					return badIndex, matchBuf, fmt.Errorf("match: read timed out: %v", readErr)
				}
			}
			switch readErr {
			case io.EOF:
				d.debugf("recv: EOF")
				eof = true // EOF is normal termination for SSH transport
			case telnetNegOnly:
				d.debugf("recv: telnetNegotiationOnly")
				continue READ_LOOP
			default:
				return badIndex, matchBuf, fmt.Errorf("match: unexpected error: %v", readErr)
			}
		}
		if n < 1 && !eof {
			return badIndex, matchBuf, fmt.Errorf("match: unexpected empty read")
		}

		lastRead := buf[:n]

		d.debugf("recv1(%d): [%q]", len(lastRead), lastRead)

		if !d.Attr.KeepControlChars {
			matchBuf, lastRead = removeControlChars(d, d.Debug, matchBuf, lastRead)
		}

		d.debugf("recv2(%d): [%q]", len(lastRead), lastRead)

		matchBuf = append(matchBuf, lastRead...)

		//lastLine := findLastLine(matchBuf)

		if expList != nil {
			var sep []byte
			if bytes.IndexByte(lastRead, CR) >= 0 {
				sep = []byte{CR, LF}
			} else {
				sep = []byte{LF}
			}
			lines := bytes.Split(lastRead, sep)
			for _, lastLine := range lines {
				for i, exp := range expList {
					d.debugf("matching: %d/%d pattern=[%s] line=[%q]", i, len(expList), patterns[i], lastLine)
					if exp.Match(lastLine) {
						d.debugf("matched: %d/%d pattern=[%s] line=[%q]", i, len(expList), patterns[i], lastLine)
						return i, matchBuf, nil // pattern found
					}
				}
			}
		}

		if eof {
			return badIndex, matchBuf, io.EOF
		}

		lineCount := bytes.Count(matchBuf, []byte{'\n'})
		d.debugf("match: FIXME limit input size: total size=%d lines=%d", len(matchBuf), lineCount)
	}
}

// Some constants.
const (
	BS = 'H' - '@' // BS backspace
	CR = '\r'      // CR carriage return
	LF = '\n'      // LF linefeed
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

func (d *Device) debugf(format string, v ...interface{}) {
	if d.Debug {
		d.logf("debug: "+format, v...)
	}
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

	d.debugf("send: [%q]", msg)

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

		d.debugf("sending command: [%s]", c)

		if c != "" {
			if err := d.sendln(logger, t, c); err != nil {
				return fmt.Errorf("sendCommands: could not send command [%d] '%s': %v", i, c, err)
			}
		}

		pattern := d.Attr.EnabledPromptPattern

		d.debugf("waiting response for command=[%s]", c)

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

		d.debugf("saving response for command=[%s]", c)

		if saveErr := d.save(logger, capture, c, matchBuf); saveErr != nil {
			return fmt.Errorf("sendCommands: could not save command '%s' result: %v", c, saveErr)
		}
	}

	return nil
}

func (d *Device) save(logger hasPrintf, capture *dialog, command string, buf []byte) error {

	if command != "" {
		command = fmt.Sprintf("%q", command)
		if d.Attr.QuoteSentCommandsFormat != "" {
			command = fmt.Sprintf(d.Attr.QuoteSentCommandsFormat, command)
		}
		command = "\n" + command + "\n"
	}

	capture.save = append(capture.save, []byte(command), buf)
	return nil
}

func (d *Device) pagingOff(logger hasPrintf, t transp, capture *dialog) error {

	if pagerErr := d.sendln(logger, t, d.Attr.DisablePagerCommand); pagerErr != nil {
		return fmt.Errorf("pager off: could not send pager disabling command '%s': %v", d.Attr.DisablePagerCommand, pagerErr)
	}

	matchCount := d.Attr.DisablePagerExtraPromptCount + 1

	for i := 0; i < matchCount; i++ {

		d.debugf("pagingOff: matching %d/%d", i, matchCount)

		var buf []byte
		var err error
		if _, buf, err = d.match(logger, t, capture, []string{d.Attr.EnabledPromptPattern}); err != nil {
			return fmt.Errorf("pagingOff: %d/%d could not match command prompt: %v", i, matchCount, err)
		}

		d.debugf("pagingOff: matching %d/%d: found buf=[%s]", i, matchCount, string(buf))
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
		d.debugf("enable: found disabled command prompt")
	case 1:
		d.debugf("enable: found enabled command prompt")
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
		d.debugf("login: found username prompt")

		if userErr := d.sendln(logger, t, d.Username()); userErr != nil {
			return false, fmt.Errorf("login: could not send username: %v", userErr)
		}

		d.debugf("login: wait password prompt")

		m2, _, err := d.match(logger, t, capture,
			[]string{
				d.Attr.PasswordPromptPattern,
				d.Attr.EnabledPromptPattern,
				d.Attr.DisabledPromptPattern,
			})
		if err != nil {
			return false, fmt.Errorf("login: could not find password prompt: %v", err)
		}

		switch m2 {
		case 1:
			d.debugf("login: found enabled command prompt")
			return true, nil
		case 2:
			d.debugf("login: found disabled command prompt")
			return false, nil
		}
		d.debugf("login: found password prompt")

	case 1:
		d.debugf("login: found password prompt (while looking for login prompt)")
	}

	d.debugf("login: will send password")

	if passErr := d.sendln(logger, t, d.LoginPassword); passErr != nil {
		return false, fmt.Errorf("login: could not send password: %v", passErr)
	}

	d.debugf("login: sent password")

	if d.Attr.PostLoginPromptPattern != "" {

		d.debugf("post-login-prompt: looking for pattern=[%s]", d.Attr.PostLoginPromptPattern)

		var m int
		var mismatch error
		m, _, mismatch = d.match(logger, t, capture,
			[]string{
				d.Attr.DisabledPromptPattern,
				d.Attr.EnabledPromptPattern,
				d.Attr.PostLoginPromptPattern,
			})
		if mismatch != nil {
			return false, fmt.Errorf("post-login-prompt: match: %v", mismatch)
		}

		if m == 2 {

			d.debugf("post-login-prompt: prompt FOUND")

			if nlErr := d.send(logger, t, d.Attr.PostLoginPromptResponse); nlErr != nil {
				return false, fmt.Errorf("post-login-prompt: error: %v", nlErr)
			}

			d.debugf("post-login-prompt: response sent: [%q]", d.Attr.PostLoginPromptResponse)
		} else {

			enabled := m == 1

			return enabled, nil
		}
	}

	m, _, err := d.match(logger, t, capture, []string{d.Attr.DisabledPromptPattern, d.Attr.EnabledPromptPattern})
	if err != nil {
		return false, fmt.Errorf("login: could not find command prompt: %v", err)
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

// ClearDeviceStatus forgets about last success (expire holdtime).
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

// UpdateLastSuccess loads device last success from filesystem.
func UpdateLastSuccess(tab *DeviceTable, logger hasPrintf, repository string) {
	for _, d := range tab.ListDevices() {
		prefix := d.DevicePathPrefix(d.DeviceDir(repository))

		lastConfig, lastErr := store.FindLastConfig(prefix, logger)
		if lastErr != nil {
			logger.Printf("UpdateLastSuccess: find last: '%s': %v", prefix, lastErr)
			continue
		}

		modTime, _, infoErr := store.FileInfo(lastConfig)
		if infoErr != nil {
			logger.Printf("UpdateLastSuccess: info error: '%s': %v", lastConfig, infoErr)
			continue
		}

		d.lastSuccess = modTime
		tab.UpdateDevice(d)
	}
}
