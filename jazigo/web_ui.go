package main

import (
	"fmt"
	//"log"
	//"math/rand"
	//"os"
	//"strconv"
	"io/ioutil"
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

func buildDeviceWindow(jaz *app, e gwu.Event, devId string) string {
	winName := deviceWinName(devId)
	s := e.Session()
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

	panel.Add(gwu.NewLabel("Files"), filesPanel)                    // tab 0
	panel.Add(gwu.NewLabel("View Config"), showPanel)               // tab 1
	panel.Add(gwu.NewLabel("Properties"), propPanel)                // tab 2
	panel.Add(gwu.NewLabel("Error Log"), gwu.NewLabel("Error Log")) // tab 3

	const TAB_SHOW = 1 // index

	devPrefix := dev.DeviceFullPrefix(jaz.repositoryPath, devId)
	showFile, lastErr := store.FindLastConfig(devPrefix, jaz.logger)
	if lastErr != nil {
		jaz.logger.Printf("buildDeviceWindow: could find last config for device: %v", lastErr)
	}

	loadView := func(e gwu.Event, show string) {

		showPanel.Clear()

		showPanel.Add(gwu.NewLabel("File: " + show))

		input, openErr := os.Open(show)
		if openErr != nil {
			showPanel.Add(gwu.NewLabel(fmt.Sprintf("Could not open '%s': %v", show, openErr)))
		}

		jaz.logger.Printf("FIXME web_ui loadView: limit number of lines read from file")
		b, readErr := ioutil.ReadAll(input)
		if readErr != nil {
			showPanel.Add(gwu.NewLabel(fmt.Sprintf("Could not read '%s': %v", show, readErr)))
		}

		text := string(b)

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
			show := dev.DeviceFullPath(jaz.repositoryPath, devId, m)
			buttonView.AddEHandlerFunc(func(e gwu.Event) {
				loadView(e, show)
				panel.SetSelected(TAB_SHOW)
			}, gwu.ETypeClick)

			filesTab.Add(devLink, row, 0)
			filesTab.Add(buttonView, row, 1)
			filesTab.Add(gwu.NewLabel(timeStr), row, 2)

			row++
		}

		for r := 0; r < row; r++ {
			for j := 0; j < COLS; j++ {
				filesTab.CellFmt(r, j).Style().AddClass("device_files_cell")
			}
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
		propButtonSave.SetEnabled(userIsLogged(e.Session()))
		fileList(e)  // build file list
		resetProp(e) // build file properties
		e.MarkDirty(win)
	}

	refresh(e) // first run

	propButtonReset.AddEHandlerFunc(resetProp, gwu.ETypeClick)

	propButtonSave.AddEHandlerFunc(func(e gwu.Event) {

		defer e.MarkDirty(propPanel)

		if !userIsLogged(e.Session()) {
			return // refuse to save
		}

		str := propText.Text()

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

		c.LastChange.From = eventRemoteAddress(e)
		c.LastChange.By = sessionUsername(e.Session())
		c.LastChange.When = time.Now()

		d.DevConfig = *c

		updateErr := jaz.table.UpdateDevice(d)
		if updateErr != nil {
			propMsg.SetText(fmt.Sprintf("Update error: %v", updateErr))
			return
		}

		saveConfig(jaz, c.LastChange)

		resetProp(e)

		propMsg.SetText("Device updated.")

	}, gwu.ETypeClick)

	refreshButton.AddEHandlerFunc(refresh, gwu.ETypeClick)

	win.AddEHandlerFunc(refresh, gwu.ETypeWinLoad)

	s.AddWin(win)

	return winName
}

func buildDeviceTable(jaz *app, s gwu.Session, t gwu.Table /* , killExistingDevWins bool*/) {
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
		refreshDeviceTable(jaz, t, e)
	}, gwu.ETypeChange)

	filterId.AddEHandlerFunc(func(e gwu.Event) {
		jaz.filterId = filterId.Text()
		refreshDeviceTable(jaz, t, e)
	}, gwu.ETypeChange)

	filterHost.AddEHandlerFunc(func(e gwu.Event) {
		jaz.filterHost = filterHost.Text()
		refreshDeviceTable(jaz, t, e)
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

		devWin := deviceWinName(d.Id)
		labId := gwu.NewLink(d.Id, "/"+appName+"/"+devWin)

		devId := d.Id // get dev id for closure below
		labId.AddEHandlerFunc(func(e gwu.Event) {
			buildDeviceWindow(jaz, e, devId)
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
	if jaz.oldScheduler {
		jaz.priority <- id
	} else {

		_, clearErr := dev.ClearDeviceStatus(jaz.table, id, jaz.logger, jaz.options.Get().Holdtime)
		if clearErr != nil {
			jaz.logger.Printf("runPriority: clear device %s status error: %v", id, clearErr)
			return
		}

		jaz.requestChan <- dev.FetchRequest{Id: id}
	}
}

func refreshDeviceTable(jaz *app, t gwu.Table, e gwu.Event) {
	t.Clear() // clear out table contents
	buildDeviceTable(jaz, e.Session(), t /*, false */)
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

	createButton := gwu.NewButton("Create")

	refresh := func(e gwu.Event) {
		createButton.SetEnabled(userIsLogged(e.Session()))
		e.MarkDirty(createButton)
		refreshDeviceTable(jaz, t, e)
	}

	createDevPanel := buildCreateDevPanel(jaz, s, refresh, createButton)

	refreshButton := gwu.NewButton("Refresh")
	refreshButton.AddEHandlerFunc(refresh, gwu.ETypeClick)
	win.Add(refreshButton)

	win.AddEHandlerFunc(refresh, gwu.ETypeWinLoad)

	win.Add(createDevPanel)

	win.Add(gwu.NewLabel("Hint: fill in text boxes below to select matching subset of devices."))

	buildDeviceTable(jaz, s, t /*, true*/)

	win.Add(t)

	s.AddWin(win)

	jaz.winHome = win
}

func buildCreateDevPanel(jaz *app, s gwu.Session, refresh func(gwu.Event), createButton gwu.Button) gwu.Panel {
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
	button := createButton

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

		if !userIsLogged(e.Session()) {
			return // refuse to create
		}

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

		change := conf.Change{
			From: eventRemoteAddress(e),
			By:   sessionUsername(e.Session()),
			When: time.Now(),
		}

		if createErr := dev.CreateDevice(jaz.table, jaz.logger, textModel.Text(), id, textHost.Text(), textTransport.Text(), textUser.Text(), textPass.Text(), textEnable.Text(), false, &change); createErr != nil {
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

		saveConfig(jaz, change)

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

			// replace private session, if any
			if e.Session().Private() {
				jaz.logger.Printf("removing existing PRIVATE session")
				e.RemoveSess()
			}

			newSession := e.NewSession()
			newSession.SetAttr("username", user)

			/*
				remoteAddr := eventRemoteAddress(e)
					if hrr, ok := e.(gwu.HasRequestResponse); ok {
						req := hrr.Request()
						remoteAddr = req.RemoteAddr
					}
			*/

			buildPrivateWins(jaz, newSession)

			accountPanelUpdateEvent(jaz, user, e)

			e.ReloadWin("home")
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
	if user == "" {
		return false
	}
	return user == pass // loginAuth: FIXME WRITEME
}

func buildPublicWins(jaz *app, s gwu.Session) {

	if s.Private() {
		jaz.logger.Printf("buildPublicWins: ignoring call within PRIVATE session")
		return
	}

	// create account panel
	jaz.apHome = newAccPanel("")
	jaz.apAdmin = newAccPanel("")
	jaz.apLogout = newAccPanel("")
	accountPanelUpdate(jaz, "")

	buildLoginWin(jaz, s)
	buildAdminWin(jaz, s)
	buildHomeWin(jaz, s)
}

func buildPrivateWins(jaz *app, s gwu.Session) {
	if !s.Private() {
		jaz.logger.Printf("buildPrivateWins: ignoring call within PUBLIC session")
		return
	}

	buildLogoutWin(jaz, s)
	buildAdminWin(jaz, s) // this is needed for access to admin win within PRIVATE session
	buildHomeWin(jaz, s)  // this is needed for access to home win within PRIVATE session
}

func sessionUsername(s gwu.Session) string {
	if !s.Private() {
		return ""
	}
	user := s.Attr("username")
	if user == nil {
		return ""
	}
	str, isStr := user.(string)
	if !isStr {
		return ""
	}
	return str
}

func eventRemoteAddress(e gwu.Event) string {
	if hrr, ok := e.(gwu.HasRequestResponse); ok {
		req := hrr.Request()
		return req.RemoteAddr
	}
	return "(remoteAddress?)"
}

func userIsLogged(s gwu.Session) bool {
	return sessionUsername(s) != ""
}

func buildLogoutWin(jaz *app, s gwu.Session) {
	winName := fmt.Sprintf("%s logout", appName)

	win := newWin(jaz, "logout", winName)

	win.Style().SetFullWidth()
	win.SetCellPadding(2)

	win.Add(jaz.apLogout)

	p := gwu.NewPanel()
	p.SetCellPadding(2)

	logoutButton := gwu.NewButton("Logout")
	logoutButton.AddEHandlerFunc(func(e gwu.Event) {
		if !e.Session().Private() {
			return // ignore button for public session if any
		}

		e.RemoveSess()
		e.ReloadWin("/")
	}, gwu.ETypeClick)

	p.Add(logoutButton)

	win.Add(p)
	s.AddWin(win)

	jaz.winLogout = win
}

func buildAdminWin(jaz *app, s gwu.Session) {

	winName := fmt.Sprintf("%s admin", appName)

	win := newWin(jaz, "admin", winName)

	win.Style().SetFullWidth()
	win.SetCellPadding(2)

	win.Add(jaz.apAdmin)

	settingsPanel := gwu.NewPanel()
	settingsButtonRefresh := gwu.NewButton("Refresh")
	settingsButtonSave := gwu.NewButton("Save")
	settingsMsg := gwu.NewLabel("No error")
	settingsFile := gwu.NewLabel("Save file")
	settingsText := gwu.NewTextBox("Text Box")
	settingsText.SetRows(20)
	settingsText.SetCols(70)
	settingsPanel.Add(gwu.NewLabel("Global Settings"))
	settingsPanel.Add(settingsButtonRefresh)
	settingsPanel.Add(settingsButtonSave)
	settingsPanel.Add(settingsMsg)
	settingsPanel.Add(settingsFile)
	settingsPanel.Add(settingsText)

	settingsButtonSave.SetEnabled(userIsLogged(s))

	load := func() {

		showFile, lastErr := store.FindLastConfig(jaz.configPathPrefix, jaz.logger)
		if lastErr != nil {
			jaz.logger.Printf("buildAdminWin: could find last config: %v", lastErr)
			showFile = fmt.Sprintf("Could not find last config file: %v", lastErr)
		}

		settingsFile.SetText(fmt.Sprintf("File: %s", showFile))

		opt := jaz.options.Get()
		b, dumpErr := opt.Dump()
		if dumpErr != nil {
			settingsText.SetText(fmt.Sprintf("Could not get settings: %v", dumpErr))
			return
		}

		settingsText.SetText(string(b))
	}

	load() // first run

	refresh := func(e gwu.Event) {
		settingsButtonSave.SetEnabled(userIsLogged(e.Session()))

		defer e.MarkDirty(settingsPanel)

		load()
	}

	settingsButtonRefresh.AddEHandlerFunc(refresh, gwu.ETypeClick)

	settingsButtonSave.AddEHandlerFunc(func(e gwu.Event) {

		if !userIsLogged(e.Session()) {
			return // refuse to save
		}

		defer e.MarkDirty(settingsPanel)

		str := settingsText.Text()

		opt, parseErr := conf.NewAppConfigFromString(str)
		if parseErr != nil {
			settingsMsg.SetText(fmt.Sprintf("Parsing error: %v", parseErr))
		}

		opt.LastChange.From = eventRemoteAddress(e)
		opt.LastChange.By = sessionUsername(e.Session())
		opt.LastChange.When = time.Now()
		jaz.options.Set(opt)

		saveConfig(jaz, opt.LastChange)

		refresh(e)

		settingsMsg.SetText("Saved.")

	}, gwu.ETypeClick)

	win.Add(settingsPanel)

	win.AddEHandlerFunc(refresh, gwu.ETypeWinLoad)

	s.AddWin(win)

	jaz.winAdmin = win
}
