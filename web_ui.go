package main

import (
	"fmt"
	//"log"
	//"math/rand"
	//"os"
	//"strconv"
	"sort"
	"time"

	"github.com/icza/gowut/gwu"
	"github.com/udhos/jazigo/dev"
)

func newAccPanel(user string) gwu.Panel {
	ap := gwu.NewHorizontalPanel()
	ap.Style().AddClass("account_panel")

	ap.Add(gwu.NewLabel(fmt.Sprintf("%s %s", appName, appVersion)))

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
			jaz.logf("home win insert accPanel failed")
		}
	}

	if jaz.winAdmin != nil {
		if jaz.apAdmin != nil {
			admin := jaz.winAdmin.ById(jaz.apAdmin.Id())
			jaz.winAdmin.Remove(admin)
		}
		jaz.apAdmin = newAccPanel(user)
		if !jaz.winAdmin.Insert(jaz.apAdmin, 0) {
			jaz.logf("admin win insert accPanel failed")
		}
	}

	if jaz.winLogout != nil {
		if jaz.apLogout != nil {
			logout := jaz.winLogout.ById(jaz.apLogout.Id())
			jaz.winLogout.Remove(logout)
		}
		jaz.apLogout = newAccPanel(user)
		if !jaz.winLogout.Insert(jaz.apLogout, 0) {
			jaz.logf("logout win insert accPanel failed")
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
	jaz.logf("window=[%s] attached CSS=[%s]", path, cssLink)
	return win
}

type sortById struct {
	data []*dev.Device
}

func (s sortById) Len() int {
	return len(s.data)
}
func (s sortById) Swap(i, j int) {
	s.data[i], s.data[j] = s.data[j], s.data[i]
}
func (s sortById) Less(i, j int) bool {
	return s.data[i].Id < s.data[j].Id
}

func buildDeviceTable(jaz *app, t gwu.Table) {
	const COLS = 9

	t.Add(gwu.NewLabel("Model"), 0, 0)
	t.Add(gwu.NewLabel("Device"), 0, 1)
	t.Add(gwu.NewLabel("Host"), 0, 2)
	t.Add(gwu.NewLabel("Transport"), 0, 3)
	t.Add(gwu.NewLabel("Last Status"), 0, 4)
	t.Add(gwu.NewLabel("Last Try"), 0, 5)
	t.Add(gwu.NewLabel("Last Success"), 0, 6)
	t.Add(gwu.NewLabel("Holdtime"), 0, 7)
	t.Add(gwu.NewLabel("Run Now"), 0, 8)

	for j := 0; j < COLS; j++ {
		t.CellFmt(0, j).Style().AddClass("device_table_cell")
	}

	devList := jaz.table.ListDevices()
	sort.Sort(sortById{data: devList})

	now := time.Now()

	i := 1
	for _, d := range devList {
		labMod := gwu.NewLabel(d.Model())
		labId := gwu.NewLabel(d.Id)
		labHost := gwu.NewLabel(d.HostPort)
		labTransport := gwu.NewLabel(d.Transports)
		labLastStatus := gwu.NewLabel(fmt.Sprintf("%v", d.LastStatus()))
		labLastTry := gwu.NewLabel(timestampString(d.LastTry()))
		labLastSuccess := gwu.NewLabel(timestampString(d.LastSuccess()))
		h := d.Holdtime(now, jaz.cfg.Holdtime)
		if h < 0 {
			h = 0
		}
		labHoldtime := gwu.NewLabel(fmt.Sprintf("%v", h))

		buttonRun := gwu.NewButton("Run")
		id := d.Id
		buttonRun.AddEHandlerFunc(func(e gwu.Event) {
			runPriority(jaz, id)
		}, gwu.ETypeClick)

		t.Add(labMod, i, 0)
		t.Add(labId, i, 1)
		t.Add(labHost, i, 2)
		t.Add(labTransport, i, 3)
		t.Add(labLastStatus, i, 4)
		t.Add(labLastTry, i, 5)
		t.Add(labLastSuccess, i, 6)
		t.Add(labHoldtime, i, 7)
		t.Add(buttonRun, i, 8)

		for j := 0; j < COLS; j++ {
			t.CellFmt(i, j).Style().AddClass("device_table_cell")
		}

		i++
	}
}

func runPriority(jaz *app, id string) {
	jaz.logger.Printf("runPriority: device: %s", id)
	jaz.priority <- id
}

func buildHomeWin(jaz *app, s gwu.Session) {

	winName := fmt.Sprintf("%s home", appName)
	win := newWin(jaz, "home", winName)

	win.Add(jaz.apHome)

	l := gwu.NewLabel(winName)
	l.Style().SetFontWeight(gwu.FontWeightBold).SetFontSize("130%")
	win.Add(l)

	t := gwu.NewTable()
	t.Style().AddClass("device_table")

	refresh := func(e gwu.Event) {
		t.Clear() // clear out table contents
		buildDeviceTable(jaz, t)
		e.MarkDirty(t)
	}

	refreshButton := gwu.NewButton("Refresh")
	refreshButton.AddEHandlerFunc(func(e gwu.Event) {
		refresh(e)
	}, gwu.ETypeClick)
	win.Add(refreshButton)

	win.AddEHandlerFunc(func(e gwu.Event) {
		refresh(e)
	}, gwu.ETypeWinLoad)

	buildDeviceTable(jaz, t)

	win.Add(t)

	s.AddWin(win)

	jaz.winHome = win
}

func timestampString(ts time.Time) string {
	if ts.IsZero() {
		return "never"
	}
	return ts.String()
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
	l = gwu.NewLabel("FIXME: username must be equal to password")
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

		//jaz.logf("debug login user=[%s] pass=[%s] result=[%v]", user, pass, auth)

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
	return user == pass // loginAuth: FIXME WRITEME
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
