package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
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
	repositoryPath   string
	configLock       lockfile.Lockfile
	repositoryLock   lockfile.Lockfile

	table *dev.DeviceTable

	cfg *conf.Config

	apHome    gwu.Panel
	apAdmin   gwu.Panel
	apLogout  gwu.Panel
	winHome   gwu.Window
	winAdmin  gwu.Window
	winLogout gwu.Window

	cssPath string

	logger hasPrintf

	priority chan string
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
		logger:   logger,
		priority: make(chan string),
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

func main() {

	logger := log.New(os.Stdout, "", log.LstdFlags)

	jaz := newApp(logger)

	var runOnce bool
	var staticDir string

	flag.StringVar(&jaz.configPathPrefix, "configPathPrefix", "/etc/jazigo/jazigo.conf.", "configuration path prefix")
	flag.StringVar(&jaz.repositoryPath, "repositoryPath", "/var/jazigo", "repository path")
	flag.StringVar(&staticDir, "wwwStaticPath", defaultStaticDir(), "directory for static www content")
	flag.BoolVar(&runOnce, "runOnce", false, "exit after scanning all devices once")
	flag.Parse()

	jaz.logf("runOnce: %v", runOnce)
	jaz.logf("config path prefix: %s", jaz.configPathPrefix)
	jaz.logf("repository path: %s", jaz.repositoryPath)

	if lockErr := exclusiveLock(jaz); lockErr != nil {
		jaz.logf("main: could not get exclusive lock: %v", lockErr)
		panic("main: refusing to run without exclusive lock")
	}
	defer exclusiveUnlock(jaz)

	// load config

	lastConfig, configErr := store.FindLastConfig(jaz.configPathPrefix, logger)
	if configErr != nil {
		jaz.logf("error reading config: '%s': %v", jaz.configPathPrefix, configErr)
		jaz.cfg = conf.New()
	} else {
		jaz.logf("last config: %s", lastConfig)
		var loadErr error
		jaz.cfg, loadErr = conf.Load(lastConfig)
		if loadErr != nil {
			jaz.logf("could not load config: '%s': %v", lastConfig, loadErr)
			panic("main: could not load config")
		}
	}

	jaz.logf("scan interval: %s", jaz.cfg.ScanInterval)
	jaz.logf("holdtime: %s", jaz.cfg.Holdtime)

	//dev.CreateDevice(jaz.table, jaz.logger, "cisco-ios", "lab1", "localhost:2001", "telnet", "lab", "pass", "en")
	//dev.CreateDevice(jaz.table, jaz.logger, "cisco-ios", "lab1", "localhost:2001", "telnet", "lab", "pass", "en") // ugh
	//dev.CreateDevice(jaz, jaz.logger, "cisco-ios", "lab2", "localhost:2002", "ssh", "lab", "pass", "en")
	//dev.CreateDevice(jaz, jaz.logger, "cisco-ios", "lab3", "localhost:2003", "telnet,ssh", "lab", "pass", "en")
	//dev.CreateDevice(jaz, jaz.logger, "cisco-ios", "lab4", "localhost:2004", "ssh,telnet", "lab", "pass", "en")
	//dev.CreateDevice(jaz, jaz.logger, "cisco-ios", "lab5", "localhost", "telnet", "lab", "pass", "en")
	//dev.CreateDevice(jaz, jaz.logger, "cisco-ios", "lab6", "localhost", "ssh", "rat", "lab", "en")
	//dev.CreateDevice(jaz.table, jaz.logger, "http", "lab7", "localhost:2009", "telnet", "", "", "")

	dev.CreateDevice(jaz.table, jaz.logger, "linux", "lab8", "localhost", "ssh", "rat", "lab", "lab", false)
	dev.CreateDevice(jaz.table, jaz.logger, "junos", "lab9", "ex4200lab", "ssh", "test", "lab000", "lab", false)
	dev.CreateDevice(jaz.table, jaz.logger, "junos", "lab10", "ex4200lab", "telnet", "test", "lab000", "lab", false)

	//dev.CreateDevice(jaz.table, jaz.logger, "cisco-iosxr", "lab11", "192.168.56.1:2011", "telnet", "user", "pass", "pass", false)
	//dev.CreateDevice(jaz.table, jaz.logger, "cisco-iosxr", "lab12", "192.168.56.1:2012", "ssh", "user", "pass", "cisco8", true)

	dev.UpdateLastSuccess(jaz.table, jaz.logger, jaz.repositoryPath)

	saveConfig(jaz) // FIXME this is not the right place to save the config

	appAddr := "0.0.0.0:8080"
	serverName := fmt.Sprintf("%s application", appName)

	// Create GUI server
	server := gwu.NewServer(appName, appAddr)
	//folder := "./tls/"
	//server := gwu.NewServerTLS(appName, appAddr, folder+"cert.pem", folder+"key.pem")
	server.SetText(serverName)

	staticPath := "static"
	staticPathFull := fmt.Sprintf("/%s/%s", appName, staticPath)
	jaz.logf("static dir: path=[%s] mapped to dir=[%s]", staticPathFull, staticDir)
	jaz.cssPath = fmt.Sprintf("%s/jazigo.css", staticPathFull)
	jaz.logf("css path: %s", jaz.cssPath)

	server.AddStaticDir(staticPath, staticDir)

	// create account panel
	jaz.apHome = newAccPanel("")
	jaz.apAdmin = newAccPanel("")
	jaz.apLogout = newAccPanel("")
	accountPanelUpdate(jaz, "")

	buildHomeWin(jaz, server)
	buildLoginWin(jaz, server)

	if runOnce {
		dev.ScanDevices(jaz.table, jaz.table.ListDevices(), logger, jaz.cfg.MaxConcurrency, 50*time.Millisecond, 500*time.Millisecond, jaz.repositoryPath, jaz.cfg.MaxConfigFiles, jaz.cfg.Holdtime)
		jaz.logf("runOnce: exiting after single scan")
		return
	}

	go func() {
		for {
			begin := time.Now()
			dev.ScanDevices(jaz.table, jaz.table.ListDevices(), logger, jaz.cfg.MaxConcurrency, 50*time.Millisecond, 500*time.Millisecond, jaz.repositoryPath, jaz.cfg.MaxConfigFiles, jaz.cfg.Holdtime)

		SLEEP:
			for {
				elap := time.Since(begin)
				sleep := jaz.cfg.ScanInterval - elap
				if sleep < 1 {
					sleep = 0
				}
				jaz.logf("main: sleeping for %s (target: scanInterval=%s)", sleep, jaz.cfg.ScanInterval)
				select {
				case <-time.After(sleep):
					jaz.logf("main: sleep done")
					break SLEEP
				case id := <-jaz.priority:
					jaz.logf("main: sleep interrupted by priority: device %s", id)
					d, clearErr := dev.ClearDeviceStatus(jaz.table, id, logger, jaz.cfg.Holdtime)
					if clearErr != nil {
						jaz.logf("main: sleep interrupted by priority: device %s - error: %v", id, clearErr)
						continue SLEEP
					}
					singleDevice := []*dev.Device{d}
					dev.ScanDevices(jaz.table, singleDevice, logger, jaz.cfg.MaxConcurrency, 50*time.Millisecond, 500*time.Millisecond, jaz.repositoryPath, jaz.cfg.MaxConfigFiles, jaz.cfg.Holdtime)
				}
			}
		}
	}()

	// Start GUI server
	server.SetLogger(logger)
	if err := server.Start(); err != nil {
		jaz.logf("jazigo main: Cound not start GUI server: %s", err)
		return
	}
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
	jaz.cfg.Devices = make([]conf.DevConfig, len(devices))
	for i, d := range devices {
		jaz.cfg.Devices[i] = d.DevConfig
	}

	confWriteFunc := func(w store.HasWrite) error {
		b, err := jaz.cfg.Dump()
		if err != nil {
			return err
		}
		if _, wrErr := w.Write(b); wrErr != nil {
			return wrErr
		}
		return nil
	}

	_, saveErr := store.SaveNewConfig(jaz.configPathPrefix, jaz.cfg.MaxConfigFiles, jaz.logger, confWriteFunc)
	if saveErr != nil {
		jaz.logger.Printf("main: could not save config: %v", saveErr)
		panic("main: could not save config") // FIXME log only
	}
}
