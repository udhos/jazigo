package main

import (
	"fmt"
	//"github.com/udhos/gowut/gwu"
	"github.com/icza/gowut/gwu"
	"log"
	//"math/rand"
	"os"
	//"strconv"
	"path/filepath"
	"time"
)

type app struct {
	models  map[string]*model  // label => model
	devices map[string]*device // id => device

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
		models:  map[string]*model{},
		devices: map[string]*device{},
		logger:  logger,
	}

	registerModelCiscoIOS(app.logger, app.models)

	return app
}

func main() {

	logger := log.New(os.Stdout, "", log.LstdFlags)

	jaz := newApp(logger)

	createDevice(jaz, "cisco-ios", "lab1", "localhost:2001", "telnet", "lab", "pass", "en")
	createDevice(jaz, "cisco-ios", "lab2", "localhost:2002", "ssh", "lab", "pass", "en")
	createDevice(jaz, "cisco-ios", "lab3", "localhost:2003", "telnet,ssh", "lab", "pass", "en")
	createDevice(jaz, "cisco-ios", "lab4", "localhost:2004", "ssh,telnet", "lab", "pass", "en")
	createDevice(jaz, "cisco-ios", "lab5", "localhost", "telnet", "lab", "pass", "en")
	createDevice(jaz, "cisco-ios", "lab6", "localhost", "ssh", "rat", "lab", "en")

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

	go scanDevices(jaz) // FIXME one-shot scan

	// Start GUI server
	if err := server.Start(); err != nil {
		jaz.logf("jazigo main: Cound not start GUI server: %s", err)
		return
	}
}

type fetchResult struct {
	model       string
	devId       string
	devHostPort string
	msg         string    // result error message
	code        int       // result error code
	begin       time.Time // begin timestamp
}

func scanDevices(jaz *app) {

	jaz.logf("scanDevices: starting")

	begin := time.Now()

	resultCh := make(chan fetchResult)

	baseDelay := 500 * time.Millisecond
	jaz.logf("scanDevices: non-hammering delay between captures: %d ms", baseDelay/time.Millisecond)

	wait := 0
	currDelay := time.Duration(0)

	for _, dev := range jaz.devices {
		go dev.fetch(jaz.logger, resultCh, currDelay) // per-device goroutine
		currDelay += baseDelay
		wait++
	}

	elapMax := 0 * time.Second
	elapMin := 24 * time.Hour

	for wait > 0 {
		r := <-resultCh
		wait--
		elap := time.Now().Sub(r.begin)
		jaz.logf("device result: %s %s %s msg=[%s] code=%d remain=%d elap=%s", r.model, r.devId, r.devHostPort, r.msg, r.code, wait, elap)
		if elap < elapMin {
			elapMin = elap
		}
		if elap > elapMax {
			elapMax = elap
		}
	}

	end := time.Now()
	elapsed := end.Sub(begin)
	deviceCount := len(jaz.devices)
	average := elapsed / time.Duration(deviceCount)

	jaz.logf("scanDevices: finished elapsed=%s devices=%d average=%s min=%s max=%s", elapsed, deviceCount, average, elapMin, elapMax)
}
