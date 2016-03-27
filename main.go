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

var logger = log.New(os.Stdout, "", log.LstdFlags)

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
}

func main() {

	jaz := &app{
		models:  map[string]*model{},
		devices: map[string]*device{},
	}

	registerModelCiscoIOS(jaz.models)

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

	logger.Printf("static dir: path=[%s] mapped to dir=[%s]", staticPathFull, staticDir)

	jaz.cssPath = fmt.Sprintf("%s/jazigo.css", staticPathFull)
	logger.Printf("css path: %s", jaz.cssPath)

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
		logger.Println("jazigo main: Cound not start GUI server:", err)
		return
	}
}

type fetchResult struct {
	model       string
	devId       string
	devHostPort string
	msg         string // result error message
	code        int    // result error code
}

func scanDevices(jaz *app) {

	logger.Printf("scanDevices: starting")

	begin := time.Now()

	resultCh := make(chan fetchResult)

	baseDelay := 500 * time.Millisecond
	logger.Printf("scanDevices: non-hammering delay between captures: %d ms", baseDelay/time.Millisecond)

	wait := 0
	currDelay := time.Duration(0)

	for _, dev := range jaz.devices {
		go dev.fetch(resultCh, currDelay)
		currDelay += baseDelay
		wait++
	}

	for wait > 0 {
		r := <-resultCh
		wait--
		logger.Printf("device result: %s %s %s msg=[%s] code=%d remain=%d", r.model, r.devId, r.devHostPort, r.msg, r.code, wait)
	}

	end := time.Now()
	elapsed := end.Sub(begin)
	deviceCount := len(jaz.devices)
	average := elapsed / time.Duration(deviceCount)

	logger.Printf("scanDevices: finished elapsed=%s devices=%d average=%s", elapsed.String(), deviceCount, average.String())

}
