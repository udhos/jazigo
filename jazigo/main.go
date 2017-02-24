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
const appVersion = "0.8"

type app struct {
	configPathPrefix string
	repositoryPath   string // filesystem
	logPathPrefix    string
	configLock       lockfile.Lockfile
	repositoryLock   lockfile.Lockfile
	logLock          lockfile.Lockfile

	table   *dev.DeviceTable
	options *conf.Options

	apHome    gwu.Panel
	apAdmin   gwu.Panel
	apLogout  gwu.Panel
	winHome   gwu.Window
	winAdmin  gwu.Window
	winLogout gwu.Window

	cssPath    string
	repoPath   string // www
	staticPath string // www

	logger *log.Logger

	filterModel string
	filterID    string
	filterHost  string

	priority    chan string
	requestChan chan dev.FetchRequest

	filterTable *dev.FilterTable
}

type hasPrintf interface {
	Printf(fmt string, v ...interface{})
}

func (a *app) logf(fmt string, v ...interface{}) {
	a.logger.Printf(fmt, v...)
}

func newApp() *app {
	app := &app{
		table:       dev.NewDeviceTable(),
		options:     conf.NewOptions(),
		logger:      log.New(os.Stdout, "", log.LstdFlags),
		priority:    make(chan string),
		requestChan: make(chan dev.FetchRequest),
		repoPath:    "repo",   // www
		staticPath:  "static", // www
	}

	return app
}

func defaultHomeDir() string {
	home := os.Getenv("JAZIGO_HOME")
	if home == "" {
		home = "/var/jazigo"
	}
	return home
}

func defaultRegionName() string {
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "sa-east-1"
	}
	return region
}

func addTrailingDot(path string) string {
	if path[len(path)-1] != '.' {
		return path + "."
	}
	return path
}

func main() {

	jaz := newApp()

	maxMainConfigLoadSize := int64(10000000) // 10M

	var runOnce bool
	var staticDir string
	var deviceImport bool
	var deviceDelete bool
	var devicePurge bool
	var deviceList bool
	var disableStdoutLog bool
	var logMaxFiles int
	var logMaxSize int64
	var logCheckInterval time.Duration
	var webListen string
	var s3region string

	defaultHome := defaultHomeDir()
	defaultConfigPrefix := filepath.Join(defaultHome, "etc", "jazigo.conf.")
	defaultRepo := filepath.Join(defaultHome, "repo")
	defaultLogPrefix := filepath.Join(defaultHome, "log", "jazigo.log.")
	defaultStaticDir := filepath.Join(defaultHome, "www")

	flag.StringVar(&jaz.configPathPrefix, "configPathPrefix", defaultConfigPrefix, "configuration path prefix")
	flag.StringVar(&jaz.repositoryPath, "repositoryPath", defaultRepo, "repository path")
	flag.StringVar(&jaz.logPathPrefix, "logPathPrefix", defaultLogPrefix, "log path prefix")
	flag.StringVar(&staticDir, "wwwStaticPath", defaultStaticDir, "directory for static www content")
	flag.StringVar(&webListen, "webListen", ":8080", "address:port for web UI")
	flag.StringVar(&s3region, "s3region", defaultRegionName(), "AWS S3 region")
	flag.BoolVar(&runOnce, "runOnce", false, "exit after scanning all devices once")
	flag.BoolVar(&deviceDelete, "deviceDelete", false, "delete devices specified in stdin")
	flag.BoolVar(&devicePurge, "devicePurge", false, "purge devices specified in stdin")
	flag.BoolVar(&deviceImport, "deviceImport", false, "import devices from stdin")
	flag.BoolVar(&deviceList, "deviceList", false, "list devices to stdout")
	flag.BoolVar(&disableStdoutLog, "disableStdoutLog", false, "disable logging to stdout")
	flag.IntVar(&logMaxFiles, "logMaxFiles", 20, "number of log files to keep")
	flag.Int64Var(&logMaxSize, "logMaxSize", 10000000, "size limit for log file")
	flag.DurationVar(&logCheckInterval, "logCheckInterval", time.Hour, "interval for checking log file size")
	flag.Parse()

	jaz.logPathPrefix = addTrailingDot(jaz.logPathPrefix)

	if store.S3Path(jaz.logPathPrefix) {
		jaz.logf("logging to Amazon S3 is not supported: %s", jaz.logPathPrefix)
		return
	}

	if store.S3Path(staticDir) {
		jaz.logf("static dir on Amazon S3 is not supported: %s", staticDir)
		return
	}

	if lockErr := exclusiveLock(jaz); lockErr != nil {
		jaz.logf("main: could not get exclusive lock: %v", lockErr)
		panic("main: refusing to run without exclusive lock")
	}
	defer exclusiveUnlock(jaz)

	fileLogger := NewLogfile(jaz.logPathPrefix, logMaxFiles, logMaxSize, logCheckInterval)

	// jaz.logger currently is stdout
	if disableStdoutLog {
		jaz.logger = log.New(fileLogger, "", log.LstdFlags)
		// logging to file only
	} else {
		jaz.logger = log.New(io.MultiWriter(os.Stdout, fileLogger), "", log.LstdFlags)
		// logging both to stdout and file
	}

	jaz.logf("%s %s starting", appName, appVersion)

	jaz.filterTable = dev.NewFilterTable(jaz.logger)
	dev.RegisterModels(jaz.logger, jaz.table)

	jaz.configPathPrefix = addTrailingDot(jaz.configPathPrefix)

	jaz.logf("config path prefix: %s", jaz.configPathPrefix)
	jaz.logf("repository path: %s", jaz.repositoryPath)

	store.Init(jaz.logger, s3region)

	// load config
	loadConfig(jaz, maxMainConfigLoadSize)

	jaz.logf("runOnce: %v", runOnce)
	opt := jaz.options.Get()
	jaz.logf("scan interval: %s", opt.ScanInterval)
	jaz.logf("holdtime: %s", opt.Holdtime)
	jaz.logf("maximum config files: %d", opt.MaxConfigFiles)
	jaz.logf("maximum concurrency: %d", opt.MaxConcurrency)

	if exit := manageDeviceList(jaz, deviceImport, deviceDelete, devicePurge, deviceList); exit != nil {
		jaz.logf("main: %v", exit)
		return
	}

	dev.UpdateLastSuccess(jaz.table, jaz.logger, jaz.repositoryPath)

	serverName := fmt.Sprintf("%s application", appName)

	// Create GUI server
	server := gwu.NewServer(appName, webListen)
	//folder := "./tls/"
	//server := gwu.NewServerTLS(appName, webListen, folder+"cert.pem", folder+"key.pem")
	server.SetText(serverName)

	staticPathFull := fmt.Sprintf("/%s/%s", appName, jaz.staticPath)
	jaz.logf("static dir: path=[%s] mapped to dir=[%s]", staticPathFull, staticDir)
	server.AddStaticDir(jaz.staticPath, staticDir)

	jaz.cssPath = fmt.Sprintf("%s/jazigo.css", staticPathFull)
	jaz.logf("css path: %s", jaz.cssPath)

	repoPath := jaz.repoPath
	repoPathFull := fmt.Sprintf("/%s/%s", appName, repoPath)
	jaz.logf("static dir: path=[%s] mapped to dir=[%s]", repoPathFull, jaz.repositoryPath)
	server.AddStaticDir(repoPath, jaz.repositoryPath)

	buildPublicWins(jaz, server)

	go dev.Spawner(jaz.table, jaz.logger, jaz.requestChan, jaz.repositoryPath, jaz.logPathPrefix, jaz.options, jaz.filterTable)

	if runOnce {
		dev.Scan(jaz.table, jaz.table.ListDevices(), jaz.logger, jaz.options.Get(), jaz.requestChan)
		close(jaz.requestChan) // shutdown Spawner
		jaz.logf("runOnce: exiting after single scan")
		return
	}

	go scanLoop(jaz)

	// Start GUI server
	server.SetLogger(jaz.logger)
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
		dev.Scan(jaz.table, jaz.table.ListDevices(), jaz.logger, opt, jaz.requestChan)
		elap := time.Since(begin)
		sleep := opt.ScanInterval - elap
		if sleep < 1 {
			sleep = 0
		}
		jaz.logf("scanLoop: sleeping for %s (target: scanInterval=%s)", sleep, opt.ScanInterval)
		time.Sleep(sleep)
	}
}

func loadConfig(jaz *app, maxSize int64) {

	var cfg *conf.Config

	lastConfig, configErr := store.FindLastConfig(jaz.configPathPrefix, jaz.logger)
	if configErr != nil {
		jaz.logf("error reading config: '%s': %v", jaz.configPathPrefix, configErr)
		cfg = conf.New()
	} else {
		jaz.logf("last config: %s", lastConfig)
		var loadErr error
		cfg, loadErr = conf.Load(lastConfig, maxSize)
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
			if id == "" {
				continue
			}
			if strings.HasPrefix(text, "#") {
				continue
			}

			jaz.logf("deleting device [%s]", id)

			if _, getErr := jaz.table.GetDevice(id); getErr != nil {
				jaz.logf("deleting device [%s] - not found: %v", id, getErr)
				continue
			}

			jaz.table.DeleteDevice(id)
		}

		saveConfig(jaz, conf.Change{})
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
			if id == "" {
				continue
			}
			if strings.HasPrefix(text, "#") {
				continue
			}

			jaz.logf("purging device [%s]", id)

			if _, getErr := jaz.table.GetDevice(id); getErr != nil {
				jaz.logf("purging device [%s] - not found: %v", id, getErr)
				continue
			}

			jaz.table.PurgeDevice(id)
		}

		saveConfig(jaz, conf.Change{})
	}

	if imp {
		jaz.logf("reading device list from stdin")

		autoID := "auto"
		nextID := jaz.table.FindDeviceFreeId(autoID)
		valueStr := nextID[len(autoID):]
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

			text = strings.TrimSpace(text)
			if text == "" {
				continue
			}
			if strings.HasPrefix(text, "#") {
				continue
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
			if id == autoID {
				id += strconv.Itoa(value)
				value++
			}

			dev.CreateDevice(jaz.table, jaz.logger, f[0], id, f[2], f[3], f[4], f[5], enable, debug, nil)
		}

		saveConfig(jaz, conf.Change{})
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
			fmt.Printf("%s %s %s %s %s %s %s %s\n", d.DevConfig.Model, d.DevConfig.Id, d.HostPort, d.Transports, d.LoginUser, d.LoginPassword, enable, debug)
		}
	}

	if del || purge || imp || list {
		return fmt.Errorf("device list management done")
	}

	return nil
}

func exclusiveLock(jaz *app) error {
	configLockPath := fmt.Sprintf("%slock", jaz.configPathPrefix)
	if !store.S3Path(configLockPath) {
		var newErr error
		if jaz.configLock, newErr = lockfile.New(configLockPath); newErr != nil {
			return fmt.Errorf("exclusiveLock: new failure: '%s': %v", configLockPath, newErr)
		}
		if err := jaz.configLock.TryLock(); err != nil {
			return fmt.Errorf("exclusiveLock: lock failure: '%s': %v", configLockPath, err)
		}
	}

	repositoryLockPath := filepath.Join(jaz.repositoryPath, "lock")
	if !store.S3Path(repositoryLockPath) {
		var newErr error
		if jaz.repositoryLock, newErr = lockfile.New(repositoryLockPath); newErr != nil {
			jaz.configLock.Unlock()
			return fmt.Errorf("exclusiveLock: new failure: '%s': %v", repositoryLockPath, newErr)
		}
		if err := jaz.repositoryLock.TryLock(); err != nil {
			jaz.configLock.Unlock()
			return fmt.Errorf("exclusiveLock: lock failure: '%s': %v", repositoryLockPath, err)
		}
	}

	logLockPath := fmt.Sprintf("%slock", jaz.logPathPrefix)
	if !store.S3Path(logLockPath) {
		var newErr error
		if jaz.logLock, newErr = lockfile.New(logLockPath); newErr != nil {
			jaz.configLock.Unlock()
			jaz.repositoryLock.Unlock()
			return fmt.Errorf("exclusiveLock: new failure: '%s': %v", logLockPath, newErr)
		}
		if err := jaz.logLock.TryLock(); err != nil {
			jaz.configLock.Unlock()
			jaz.repositoryLock.Unlock()
			return fmt.Errorf("exclusiveLock: lock failure: '%s': %v", logLockPath, err)
		}
	}

	return nil
}

func exclusiveUnlock(jaz *app) {
	configLockPath := fmt.Sprintf("%slock", jaz.configPathPrefix)
	if !store.S3Path(configLockPath) {
		if err := jaz.configLock.Unlock(); err != nil {
			jaz.logger.Printf("exclusiveUnlock: '%s': %v", configLockPath, err)
		}
	}

	repositoryLockPath := filepath.Join(jaz.repositoryPath, "lock")
	if !store.S3Path(repositoryLockPath) {
		if err := jaz.repositoryLock.Unlock(); err != nil {
			jaz.logger.Printf("exclusiveUnlock: '%s': %v", repositoryLockPath, err)
		}
	}

	logLockPath := fmt.Sprintf("%slock", jaz.logPathPrefix)
	if !store.S3Path(logLockPath) {
		if err := jaz.logLock.Unlock(); err != nil {
			jaz.logger.Printf("exclusiveUnlock: '%s': %v", logLockPath, err)
		}
	}
}

func saveConfig(jaz *app, change conf.Change) {

	devices := jaz.table.ListDevices()

	var cfg conf.Config
	cfg.Options = *jaz.options.Get() // clone
	cfg.Options.LastChange = change  // record change
	jaz.options.Set(&cfg.Options)    // update

	// copy devices from device table
	cfg.Devices = make([]conf.DevConfig, len(devices))
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
	_, saveErr := store.SaveNewConfig(jaz.configPathPrefix, cfg.Options.MaxConfigFiles, jaz.logger, confWriteFunc, true, "detect")
	if saveErr != nil {
		jaz.logger.Printf("main: could not save config: %v", saveErr)
	}
}
