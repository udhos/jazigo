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

const appName = "jazigo"

var logger = log.New(os.Stdout, "", log.LstdFlags)

const hardUser = "a"
const hardPass = "a"

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

func newAccPanel(user string) gwu.Panel {
	ap := gwu.NewPanel()
	if user == "" {
		// guest user

		// create login button
		b := gwu.NewButton("Login")
		b.AddEHandlerFunc(func(e gwu.Event) {
			e.ReloadWin("login")
		}, gwu.ETypeClick)
		ap.Add(b)

	} else {

		// logged user
		l := gwu.NewLabel(fmt.Sprintf("username=[%s]", user))
		ap.Add(l)
	}
	return ap
}

func accountPanelUpdate(jaz *app, user string) {

	if jaz.winHome != nil {
		if jaz.apHome != nil {
			home := jaz.winHome.ById(jaz.apHome.Id())
			jaz.winHome.Remove(home)
		}
		jaz.apHome = newAccPanel(user)
		if !jaz.winHome.Insert(jaz.apHome, 0) {
			log.Printf("home win insert accPanel failed")
		}
	}

	if jaz.winAdmin != nil {
		if jaz.apAdmin != nil {
			admin := jaz.winAdmin.ById(jaz.apAdmin.Id())
			jaz.winAdmin.Remove(admin)
		}
		jaz.apAdmin = newAccPanel(user)
		if !jaz.winAdmin.Insert(jaz.apAdmin, 0) {
			log.Printf("admin win insert accPanel failed")
		}
		log.Printf("XXX admin updated")
	}

	if jaz.winLogout != nil {
		if jaz.apLogout != nil {
			logout := jaz.winLogout.ById(jaz.apLogout.Id())
			jaz.winLogout.Remove(logout)
		}
		jaz.apLogout = newAccPanel(user)
		if !jaz.winLogout.Insert(jaz.apLogout, 0) {
			log.Printf("logout win insert accPanel failed")
		}
	}

}

func accountPanelUpdateEvent(jaz *app, user string, e gwu.Event) {
	accountPanelUpdate(jaz, user)

	if jaz.winHome != nil {
		e.MarkDirty(jaz.winHome)
	}

	if jaz.winAdmin != nil {
		e.MarkDirty(jaz.winAdmin)
	}

	if jaz.winLogout != nil {
		e.MarkDirty(jaz.winLogout)
	}
}

func main() {

	jaz := &app{
		models:  map[string]*model{},
		devices: map[string]*device{},
	}

	registerModelCiscoIOS(jaz.models)

	createDevice(jaz, "cisco-ios", "lab1", "localhost:2001", "telnet,ssh", "lab", "pass", "en")
	createDevice(jaz, "cisco-ios", "lab2", "localhost:2002", "telnet,ssh", "lab", "pass", "en")

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

func buildHomeWin(jaz *app, s gwu.Session) {

	winName := fmt.Sprintf("%s home", appName)
	win := gwu.NewWindow("home", winName)

	win.Add(jaz.apHome)

	l := gwu.NewLabel(winName)
	l.Style().SetFontWeight(gwu.FontWeightBold).SetFontSize("130%")
	win.Add(l)

	/*
		win.Add(gwu.NewLabel("Click on the button to login:"))
		b := gwu.NewButton("Login")
		b.AddEHandlerFunc(func(e gwu.Event) {
			e.ReloadWin("login")
		}, gwu.ETypeClick)
		win.Add(b)
	*/

	s.AddWin(win)

	jaz.winHome = win
}

func buildLoginWin(jaz *app, s gwu.Session) {

	winName := fmt.Sprintf("%s login", appName)
	win := gwu.NewWindow("login", winName)

	win.Style().SetFullSize()
	win.SetAlign(gwu.HACenter, gwu.VAMiddle)

	p := gwu.NewPanel()
	p.SetHAlign(gwu.HACenter)
	p.SetCellPadding(2)

	l := gwu.NewLabel(winName)
	l.Style().SetFontWeight(gwu.FontWeightBold).SetFontSize("150%")
	p.Add(l)
	l = gwu.NewLabel("Login")
	l.Style().SetFontWeight(gwu.FontWeightBold).SetFontSize("130%")
	p.Add(l)
	p.CellFmt(l).Style().SetBorder2(1, gwu.BrdStyleDashed, gwu.ClrNavy)
	l = gwu.NewLabel(fmt.Sprintf("user/pass: %s/%s", hardUser, hardPass))
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
	tb.AddSyncOnETypes(gwu.ETypeKeyUp) // synchronize values during editing (while you type in characters)

	tb.Style().SetWidthPx(160)
	table.Add(tb, 0, 1)
	table.Add(gwu.NewLabel("Password:"), 1, 0)
	pb := gwu.NewPasswBox("")
	pb.AddSyncOnETypes(gwu.ETypeKeyUp) // synchronize values during editing (while you type in characters)

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
		pass := pb.Text()
		auth := loginAuth(user, pass)

		//logger.Printf("debug login user=[%s] pass=[%s] result=[%v]", user, pass, auth)

		if auth {

			// Clear username/password fields
			tb.SetText("")
			pb.SetText("")

			// Clear error message
			errL.SetText("")
			e.MarkDirty(errL)

			newSession := e.NewSession()
			newSession.SetAttr("username", user)

			remoteAddr := "(remoteAddr?)"
			if hrr, ok := e.(gwu.HasRequestResponse); ok {
				req := hrr.Request()
				remoteAddr = req.RemoteAddr
			}

			buildPrivateWins(jaz, newSession, remoteAddr)

			accountPanelUpdateEvent(jaz, user, e)

			e.ReloadWin("admin")
		} else {
			//e.SetFocusedComp(tb)
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
	return user == hardUser && pass == hardPass
}

func buildPrivateWins(jaz *app, s gwu.Session, remoteAddr string) {
	user := s.Attr("username").(string)

	buildLogoutWin(jaz, s, user, remoteAddr)
	buildAdminWin(jaz, s, user, remoteAddr)
}

func buildLogoutWin(jaz *app, s gwu.Session, user, remoteAddr string) {
	winName := fmt.Sprintf("%s logout", appName)
	winHeader := fmt.Sprintf("%s - user=%s - address=%s", winName, user, remoteAddr)

	win := gwu.NewWindow("logout", winName)
	win.Style().SetFullWidth()
	win.SetCellPadding(2)

	win.Add(jaz.apLogout)

	title := gwu.NewLabel(winHeader)
	win.Add(title)

	p := gwu.NewPanel()
	p.SetCellPadding(2)

	logoutButton := gwu.NewButton("Logout")

	p.Add(logoutButton)

	win.Add(p)
	s.AddWin(win)

	jaz.winLogout = win
}

func buildAdminWin(jaz *app, s gwu.Session, user, remoteAddr string) {
	winName := fmt.Sprintf("%s admin", appName)
	winHeader := fmt.Sprintf("%s - user=%s - address=%s", winName, user, remoteAddr)

	win := gwu.NewWindow("admin", winName)
	win.Style().SetFullWidth()
	win.SetCellPadding(2)

	win.Add(jaz.apAdmin)

	title := gwu.NewLabel(winHeader)
	win.Add(title)

	win.Add(gwu.NewLabel("click on this window to see updates"))

	win.AddEHandlerFunc(func(e gwu.Event) {

		if hrr, ok := e.(gwu.HasRequestResponse); ok {
			req := hrr.Request()
			remoteAddr = req.RemoteAddr
		}

		win.Add(gwu.NewLabel(fmt.Sprintf("click - addr=%v", remoteAddr)))
		e.MarkDirty(win)
	}, gwu.ETypeClick)

	s.AddWin(win)

	jaz.winAdmin = win
}
