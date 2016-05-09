package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/icza/gowut/gwu"
	"github.com/udhos/lockfile"

	"github.com/udhos/jazigo/conf"
	"github.com/udhos/jazigo/dev"
	"github.com/udhos/jazigo/store"
)

const appName = "jazigo"
const appVersion = "0.0"

type app struct {
	configPathPrefix string
	repositoryPath   string // filesystem
	configLock       lockfile.Lockfile
	repositoryLock   lockfile.Lockfile

	table   *dev.DeviceTable
	options *conf.Options

	apHome    gwu.Panel
	apAdmin   gwu.Panel
	apLogout  gwu.Panel
	winHome   gwu.Window
	winAdmin  gwu.Window
	winLogout gwu.Window

	cssPath  string
	repoPath string // www

	logger hasPrintf

	filterModel string
	filterId    string
	filterHost  string

	priority     chan string
	requestChan  chan dev.FetchRequest
	oldScheduler bool
}

type hasPrintf interface {
	Printf(fmt string, v ...interface{})
}

func (a *app) logf(fmt string, v ...interface{}) {
	a.logger.Printf(fmt, v...)
}

func newApp(logger hasPrintf) *app {
	app := &app{
		table:    dev.NewDeviceTable(),
		options:  conf.NewOptions(),
		logger:   logger,
		priority: make(chan string),
		repoPath: "repo",
	}

	app.logf("%s %s starting", appName, appVersion)

	dev.RegisterModels(app.logger, app.table)

	return app
}

func defaultStaticDir() string {
	gopath := os.Getenv("GOPATH")
	pkgPath := filepath.Join("src", "github.com", "udhos", "jazigo") // from package github.com/udhos/jazigo
	return filepath.Join(gopath, pkgPath, "www")
}

func addTrailingDot(path string) string {
	if path[len(path)-1] != '.' {
		return path + "."
	}
	return path
}

func main() {

	logger := log.New(os.Stdout, "", log.LstdFlags)

	jaz := newApp(logger)

	var runOnce bool
	var staticDir string
	var deviceImport bool
	var deviceDelete bool
	var devicePurge bool
	var deviceList bool

	flag.StringVar(&jaz.configPathPrefix, "configPathPrefix", "/etc/jazigo/jazigo.conf.", "configuration path prefix")
	flag.StringVar(&jaz.repositoryPath, "repositoryPath", "/var/jazigo", "repository path")
	flag.StringVar(&staticDir, "wwwStaticPath", defaultStaticDir(), "directory for static www content")
	flag.BoolVar(&runOnce, "runOnce", false, "exit after scanning all devices once")
	flag.BoolVar(&deviceDelete, "deviceDelete", false, "delete devices specified in stdin")
	flag.BoolVar(&devicePurge, "devicePurge", false, "purge devices specified in stdin")
	flag.BoolVar(&deviceImport, "deviceImport", false, "import devices from stdin")
	flag.BoolVar(&deviceList, "deviceList", false, "list devices from stdout")
	flag.BoolVar(&jaz.oldScheduler, "oldScheduler", false, "use old scheduler")
	flag.Parse()

	jaz.configPathPrefix = addTrailingDot(jaz.configPathPrefix)

	jaz.logf("config path prefix: %s", jaz.configPathPrefix)
	jaz.logf("repository path: %s", jaz.repositoryPath)

	if lockErr := exclusiveLock(jaz); lockErr != nil {
		jaz.logf("main: could not get exclusive lock: %v", lockErr)
		panic("main: refusing to run without exclusive lock")
	}
	defer exclusiveUnlock(jaz)

	// load config
	loadConfig(jaz)

	jaz.logf("runOnce: %v", runOnce)
	opt := jaz.options.Get()
	jaz.logf("scan interval: %s", opt.ScanInterval)
	jaz.logf("holdtime: %s", opt.Holdtime)
	jaz.logf("maximum config files: %d", opt.MaxConfigFiles)
	jaz.logf("maximum concurrency: %d", opt.MaxConcurrency)

	//dev.CreateDevice(jaz.table, jaz.logger, "cisco-ios", "lab1", "localhost:2001", "telnet", "lab", "pass", "en")
	//dev.CreateDevice(jaz.table, jaz.logger, "cisco-ios", "lab1", "localhost:2001", "telnet", "lab", "pass", "en") // ugh
	//dev.CreateDevice(jaz, jaz.logger, "cisco-ios", "lab2", "localhost:2002", "ssh", "lab", "pass", "en")
	//dev.CreateDevice(jaz, jaz.logger, "cisco-ios", "lab3", "localhost:2003", "telnet,ssh", "lab", "pass", "en")
	//dev.CreateDevice(jaz, jaz.logger, "cisco-ios", "lab4", "localhost:2004", "ssh,telnet", "lab", "pass", "en")
	//dev.CreateDevice(jaz, jaz.logger, "cisco-ios", "lab5", "localhost", "telnet", "lab", "pass", "en")
	//dev.CreateDevice(jaz, jaz.logger, "cisco-ios", "lab6", "localhost", "ssh", "rat", "lab", "en")
	//dev.CreateDevice(jaz.table, jaz.logger, "http", "lab7", "localhost:2009", "telnet", "", "", "")
	//dev.CreateDevice(jaz.table, jaz.logger, "linux", "lab8", "localhost", "ssh", "rat", "lab", "lab", false)
	//dev.CreateDevice(jaz.table, jaz.logger, "junos", "lab9", "ex4200lab", "ssh", "test", "lab000", "lab", false)
	//dev.CreateDevice(jaz.table, jaz.logger, "junos", "lab10", "ex4200lab", "telnet", "test", "lab000", "lab", false)
	//dev.CreateDevice(jaz.table, jaz.logger, "cisco-iosxr", "lab11", "192.168.56.1:2011", "telnet", "user", "pass", "pass", false)
	//dev.CreateDevice(jaz.table, jaz.logger, "cisco-iosxr", "lab12", "192.168.56.1:2012", "ssh", "user", "pass", "pass", true)

	if exit := manageDeviceList(jaz, deviceImport, deviceDelete, devicePurge, deviceList); exit != nil {
		jaz.logf("main: %v", exit)
		return
	}

	dev.UpdateLastSuccess(jaz.table, jaz.logger, jaz.repositoryPath)

	appAddr := ":8080"
	serverName := fmt.Sprintf("%s application", appName)

	// Create GUI server
	server := gwu.NewServer(appName, appAddr)
	//folder := "./tls/"
	//server := gwu.NewServerTLS(appName, appAddr, folder+"cert.pem", folder+"key.pem")
	server.SetText(serverName)

	staticPath := "static"
	staticPathFull := fmt.Sprintf("/%s/%s", appName, staticPath)
	jaz.logf("static dir: path=[%s] mapped to dir=[%s]", staticPathFull, staticDir)
	server.AddStaticDir(staticPath, staticDir)

	jaz.cssPath = fmt.Sprintf("%s/jazigo.css", staticPathFull)
	jaz.logf("css path: %s", jaz.cssPath)

	repoPath := jaz.repoPath
	repoPathFull := fmt.Sprintf("/%s/%s", appName, repoPath)
	jaz.logf("static dir: path=[%s] mapped to dir=[%s]", repoPathFull, jaz.repositoryPath)
	server.AddStaticDir(repoPath, jaz.repositoryPath)

	// create account panel
	jaz.apHome = newAccPanel("")
	jaz.apAdmin = newAccPanel("")
	jaz.apLogout = newAccPanel("")
	accountPanelUpdate(jaz, "")

	buildHomeWin(jaz, server)
	buildLoginWin(jaz, server)

	if jaz.oldScheduler {

		if runOnce {
			dev.ScanDevices(jaz.table, jaz.table.ListDevices(), logger, opt.MaxConcurrency, 50*time.Millisecond, 500*time.Millisecond, jaz.repositoryPath, opt.MaxConfigFiles, opt.Holdtime)
			jaz.logf("runOnce: exiting after single scan")
			return
		}

		go func() {
			for {
				begin := time.Now()
				opt := jaz.options.Get()
				dev.ScanDevices(jaz.table, jaz.table.ListDevices(), logger, opt.MaxConcurrency, 50*time.Millisecond, 500*time.Millisecond, jaz.repositoryPath, opt.MaxConfigFiles, opt.Holdtime)

			SLEEP:
				for {
					opt = jaz.options.Get()
					elap := time.Since(begin)
					sleep := opt.ScanInterval - elap
					if sleep < 1 {
						sleep = 0
					}
					jaz.logf("main: sleeping for %s (target: scanInterval=%s)", sleep, opt.ScanInterval)
					select {
					case <-time.After(sleep):
						jaz.logf("main: sleep done")
						break SLEEP
					case id := <-jaz.priority:
						jaz.logf("main: sleep interrupted by priority: device %s", id)
						d, clearErr := dev.ClearDeviceStatus(jaz.table, id, logger, opt.Holdtime)
						if clearErr != nil {
							jaz.logf("main: sleep interrupted by priority: device %s - error: %v", id, clearErr)
							continue SLEEP
						}
						singleDevice := []*dev.Device{d}
						dev.ScanDevices(jaz.table, singleDevice, logger, opt.MaxConcurrency, 50*time.Millisecond, 500*time.Millisecond, jaz.repositoryPath, opt.MaxConfigFiles, opt.Holdtime)
					}
				}
			}
		}()

	} else {

		if runOnce {
			dev.Scan(jaz.table, jaz.table.ListDevices(), jaz.logger, jaz.options.Get())
			jaz.logf("runOnce: exiting after single scan")
			return
		}

		go scanLoop(jaz)
	}

	// Start GUI server
	server.SetLogger(logger)
	if err := server.Start(); err != nil {
		jaz.logf("jazigo main: Cound not start GUI server: %s", err)
		return
	}
}

func scanLoop(jaz *app) {
	for {
		jaz.logf("scanLoop: starting")
		opt := jaz.options.Get()
		begin := time.Now()
		dev.Scan(jaz.table, jaz.table.ListDevices(), jaz.logger, opt)
		elap := time.Since(begin)
		sleep := opt.ScanInterval - elap
		if sleep < 1 {
			sleep = 0
		}
		jaz.logf("scanLoop: sleeping for %s (target: scanInterval=%s)", sleep, opt.ScanInterval)
		time.Sleep(sleep)
	}
}

func loadConfig(jaz *app) {

	var cfg *conf.Config

	lastConfig, configErr := store.FindLastConfig(jaz.configPathPrefix, jaz.logger)
	if configErr != nil {
		jaz.logf("error reading config: '%s': %v", jaz.configPathPrefix, configErr)
		cfg = conf.New()
	} else {
		jaz.logf("last config: %s", lastConfig)
		var loadErr error
		cfg, loadErr = conf.Load(lastConfig)
		if loadErr != nil {
			jaz.logf("could not load config: '%s': %v", lastConfig, loadErr)
			panic("main: could not load config")
		}
	}

	jaz.options.Set(&cfg.Options)

	for _, c := range cfg.Devices {
		d, newErr := dev.NewDeviceFromConf(jaz.table, jaz.logger, &c)
		if newErr != nil {
			jaz.logger.Printf("loadConfig: failure creating device '%s': %v", c.Id, newErr)
			continue
		}
		if addErr := jaz.table.SetDevice(d); addErr != nil {
			jaz.logger.Printf("loadConfig: failure adding device '%s': %v", c.Id, addErr)
			continue
		}
		jaz.logger.Printf("loadConfig: loaded device '%s'", c.Id)
	}
}

func manageDeviceList(jaz *app, imp, del, purge, list bool) error {
	if del && purge {
		return fmt.Errorf("deviceDelete and devicePurge are mutually exclusive")
	}
	if imp && del {
		return fmt.Errorf("deviceImport and deviceDelete are mutually exclusive")
	}
	if imp && purge {
		return fmt.Errorf("deviceImport and devicePurge are mutually exclusive")
	}

	if del {
		jaz.logf("main: reading device list from stdin")

		reader := bufio.NewReader(os.Stdin)
	LOOP_DEL:
		for {
			text, inErr := reader.ReadString('\n')
			switch inErr {
			case io.EOF:
				break LOOP_DEL
			case nil:
			default:
				return fmt.Errorf("stdin error: %v", inErr)
			}

			id := strings.TrimSpace(text)

			jaz.logf("deleting device [%s]", id)

			if _, getErr := jaz.table.GetDevice(id); getErr != nil {
				jaz.logf("deleting device [%s] - not found: %v", id, getErr)
				continue
			}

			jaz.table.DeleteDevice(id)
		}

		saveConfig(jaz)
	}

	if purge {
		jaz.logf("main: reading device list from stdin")

		reader := bufio.NewReader(os.Stdin)
	LOOP_PURGE:
		for {
			text, inErr := reader.ReadString('\n')
			switch inErr {
			case io.EOF:
				break LOOP_PURGE
			case nil:
			default:
				return fmt.Errorf("stdin error: %v", inErr)
			}

			id := strings.TrimSpace(text)

			jaz.logf("purging device [%s]", id)

			if _, getErr := jaz.table.GetDevice(id); getErr != nil {
				jaz.logf("purging device [%s] - not found: %v", id, getErr)
				continue
			}

			jaz.table.PurgeDevice(id)
		}

		saveConfig(jaz)
	}

	if imp {
		jaz.logf("reading device list from stdin")

		autoId := "auto"
		nextId := jaz.table.FindDeviceFreeId(autoId)
		valueStr := nextId[len(autoId):]
		value, valErr := strconv.Atoi(valueStr)
		if valErr != nil {
			return fmt.Errorf("could not get free device id: %v", valErr)
		}

		reader := bufio.NewReader(os.Stdin)
	LOOP_ADD:
		for {
			text, inErr := reader.ReadString('\n')
			switch inErr {
			case io.EOF:
				break LOOP_ADD
			case nil:
			default:
				return fmt.Errorf("stdin error: %v", inErr)
			}

			f := strings.Fields(text)

			count := len(f)
			if count < 6 {
				return fmt.Errorf("missing fields from device line: [%s]", text)
			}
			enable := ""
			if count > 6 {
				enable = f[6]
			}
			debug := false
			if count > 7 {
				debug = true
			}

			id := f[1]
			if id == autoId {
				id += strconv.Itoa(value)
				value++
			}

			dev.CreateDevice(jaz.table, jaz.logger, f[0], id, f[2], f[3], f[4], f[5], enable, debug)
		}

		saveConfig(jaz)
	}

	if list {
		devices := jaz.table.ListDevices()

		jaz.logf("main: issuing device list to stdout: %d devices", len(devices))

		for _, d := range devices {
			enable := d.EnablePassword
			if enable == "" {
				enable = "."
			}
			debug := ""
			if d.Debug {
				debug = "debug"
			}
			fmt.Printf("%s %s %s %s %s %s %s %s\n", d.DevConfig.Model, d.Id, d.HostPort, d.Transports, d.LoginUser, d.LoginPassword, enable, debug)
		}
	}

	if del || purge || imp || list {
		return fmt.Errorf("device list management done")
	}

	return nil
}

func exclusiveLock(jaz *app) error {
	configLockPath := fmt.Sprintf("%slock", jaz.configPathPrefix)
	var newErr error
	if jaz.configLock, newErr = lockfile.New(configLockPath); newErr != nil {
		return fmt.Errorf("exclusiveLock: new failure: '%s': %v", configLockPath, newErr)
	}
	if err := jaz.configLock.TryLock(); err != nil {
		return fmt.Errorf("exclusiveLock: lock failure: '%s': %v", configLockPath, err)
	}

	repositoryLockPath := filepath.Join(jaz.repositoryPath, "lock")
	if jaz.repositoryLock, newErr = lockfile.New(repositoryLockPath); newErr != nil {
		jaz.configLock.Unlock()
		return fmt.Errorf("exclusiveLock: new failure: '%s': %v", repositoryLockPath, newErr)
	}
	if err := jaz.repositoryLock.TryLock(); err != nil {
		jaz.configLock.Unlock()
		return fmt.Errorf("exclusiveLock: lock failure: '%s': %v", repositoryLockPath, err)
	}

	return nil
}

func exclusiveUnlock(jaz *app) {
	configLockPath := fmt.Sprintf("%slock", jaz.configPathPrefix)
	if err := jaz.configLock.Unlock(); err != nil {
		jaz.logger.Printf("exclusiveUnlock: '%s': %v", configLockPath, err)
	}

	repositoryLockPath := filepath.Join(jaz.repositoryPath, "lock")
	if err := jaz.repositoryLock.Unlock(); err != nil {
		jaz.logger.Printf("exclusiveUnlock: '%s': %v", repositoryLockPath, err)
	}
}

func saveConfig(jaz *app) {

	devices := jaz.table.ListDevices()

	var cfg conf.Config
	cfg.Options = *jaz.options.Get() // clone
	cfg.Devices = make([]conf.DevConfig, len(devices))

	// copy devices from device table
	for i, d := range devices {
		cfg.Devices[i] = d.DevConfig
	}

	confWriteFunc := func(w store.HasWrite) error {
		b, err := cfg.Dump()
		if err != nil {
			return err
		}
		n, wrErr := w.Write(b)
		if wrErr != nil {
			return wrErr
		}
		if n != len(b) {
			return fmt.Errorf("saveConfig: partial write: wrote=%d size=%d", n, len(b))
		}
		return nil
	}

	// save
	_, saveErr := store.SaveNewConfig(jaz.configPathPrefix, cfg.Options.MaxConfigFiles, jaz.logger, confWriteFunc)
	if saveErr != nil {
		jaz.logger.Printf("main: could not save config: %v", saveErr)
	}
}
