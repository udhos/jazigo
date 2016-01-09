package main

import (
	"fmt"
	"github.com/udhos/gowut/gwu"
	"log"
	//"math/rand"
	"os"
	//"strconv"
)

type SessHandler struct{}

func (h SessHandler) Created(s gwu.Session) {
	logger.Println("SESSION created:", s.Id())
	//buildLoginWin(s)
}

func (h SessHandler) Removed(s gwu.Session) {
	logger.Println("SESSION removed:", s.Id())
}

const appName = "jazigo"

var logger = log.New(os.Stdout, "", log.LstdFlags)

func main() {

	appAddr := "0.0.0.0:8080"
	serverName := fmt.Sprintf("%s application", appName)

	// Create GUI server
	server := gwu.NewServer(appName, appAddr)
	//folder := "./tls/"
	//server := gwu.NewServerTLS(appName, appAddr, folder+"cert.pem", folder+"key.pem")
	server.SetText(serverName)

	server.AddSessCreatorName("login", fmt.Sprintf("%s login window", appName))
	server.AddSHandler(SessHandler{})

	buildHomeWin(server)

	server.SetLogger(logger)

	// Start GUI server
	if err := server.Start(); err != nil {
		logger.Println("jazigo main: Cound not start GUI server:", err)
		return
	}
}

func buildHomeWin(s gwu.Session) {
	// Add home window
	win := gwu.NewWindow("home", fmt.Sprintf("%s home window", appName))
	l := gwu.NewLabel(fmt.Sprintf("%s home", appName))
	l.Style().SetFontWeight(gwu.FontWeightBold).SetFontSize("130%")
	win.Add(l)
	win.Add(gwu.NewLabel("Click on the button to login:"))
	b := gwu.NewButton("Login")
	b.AddEHandlerFunc(func(e gwu.Event) {
		e.ReloadWin("login")
	}, gwu.ETypeClick)
	win.Add(b)
	s.AddWin(win)
}
