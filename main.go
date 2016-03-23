package main

import (
	"fmt"
	//"github.com/udhos/gowut/gwu"
	"github.com/icza/gowut/gwu"
	"log"
	//"math/rand"
	"os"
	//"strconv"
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
}

func main() {

	jaz := &app{
		models:  map[string]*model{},
		devices: map[string]*device{},
	}

	registerModelCiscoIOS(jaz.models)

	createDevice(jaz, "cisco-ios", "lab1", "localhost:2001", "tcp,ssh", "lab", "pass", "en")
	createDevice(jaz, "cisco-ios", "lab2", "localhost:2002", "tcp,ssh", "lab", "pass", "en")

	appAddr := "0.0.0.0:8080"
	serverName := fmt.Sprintf("%s application", appName)

	// Create GUI server
	server := gwu.NewServer(appName, appAddr)
	//folder := "./tls/"
	//server := gwu.NewServerTLS(appName, appAddr, folder+"cert.pem", folder+"key.pem")
	server.SetText(serverName)

	// create account panel
	jaz.apHome = newAccPanel("")
	jaz.apAdmin = newAccPanel("")
	jaz.apLogout = newAccPanel("")
	accountPanelUpdate(jaz, "")

	buildHomeWin(jaz, server)
	buildLoginWin(jaz, server)

	server.SetLogger(logger)

	// Start GUI server
	if err := server.Start(); err != nil {
		logger.Println("jazigo main: Cound not start GUI server:", err)
		return
	}
}
