package main

import (
	"fmt"
	//"github.com/udhos/gowut/gwu"
	"github.com/icza/gowut/gwu"
	"log"
	//"math/rand"
	//"os"
	//"strconv"
)

const appName = "jazigo"

const hardUser = "a"
const hardPass = "a"

func newAccPanel(user string) gwu.Panel {
	ap := gwu.NewPanel()
	ap.Style().AddClass("account_panel")
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

func newWin(jaz *app, path, name string) gwu.Window {
	win := gwu.NewWindow(path, name)
	cssLink := fmt.Sprintf(`<link rel="stylesheet" type="text/css" href="%s">`, jaz.cssPath)
	win.AddHeadHtml(cssLink)
	log.Printf("window=[%s] attached CSS=[%s]", path, cssLink)
	return win
}

func buildHomeWin(jaz *app, s gwu.Session) {

	winName := fmt.Sprintf("%s home", appName)
	win := newWin(jaz, "home", winName)

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
	win := newWin(jaz, "login", winName)

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

	win := newWin(jaz, "logout", winName)

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

	win := newWin(jaz, "admin", winName)

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
