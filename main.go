package main

import (
	"fmt"
	"log"
	//"math/rand"
	"os"
	//"strconv"
	"flag"
	"path/filepath"
	"time"

	"github.com/icza/gowut/gwu"

	"github.com/udhos/jazigo/conf"
	"github.com/udhos/jazigo/dev"
)

const appName = "jazigo"
const appVersion = "0.0"

type app struct {
	configPathPrefix string
	maxConfigFiles   int
	repositoryPath   string

	models  map[string]*dev.Model  // label => model
	devices map[string]*dev.Device // id => device

	apHome    gwu.Panel
	apAdmin   gwu.Panel
	apLogout  gwu.Panel
	winHome   gwu.Window
	winAdmin  gwu.Window
	winLogout gwu.Window

	cssPath string

	logger hasPrintf
}

func (a *app) GetModel(modelName string) (*dev.Model, error) {
	if m, ok := a.models[modelName]; ok {
		return m, nil
	}
	return nil, fmt.Errorf("GetModel: not found")
}

func (a *app) SetDevice(id string, d *dev.Device) error {
	if _, found := a.devices[id]; found {
		return fmt.Errorf("app.SetDevice: found")
	}
	a.devices[id] = d
	return nil
}

func (a *app) ListDevices() []*dev.Device {
	return dev.DeviceMapToSlice(a.devices)
}

type hasPrintf interface {
	Printf(fmt string, v ...interface{})
}

func (a *app) logf(fmt string, v ...interface{}) {
	a.logger.Printf(fmt, v...)
}

func newApp(logger hasPrintf) *app {
	app := &app{
		models:  map[string]*dev.Model{},
		devices: map[string]*dev.Device{},
		logger:  logger,
	}

	app.logf("%s %s starting", appName, appVersion)

	dev.RegisterModels(app.logger, app.models)

	return app
}

func main() {

	logger := log.New(os.Stdout, "", log.LstdFlags)

	jaz := newApp(logger)

	flag.StringVar(&jaz.configPathPrefix, "configPathPrefix", "/etc/jazigo/jazigo.conf.", "configuration path prefix")
	flag.StringVar(&jaz.repositoryPath, "repositoryPath", "/var/jazigo", "repository path")
	flag.IntVar(&jaz.maxConfigFiles, "maxConfigFiles", 10, "limit number of configuration files (negative value means unlimited)")
	flag.Parse()
	jaz.logf("config path prefix: %s", jaz.configPathPrefix)
	jaz.logf("repository path: %s", jaz.repositoryPath)

	lastConfig, configErr := conf.FindLastConfig(jaz.configPathPrefix, logger)
	if configErr != nil {
		jaz.logf("error reading config: '%s': %v", jaz.configPathPrefix, configErr)
	} else {
		jaz.logf("last config: %s", lastConfig)
	}

	dev.CreateDevice(jaz, jaz.logger, "cisco-ios", "lab1", "localhost:2001", "telnet", "lab", "pass", "en")
	dev.CreateDevice(jaz, jaz.logger, "cisco-ios", "lab1", "localhost:2001", "telnet", "lab", "pass", "en") // ugh
	//dev.CreateDevice(jaz, jaz.logger, "cisco-ios", "lab2", "localhost:2002", "ssh", "lab", "pass", "en")
	//dev.CreateDevice(jaz, jaz.logger, "cisco-ios", "lab3", "localhost:2003", "telnet,ssh", "lab", "pass", "en")
	//dev.CreateDevice(jaz, jaz.logger, "cisco-ios", "lab4", "localhost:2004", "ssh,telnet", "lab", "pass", "en")
	//dev.CreateDevice(jaz, jaz.logger, "cisco-ios", "lab5", "localhost", "telnet", "lab", "pass", "en")
	//dev.CreateDevice(jaz, jaz.logger, "cisco-ios", "lab6", "localhost", "ssh", "rat", "lab", "en")
	dev.CreateDevice(jaz, jaz.logger, "linux", "lab7", "localhost", "ssh", "rat", "lab", "lab")
	dev.CreateDevice(jaz, jaz.logger, "junos", "lab8", "localhost:2008", "telnet", "rat", "lab", "lab")
	dev.CreateDevice(jaz, jaz.logger, "http", "lab9", "localhost:2009", "telnet", "", "", "")

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

	server.SetLogger(logger)

	logger.Printf("FIXME: calling one-shot ScanDevices")
	go dev.ScanDevices(jaz, logger, 3, 50*time.Millisecond, 500*time.Millisecond, jaz.repositoryPath, jaz.maxConfigFiles) // FIXME one-shot scan

	// Start GUI server
	if err := server.Start(); err != nil {
		jaz.logf("jazigo main: Cound not start GUI server: %s", err)
		return
	}
}
