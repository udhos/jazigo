package main

import (
	"fmt"
	//"log"
	//"math/rand"
	//"os"
	//"strconv"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/icza/gowut/gwu"
	"github.com/udhos/jazigo/conf"
	"github.com/udhos/jazigo/dev"
	"github.com/udhos/jazigo/store"
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

func deviceWinName(id string) string {
	return "device-" + id
}

func buildDeviceWindow(jaz *app, s gwu.Session, e gwu.Event, devId string) string {
	winName := deviceWinName(devId)
	win := s.WinByName(winName)
	if win != nil {
		return winName
	}
	winTitle := "Device: " + devId
	win = newWin(jaz, winName, winTitle)
	win.Add(gwu.NewLabel(winTitle))

	refreshButton := gwu.NewButton("Refresh")
	win.Add(refreshButton)

	panel := gwu.NewTabPanel()

	filesPanel := gwu.NewPanel()
	filesMsg := gwu.NewLabel("No error")
	filesTab := gwu.NewTable()
	filesPanel.Add(filesMsg)
	filesPanel.Add(filesTab)

	filesTab.Style().AddClass("device_files_table")

	propPanel := gwu.NewPanel()
	propButtonReset := gwu.NewButton("Reset")
	propButtonSave := gwu.NewButton("Save")
	propMsg := gwu.NewLabel("No error")
	propText := gwu.NewTextBox("Text Box")
	propText.SetRows(40)
	propText.SetCols(100)
	propPanel.Add(propButtonReset)
	propPanel.Add(propButtonSave)
	propPanel.Add(propMsg)
	propPanel.Add(propText)

	showPanel := gwu.NewPanel()

	panel.Add(gwu.NewLabel("Files"), filesPanel)
	panel.Add(gwu.NewLabel("Properties"), propPanel)
	panel.Add(gwu.NewLabel("View Config"), showPanel)
	panel.Add(gwu.NewLabel("Error Log"), gwu.NewLabel("Error Log"))

	const TAB_SHOW = 2 // index

	devPrefix := dev.DeviceFullPrefix(jaz.repositoryPath, devId)
	showFile, lastErr := store.FindLastConfig(devPrefix, jaz.logger)
	if lastErr != nil {
		jaz.logger.Printf("buildDeviceWindow: could find last config for device: %v", lastErr)
	}

	loadView := func(e gwu.Event, show string) {
		text := "Load from: " + show
		showPanel.Clear()
		showPanel.Add(gwu.NewLabel("Config: " + show))
		showBox := gwu.NewTextBox("")
		showBox.SetRows(40)
		showBox.SetCols(100)
		showBox.SetText(text)
		showPanel.Add(showBox)
		e.MarkDirty(panel)
	}

	loadView(e, showFile) // first run

	win.Add(panel)

	fileList := func(e gwu.Event) {
		prefix := dev.DeviceFullPrefix(jaz.repositoryPath, devId)
		dirname, matches, listErr := store.ListConfigSorted(prefix, true, jaz.logger)
		if listErr != nil {
			filesMsg.SetText(fmt.Sprintf("List files error: %v", listErr))
			e.MarkDirty(filesPanel)
			return
		}

		filesMsg.SetText(fmt.Sprintf("%d files", len(matches)))

		filesTab.Clear()

		const COLS = 3

		row := 0

		// header
		filesTab.Add(gwu.NewLabel("Download"), row, 0)
		filesTab.Add(gwu.NewLabel("View"), row, 1)
		filesTab.Add(gwu.NewLabel("Time"), row, 2)

		row++

		for _, m := range matches {
			path := filepath.Join(dirname, m)
			timeStr := "unknown"
			f, openErr := os.Open(path)
			if openErr != nil {
				timeStr += fmt.Sprintf("(could not open file: %v)", openErr)
			}
			info, statErr := f.Stat()
			if statErr == nil {
				timeStr = info.ModTime().String()
			} else {
				timeStr += fmt.Sprintf("(could not stat file: %v)", statErr)
			}

			filePath := fmt.Sprintf("/%s/%s/%s/%s", appName, jaz.repoPath, devId, m)
			devLink := gwu.NewLink(m, filePath)

			buttonView := gwu.NewButton("Open")
			show := dev.DeviceFullPath(jaz.repoPath, devId, m)
			buttonView.AddEHandlerFunc(func(e gwu.Event) {
				loadView(e, show)
				panel.SetSelected(TAB_SHOW)
			}, gwu.ETypeClick)

			filesTab.Add(devLink, row, 0)
			filesTab.Add(buttonView, row, 1)
			filesTab.Add(gwu.NewLabel(timeStr), row, 2)

			for j := 0; j < COLS; j++ {
				filesTab.CellFmt(row, j).Style().AddClass("device_files_cell")
			}

			row++
		}
	}

	resetProp := func(e gwu.Event) {
		d, getErr := jaz.table.GetDevice(devId)
		if getErr != nil {
			propMsg.SetText(fmt.Sprintf("Get device error: %v", getErr))
			e.MarkDirty(propPanel)
			return
		}

		b, dumpErr := d.DevConfig.Dump()
		if dumpErr != nil {
			propMsg.SetText(fmt.Sprintf("Device dump error: %v", dumpErr))
			e.MarkDirty(propPanel)
			return
		}

		propText.SetText(string(b))

		e.MarkDirty(propPanel)
	}

	refresh := func(e gwu.Event) {
		fileList(e)  // build file list
		resetProp(e) // // build file properties
		e.MarkDirty(win)
	}

	refresh(e) // first run

	propButtonReset.AddEHandlerFunc(resetProp, gwu.ETypeClick)

	propButtonSave.AddEHandlerFunc(func(e gwu.Event) {
		str := propText.Text()

		defer e.MarkDirty(propPanel)

		c, parseErr := conf.NewDeviceFromString(str)
		if parseErr != nil {
			propMsg.SetText(fmt.Sprintf("Parse device error: %v", parseErr))
			return
		}

		d, getErr := jaz.table.GetDevice(devId)
		if getErr != nil {
			propMsg.SetText(fmt.Sprintf("Get device error: %v", getErr))
			return
		}

		d.DevConfig = *c

		updateErr := jaz.table.UpdateDevice(d)
		if updateErr != nil {
			propMsg.SetText(fmt.Sprintf("Update error: %v", updateErr))
			return
		}

		saveConfig(jaz)

		propMsg.SetText("Device updated.")

	}, gwu.ETypeClick)

	refreshButton.AddEHandlerFunc(refresh, gwu.ETypeClick)

	win.AddEHandlerFunc(refresh, gwu.ETypeWinLoad)

	s.AddWin(win)

	return winName
}

func buildDeviceTable(jaz *app, s gwu.Session, t gwu.Table) {
	const COLS = 9

	row := 0 // filter
	filterModel := gwu.NewTextBox(jaz.filterModel)
	filterId := gwu.NewTextBox(jaz.filterId)
	filterHost := gwu.NewTextBox(jaz.filterHost)

	filterModel.AddSyncOnETypes(gwu.ETypeKeyUp) // synchronize values during editing (while you type in characters)
	filterId.AddSyncOnETypes(gwu.ETypeKeyUp)    // synchronize values during editing (while you type in characters)
	filterHost.AddSyncOnETypes(gwu.ETypeKeyUp)  // synchronize values during editing (while you type in characters)

	filterModel.AddEHandlerFunc(func(e gwu.Event) {
		jaz.filterModel = filterModel.Text()
		refreshDeviceTable(jaz, s, t, e)
	}, gwu.ETypeChange)

	filterId.AddEHandlerFunc(func(e gwu.Event) {
		jaz.filterId = filterId.Text()
		refreshDeviceTable(jaz, s, t, e)
	}, gwu.ETypeChange)

	filterHost.AddEHandlerFunc(func(e gwu.Event) {
		jaz.filterHost = filterHost.Text()
		refreshDeviceTable(jaz, s, t, e)
	}, gwu.ETypeChange)

	t.Add(filterModel, row, 0)
	t.Add(filterId, row, 1)
	t.Add(filterHost, row, 2)
	t.Add(gwu.NewLabel(""), row, 3)
	t.Add(gwu.NewLabel(""), row, 4)
	t.Add(gwu.NewLabel(""), row, 5)
	t.Add(gwu.NewLabel(""), row, 6)
	t.Add(gwu.NewLabel(""), row, 7)
	t.Add(gwu.NewLabel(""), row, 8)

	row = 1 // header
	t.Add(gwu.NewLabel("Model"), row, 0)
	t.Add(gwu.NewLabel("Device"), row, 1)
	t.Add(gwu.NewLabel("Host"), row, 2)
	t.Add(gwu.NewLabel("Transport"), row, 3)
	t.Add(gwu.NewLabel("Last Status"), row, 4)
	t.Add(gwu.NewLabel("Last Try"), row, 5)
	t.Add(gwu.NewLabel("Last Success"), row, 6)
	t.Add(gwu.NewLabel("Holdtime"), row, 7)
	t.Add(gwu.NewLabel("Run Now"), row, 8)

	devList := jaz.table.ListDevices()
	sort.Sort(sortById{data: devList})

	now := time.Now()

	options := jaz.options.Get()

	row = 2
	for _, d := range devList {

		if !strings.Contains(d.Model(), filterModel.Text()) {
			continue
		}
		if !strings.Contains(d.Id, filterId.Text()) {
			continue
		}
		if !strings.Contains(d.HostPort, filterHost.Text()) {
			continue
		}

		labMod := gwu.NewLabel(d.Model())

		devWin := "device-" + d.Id
		labId := gwu.NewLink(d.Id, "/"+appName+"/"+devWin)
		devId := d.Id // get dev id for closure below
		labId.AddEHandlerFunc(func(e gwu.Event) {
			buildDeviceWindow(jaz, s, e, devId)
		}, gwu.ETypeClick)

		labHost := gwu.NewLabel(d.HostPort)
		labTransport := gwu.NewLabel(d.Transports)
		labLastStatus := gwu.NewLabel(fmt.Sprintf("%v", d.LastStatus()))
		labLastTry := gwu.NewLabel(timestampString(d.LastTry()))
		labLastSuccess := gwu.NewLabel(timestampString(d.LastSuccess()))
		h := d.Holdtime(now, options.Holdtime)
		if h < 0 {
			h = 0
		}
		labHoldtime := gwu.NewLabel(fmt.Sprintf("%v", h))

		buttonRun := gwu.NewButton("Run")
		id := d.Id
		buttonRun.AddEHandlerFunc(func(e gwu.Event) {
			// run in a goroutine to not block the UI on channel write
			go runPriority(jaz, id)
		}, gwu.ETypeClick)

		t.Add(labMod, row, 0)
		t.Add(labId, row, 1)
		t.Add(labHost, row, 2)
		t.Add(labTransport, row, 3)
		t.Add(labLastStatus, row, 4)
		t.Add(labLastTry, row, 5)
		t.Add(labLastSuccess, row, 6)
		t.Add(labHoldtime, row, 7)
		t.Add(buttonRun, row, 8)

		row++
	}

	for r := 0; r < row; r++ {
		for j := 0; j < COLS; j++ {
			t.CellFmt(r, j).Style().AddClass("device_table_cell")
		}
	}
}

func runPriority(jaz *app, id string) {
	jaz.logger.Printf("runPriority: device: %s", id)
	jaz.priority <- id
}

func refreshDeviceTable(jaz *app, s gwu.Session, t gwu.Table, e gwu.Event) {
	t.Clear() // clear out table contents
	buildDeviceTable(jaz, s, t)
	e.MarkDirty(t)
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
		refreshDeviceTable(jaz, s, t, e)
	}

	createDevPanel := buildCreateDevPanel(jaz, s, refresh)

	refreshButton := gwu.NewButton("Refresh")
	refreshButton.AddEHandlerFunc(refresh, gwu.ETypeClick)
	win.Add(refreshButton)

	win.AddEHandlerFunc(refresh, gwu.ETypeWinLoad)

	win.Add(createDevPanel)

	buildDeviceTable(jaz, s, t)

	win.Add(t)

	s.AddWin(win)

	jaz.winHome = win
}

func buildCreateDevPanel(jaz *app, s gwu.Session, refresh func(gwu.Event)) gwu.Panel {
	createDevPanel := gwu.NewPanel()
	createPanel := gwu.NewHorizontalPanel()
	msg := gwu.NewLabel("Message")
	createDevPanel.Add(createPanel)
	createDevPanel.Add(msg)

	panelModel := gwu.NewPanel()
	panelId := gwu.NewPanel()
	panelHost := gwu.NewPanel()
	panelTransport := gwu.NewPanel()
	panelUser := gwu.NewPanel()
	panelPass := gwu.NewPanel()
	panelEnable := gwu.NewPanel()
	button := gwu.NewButton("Create")

	labelModel := gwu.NewLabel("Model")
	labelId := gwu.NewLabel("Id")
	labelHost := gwu.NewLabel("Host")
	labelTransport := gwu.NewLabel("Transports")
	labelUser := gwu.NewLabel("User")
	labelPass := gwu.NewLabel("Pass")
	labelEnable := gwu.NewLabel("Enable")

	autoIdPrefix := "auto"

	textModel := gwu.NewTextBox("cisco-ios")
	textId := gwu.NewTextBox(autoIdPrefix)
	textHost := gwu.NewTextBox("")
	textTransport := gwu.NewTextBox("ssh,telnet")
	textUser := gwu.NewTextBox("")
	textPass := gwu.NewTextBox("")
	textEnable := gwu.NewTextBox("")

	panelModel.Add(labelModel)
	panelModel.Add(textModel)
	panelId.Add(labelId)
	panelId.Add(textId)
	panelHost.Add(labelHost)
	panelHost.Add(textHost)
	panelTransport.Add(labelTransport)
	panelTransport.Add(textTransport)
	panelUser.Add(labelUser)
	panelUser.Add(textUser)
	panelPass.Add(labelPass)
	panelPass.Add(textPass)
	panelEnable.Add(labelEnable)
	panelEnable.Add(textEnable)

	createPanel.Add(panelModel)
	createPanel.Add(panelId)
	createPanel.Add(panelHost)
	createPanel.Add(panelTransport)
	createPanel.Add(panelUser)
	createPanel.Add(panelPass)
	createPanel.Add(panelEnable)
	createPanel.Add(button)

	createAutoId := func() {
		if strings.HasPrefix(textId.Text(), autoIdPrefix) {
			textId.SetText(jaz.table.FindDeviceFreeId(autoIdPrefix))
		}
	}

	button.AddEHandlerFunc(func(e gwu.Event) {
		id := textId.Text()

		if id == autoIdPrefix {
			id = jaz.table.FindDeviceFreeId(autoIdPrefix)
		}

		/*
			_, err1 := jaz.table.GetDevice(id)
			if err1 == nil {
				msg.SetText("Device ID already exists: " + id)
				e.MarkDirty(createDevPanel)
				return
			}
		*/
		if createErr := dev.CreateDevice(jaz.table, jaz.logger, textModel.Text(), id, textHost.Text(), textTransport.Text(), textUser.Text(), textPass.Text(), textEnable.Text(), false); createErr != nil {
			msg.SetText("Could not create device: " + createErr.Error())
			e.MarkDirty(createDevPanel)
			return
		}
		/*
			_, err2 := jaz.table.GetDevice(id)
			if err2 != nil {
				msg.SetText("Could not create device with ID: " + id)
				e.MarkDirty(createDevPanel)
				return
			}
		*/

		saveConfig(jaz)

		createAutoId() // prepare next auto id
		e.MarkDirty(textId)

		msg.SetText("Device created: " + id)
		e.MarkDirty(createDevPanel) // redraw msg
		refresh(e)                  // redraw device table
	}, gwu.ETypeClick)

	createAutoId() // first call

	return createDevPanel
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
