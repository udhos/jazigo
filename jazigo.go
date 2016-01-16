package main

import (
	"fmt"
	"github.com/udhos/gowut/gwu"
	"log"
	//"math/rand"
	"os"
	//"strconv"
)

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

	buildHomeWin(server)
	buildLoginWin(server)

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

func buildLoginWin(s gwu.Session) {
	windowName := fmt.Sprintf("%s login window", appName)

	win := gwu.NewWindow("login", windowName)
	win.Style().SetFullSize()
	win.SetAlign(gwu.HACenter, gwu.VAMiddle)

	p := gwu.NewPanel()
	p.SetHAlign(gwu.HACenter)
	p.SetCellPadding(2)

	l := gwu.NewLabel(windowName)
	l.Style().SetFontWeight(gwu.FontWeightBold).SetFontSize("150%")
	p.Add(l)
	l = gwu.NewLabel("Login")
	l.Style().SetFontWeight(gwu.FontWeightBold).SetFontSize("130%")
	p.Add(l)
	p.CellFmt(l).Style().SetBorder2(1, gwu.BrdStyleDashed, gwu.ClrNavy)
	l = gwu.NewLabel("user/pass: admin/a")
	l.Style().SetFontSize("80%").SetFontStyle(gwu.FontStyleItalic)
	p.Add(l)

	errL := gwu.NewLabel("")
	errL.Style().SetColor(gwu.ClrRed)
	p.Add(errL)

	table := gwu.NewTable()
	table.SetCellPadding(2)
	table.EnsureSize(2, 2)
	table.Add(gwu.NewLabel("Username:"), 0, 0)
	tb := gwu.NewTextBox("")

	tb.Style().SetWidthPx(160)
	table.Add(tb, 0, 1)
	table.Add(gwu.NewLabel("Password:"), 1, 0)
	pb := gwu.NewPasswBox("")

	pb.Style().SetWidthPx(160)
	table.Add(pb, 1, 1)
	p.Add(table)
	b := gwu.NewButton("OK")

	p.Add(b)
	l = gwu.NewLabel("")
	p.Add(l)
	p.CellFmt(l).Style().SetHeightPx(200)

	loginHandler := func(e gwu.Event) {
		user := tb.Text()
		if loginAuth(user, pb.Text()) {
			//e.Session().RemoveWin(win) // Login win is removed, password will not be retrievable from the browser
			//buildPrivateWins(e.Session())
			// FIXME: Should clear username/password fields?
			newSession := e.NewSession()
			newSession.SetAttr("username", user)
			buildPrivateWins(newSession)
			e.ReloadWin("main")
		} else {
			e.SetFocusedComp(tb)
			errL.SetText("Invalid user name or password!")
			e.MarkDirty(errL)
		}
	}

	enterHandler := func(e gwu.Event) {
		if e.Type() == gwu.ETypeKeyPress && e.KeyCode() == gwu.KeyEnter {
			// enter key was pressed
			loginHandler(e)
		}
	}

	tb.AddEHandlerFunc(enterHandler, gwu.ETypeKeyPress)
	pb.AddEHandlerFunc(enterHandler, gwu.ETypeKeyPress)
	b.AddEHandlerFunc(loginHandler, gwu.ETypeClick)

	win.Add(p)
	win.SetFocusedCompId(tb.Id())

	s.AddWin(win)
}

func loginAuth(user, pass string) bool {
	return user == "admin" && pass == "a"
}

func buildPrivateWins(s gwu.Session) {
	// Create and build a window

	user := s.Attr("username").(string)
	addr := s.RemoteAddr()

	winName := fmt.Sprintf("%s main window - user=%s - address=%s", appName, user, addr)
	win := gwu.NewWindow("main", winName)

	win.Style().SetFullWidth()
	win.SetCellPadding(2)

	title := gwu.NewLabel(winName)
	win.Add(title)

	win.Add(gwu.NewLabel("click on this window to see updates"))

	win.AddEHandlerFunc(func(e gwu.Event) {
		win.Add(gwu.NewLabel(fmt.Sprintf("click - addr=%v", s.RemoteAddr())))
		e.MarkDirty(win)
	}, gwu.ETypeClick)

	s.AddWin(win)
}
