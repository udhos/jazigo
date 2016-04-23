package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/icza/gowut/gwu"

	"github.com/udhos/jazigo/dev"
	"github.com/udhos/jazigo/store"
)

const appName = "jazigo"
const appVersion = "0.0"

type app struct {
	configPathPrefix string
	repositoryPath   string
	maxConfigFiles   int
	holdtime         time.Duration
	scanInterval     time.Duration

	table *dev.DeviceTable

	apHome    gwu.Panel
	apAdmin   gwu.Panel
	apLogout  gwu.Panel
	winHome   gwu.Window
	winAdmin  gwu.Window
	winLogout gwu.Window

	cssPath string

	logger hasPrintf
}

type hasPrintf interface {
	Printf(fmt string, v ...interface{})
}

func (a *app) logf(fmt string, v ...interface{}) {
	a.logger.Printf(fmt, v...)
}

func newApp(logger hasPrintf) *app {
	app := &app{
		table:        dev.NewDeviceTable(),
		logger:       logger,
		holdtime:     60 * time.Second, // FIXME: 12h (do not collect/save new backup before this timeout)
		scanInterval: 10 * time.Second, // FIXME: 1h (interval between full table scan)
	}

	app.logf("%s %s starting", appName, appVersion)

	dev.RegisterModels(app.logger, app.table)

	return app
}

func main() {

	logger := log.New(os.Stdout, "", log.LstdFlags)

	jaz := newApp(logger)

	var runOnce bool

	flag.StringVar(&jaz.configPathPrefix, "configPathPrefix", "/etc/jazigo/jazigo.conf.", "configuration path prefix")
	flag.StringVar(&jaz.repositoryPath, "repositoryPath", "/var/jazigo", "repository path")
	flag.IntVar(&jaz.maxConfigFiles, "maxConfigFiles", 10, "limit number of configuration files (negative value means unlimited)")
	flag.BoolVar(&runOnce, "runOnce", false, "exit after scanning all devices once")
	flag.Parse()
	jaz.logf("config path prefix: %s", jaz.configPathPrefix)
	jaz.logf("repository path: %s", jaz.repositoryPath)
	jaz.logf("scan interval: %s", jaz.scanInterval)
	jaz.logf("holdtime: %s", jaz.holdtime)
	jaz.logf("runOnce: %v", runOnce)

	lastConfig, configErr := store.FindLastConfig(jaz.configPathPrefix, logger)
	if configErr != nil {
		jaz.logf("error reading config: '%s': %v", jaz.configPathPrefix, configErr)
	} else {
		jaz.logf("last config: %s", lastConfig)
	}

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

	dev.UpdateLastSuccess(jaz.table, jaz.logger, jaz.repositoryPath)

	appAddr := "0.0.0.0:8080"
	serverName := fmt.Sprintf("%s application", appName)

	// Create GUI server
	server := gwu.NewServer(appName, appAddr)
	//folder := "./tls/"
	//server := gwu.NewServerTLS(appName, appAddr, folder+"cert.pem", folder+"key.pem")
	server.SetText(serverName)

	gopath := os.Getenv("GOPATH")
	pkgPath := filepath.Join("src", "github.com", "udhos", "jazigo") // from package github.com/udhos/jazigo
	staticDir := filepath.Join(gopath, pkgPath, "www")
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
		dev.ScanDevices(jaz.table, logger, 3, 50*time.Millisecond, 500*time.Millisecond, jaz.repositoryPath, jaz.maxConfigFiles, jaz.holdtime)
		jaz.logf("runOnce: exiting after single scan")
		return
	}

	go func() {
		for {
			begin := time.Now()
			dev.ScanDevices(jaz.table, logger, 3, 50*time.Millisecond, 500*time.Millisecond, jaz.repositoryPath, jaz.maxConfigFiles, jaz.holdtime)
			elap := time.Since(begin)
			sleep := jaz.scanInterval - elap
			if sleep < 1 {
				sleep = 0
			}
			jaz.logf("main: scan loop sleeping for %s (target: scanInterval=%s)", sleep, jaz.scanInterval)
			time.Sleep(sleep)
		}
	}()

	// Start GUI server
	server.SetLogger(logger)
	if err := server.Start(); err != nil {
		jaz.logf("jazigo main: Cound not start GUI server: %s", err)
		return
	}
}
