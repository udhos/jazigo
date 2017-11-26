package main

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/icza/gowut/gwu"
	"github.com/udhos/difflib"
	"github.com/udhos/jazigo/conf"
	"github.com/udhos/jazigo/dev"
	"github.com/udhos/jazigo/store"
)

func newAccPanel(user, staticPath string) gwu.Panel {
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

	// link to github repo
	imageGithub := gwu.NewImage("Forkme", staticPath+"/GitHub-Mark-32px.png")
	linkGithub := gwu.NewLink("", "https://github.com/udhos/jazigo")
	linkGithub.SetComp(imageGithub)
	ap.Add(linkGithub)

	return ap
}

func accountPanelUpdate(jaz *app, user string) {

	if jaz.winHome != nil {
		if jaz.apHome != nil {
			home := jaz.winHome.ByID(jaz.apHome.ID())
			jaz.winHome.Remove(home)
		}
		jaz.apHome = newAccPanel(user, jaz.staticPath)
		if !jaz.winHome.Insert(jaz.apHome, 0) {
			jaz.logf("home win insert accPanel failed")
		}
	}

	if jaz.winAdmin != nil {
		if jaz.apAdmin != nil {
			admin := jaz.winAdmin.ByID(jaz.apAdmin.ID())
			jaz.winAdmin.Remove(admin)
		}
		jaz.apAdmin = newAccPanel(user, jaz.staticPath)
		if !jaz.winAdmin.Insert(jaz.apAdmin, 0) {
			jaz.logf("admin win insert accPanel failed")
		}
	}

	if jaz.winLogout != nil {
		if jaz.apLogout != nil {
			logout := jaz.winLogout.ByID(jaz.apLogout.ID())
			jaz.winLogout.Remove(logout)
		}
		jaz.apLogout = newAccPanel(user, jaz.staticPath)
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
	win.AddHeadHTML(cssLink)
	jaz.logf("window=[%s] attached CSS=[%s]", path, cssLink)
	return win
}

type sortByID struct {
	data []*dev.Device
}

func (s sortByID) Len() int {
	return len(s.data)
}
func (s sortByID) Swap(i, j int) {
	s.data[i], s.data[j] = s.data[j], s.data[i]
}
func (s sortByID) Less(i, j int) bool {
	return s.data[i].ID < s.data[j].ID
}

func deviceWinName(id string) string {
	return "device-" + id
}

func splitBufLines(b []byte) []string {
	list := strings.Split(string(b), "\n")
	last := len(list) - 1
	if last < 0 {
		return list
	}
	if list[last] == "" {
		return list[:last]
	}
	return list
}

func buildDeviceWindow(jaz *app, e gwu.Event, devID string) string {
	winName := deviceWinName(devID)
	s := e.Session()
	win := s.WinByName(winName)
	if win != nil {
		return winName
	}
	winTitle := "Device: " + devID
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
	logPanel := gwu.NewPanel()
	diffPanel := gwu.NewPanel()

	panel.Add(gwu.NewLabel("Files"), filesPanel)      // tab 0
	panel.Add(gwu.NewLabel("View Config"), showPanel) // tab 1
	panel.Add(gwu.NewLabel("Properties"), propPanel)  // tab 2
	panel.Add(gwu.NewLabel("Error Log"), logPanel)    // tab 3
	panel.Add(gwu.NewLabel("Diff"), diffPanel)        // tab 4

	const tabShow = 1 // index
	const tabDiff = 4 // index

	devPrefix := dev.DeviceFullPrefix(jaz.repositoryPath, devID)
	showFile, lastErr := store.FindLastConfig(devPrefix, jaz.logger)
	if lastErr != nil {
		jaz.logger.Printf("buildDeviceWindow: could not find last config for device: %v", lastErr)
	}

	loadLog := func(e gwu.Event) {

		logPath := dev.ErrlogPath(jaz.logPathPrefix, devID)
		logPanel.Clear()
		logPanel.Add(gwu.NewLabel("File: " + logPath))

		maxSize := int64(1000 * 100) // 1000 x 100-byte lines

		d, getErr := jaz.table.GetDevice(devID)
		if getErr != nil {
			logPanel.Add(gwu.NewLabel(fmt.Sprintf("Get device error: %v", getErr)))
		} else {
			maxSize = 1000 * int64(d.Attr.ErrlogHistSize) // max 1000 bytes per line
		}

		b, readErr := store.FileRead(logPath, maxSize)
		if readErr != nil {
			logPanel.Add(gwu.NewLabel(fmt.Sprintf("Could not read '%s': %v", logPath, readErr)))
		}

		text := string(b)

		logBox := gwu.NewTextBox("")
		logBox.SetRows(30)
		logBox.SetCols(100)
		logBox.SetText(text)
		logPanel.Add(logBox)
		e.MarkDirty(logPanel)
	}

	loadView := func(e gwu.Event, show string) {
		showPanel.Clear()
		showPanel.Add(gwu.NewLabel("File: " + show))

		options := jaz.options.Get()
		b, readErr := store.FileRead(show, options.MaxConfigLoadSize)
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

	loadDiff := func(e gwu.Event, from, to string) {

		jaz.logger.Printf("diff: from=%s to=%s", from, to)

		diffPanel.Clear()
		diffPanel.Add(gwu.NewLabel("From: " + from))
		diffPanel.Add(gwu.NewLabel("To: " + to))

		options := jaz.options.Get()

		bufFrom, errReadFrom := store.FileRead(from, options.MaxConfigLoadSize)
		if errReadFrom != nil {
			diffPanel.Add(gwu.NewLabel(fmt.Sprintf("Could not read '%s': %v", from, errReadFrom)))
		}

		bufTo, errReadTo := store.FileRead(to, options.MaxConfigLoadSize)
		if errReadTo != nil {
			diffPanel.Add(gwu.NewLabel(fmt.Sprintf("Could not read '%s': %v", from, errReadTo)))
		}

		seqFrom := splitBufLines(bufFrom)
		seqTo := splitBufLines(bufTo)
		diff := difflib.Diff(seqFrom, seqTo)

		diffBox := gwu.NewTable()
		diffBox.Style().AddClass("diffbox")

		colLineNumFrom := 0
		colLineTextFrom := 1
		colLineTextTo := 2
		colLineNumTo := 3

		var f, t int

		for _, d := range diff {

			switch d.Delta {
			case difflib.LeftOnly:
				diffBox.Add(gwu.NewLabel(strconv.Itoa(f+1)), f, colLineNumFrom)
				diffBox.CellFmt(f, colLineNumFrom).Style().AddClass("diffbox_linenum")
				lab := gwu.NewLabel(d.Payload)
				diffBox.Add(lab, f, colLineTextFrom)
				diffBox.CellFmt(f, colLineTextFrom).Style().AddClass("diffbox_deleted")
				diffBox.CellFmt(f, colLineTextFrom).Style().AddClass("diffbox_text_cell")
				f++
			case difflib.RightOnly:
				diffBox.Add(gwu.NewLabel(strconv.Itoa(t+1)), t, colLineNumTo)
				diffBox.CellFmt(t, colLineNumTo).Style().AddClass("diffbox_linenum")
				lab := gwu.NewLabel(d.Payload)
				diffBox.Add(lab, t, colLineTextTo)
				diffBox.CellFmt(t, colLineTextTo).Style().AddClass("diffbox_added")
				diffBox.CellFmt(t, colLineTextTo).Style().AddClass("diffbox_text_cell")
				t++
			case difflib.Common:
				diffBox.Add(gwu.NewLabel(strconv.Itoa(f+1)), f, colLineNumFrom)
				diffBox.CellFmt(f, colLineNumFrom).Style().AddClass("diffbox_linenum")
				diffBox.Add(gwu.NewLabel(strconv.Itoa(t+1)), t, colLineNumTo)
				diffBox.CellFmt(t, colLineNumTo).Style().AddClass("diffbox_linenum")
				labF := gwu.NewLabel(d.Payload)
				labT := gwu.NewLabel(d.Payload)
				diffBox.Add(labF, f, colLineTextFrom)
				diffBox.Add(labT, t, colLineTextTo)
				diffBox.CellFmt(f, colLineTextFrom).Style().AddClass("diffbox_text_cell")
				diffBox.CellFmt(t, colLineTextTo).Style().AddClass("diffbox_text_cell")
				f++
				t++
			}
		}

		diffPanel.Add(diffBox)
		e.MarkDirty(panel)
	}

	{
		// Preload diff panel
		prefix := dev.DeviceFullPrefix(jaz.repositoryPath, devID)
		_, matches, listErr := store.ListConfigSorted(prefix, true, jaz.logger)
		if listErr != nil {
			jaz.logger.Printf("failure preloading diff panel: %v", listErr)
		}
		if len(matches) > 0 {
			diffTo := dev.DeviceFullPath(jaz.repositoryPath, devID, matches[0])
			var f string
			if len(matches) > 1 {
				f = matches[1] // previous file
			} else {
				f = matches[0] // there is no previous file
			}
			diffFrom := dev.DeviceFullPath(jaz.repositoryPath, devID, f)
			loadDiff(e, diffFrom, diffTo)
		}
	}

	win.Add(panel)

	fileList := func(e gwu.Event) {
		prefix := dev.DeviceFullPrefix(jaz.repositoryPath, devID)
		dirname, matches, listErr := store.ListConfigSorted(prefix, true, jaz.logger)
		if listErr != nil {
			filesMsg.SetText(fmt.Sprintf("List files error: %v", listErr))
			e.MarkDirty(filesPanel)
			return
		}

		filesMsg.SetText(fmt.Sprintf("%d files", len(matches)))

		filesTab.Clear()

		const COLS = 6

		row := 0

		// header
		filesTab.Add(gwu.NewLabel("Download"), row, 0)
		filesTab.Add(gwu.NewLabel("View"), row, 1)
		filesTab.Add(gwu.NewLabel("Size"), row, 2)
		filesTab.Add(gwu.NewLabel("Time"), row, 3)
		filesTab.Add(gwu.NewLabel("Diff From"), row, 4)
		filesTab.Add(gwu.NewLabel("Compare"), row, 5)

		row++

		// Scan files:

		for i, m := range matches {
			path := filepath.Join(dirname, m)
			timeStr := "unknown"

			modTime, size, infoErr := store.FileInfo(path)
			if infoErr == nil {
				timeStr = timestampString(modTime)
			} else {
				timeStr += fmt.Sprintf("(could not get file info: %v)", infoErr)
			}

			var filePath string

			if store.S3Path(path) {
				filePath = store.S3URL(path)
			} else {
				filePath = fmt.Sprintf("%s/%s/%s", jaz.repoPath, devID, m)
			}
			devLink := gwu.NewLink(m, filePath)

			buttonView := gwu.NewButton("Open")
			show := dev.DeviceFullPath(jaz.repositoryPath, devID, m)
			buttonView.AddEHandlerFunc(func(e gwu.Event) {
				loadView(e, show)
				panel.SetSelected(tabShow)
			}, gwu.ETypeClick)

			listDiffSrc := gwu.NewListBox(matches)
			buttonDiff := gwu.NewButton("Diff")

			var diffFrom int
			if i < len(matches)-1 {
				// default diff src is previous file
				diffFrom = i + 1
			} else {
				// there is no previous file
				diffFrom = i
			}
			listDiffSrc.SetSelectedIndices([]int{diffFrom})

			diffTo := dev.DeviceFullPath(jaz.repositoryPath, devID, m)
			buttonDiff.AddEHandlerFunc(func(e gwu.Event) {
				from := listDiffSrc.SelectedIdx()
				f := matches[from]
				diffFrom := dev.DeviceFullPath(jaz.repositoryPath, devID, f)
				loadDiff(e, diffFrom, diffTo)
				panel.SetSelected(tabDiff)
			}, gwu.ETypeClick)

			filesTab.Add(devLink, row, 0)
			filesTab.Add(buttonView, row, 1)
			filesTab.Add(gwu.NewLabel(strconv.FormatInt(size, 10)), row, 2)
			filesTab.Add(gwu.NewLabel(timeStr), row, 3)
			filesTab.Add(listDiffSrc, row, 4)
			filesTab.Add(buttonDiff, row, 5)

			row++
		}

		// Attach CSS formatting to cells

		for r := 0; r < row; r++ {
			for j := 0; j < COLS; j++ {
				filesTab.CellFmt(r, j).Style().AddClass("device_files_cell")
			}
		}
	}

	resetProp := func(e gwu.Event) {
		d, getErr := jaz.table.GetDevice(devID)
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
		loadLog(e)   // load log
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

		d, getErr := jaz.table.GetDevice(devID)
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

func buildDeviceTable(jaz *app, s gwu.Session, t gwu.Table, tabSumm gwu.Panel) {
	const COLS = 10

	row := 0 // filter
	filterModel := gwu.NewTextBox(jaz.filterModel)
	filterID := gwu.NewTextBox(jaz.filterID)
	filterHost := gwu.NewTextBox(jaz.filterHost)

	inputCols := 10
	filterModel.SetCols(inputCols)
	filterID.SetCols(inputCols)
	filterHost.SetCols(inputCols)

	filterModel.AddSyncOnETypes(gwu.ETypeKeyUp) // synchronize values during editing (while you type in characters)
	filterID.AddSyncOnETypes(gwu.ETypeKeyUp)    // synchronize values during editing (while you type in characters)
	filterHost.AddSyncOnETypes(gwu.ETypeKeyUp)  // synchronize values during editing (while you type in characters)

	filterModel.AddEHandlerFunc(func(e gwu.Event) {
		jaz.filterModel = filterModel.Text()
		refreshDeviceTable(jaz, t, tabSumm, e)
	}, gwu.ETypeChange)

	filterID.AddEHandlerFunc(func(e gwu.Event) {
		jaz.filterID = filterID.Text()
		refreshDeviceTable(jaz, t, tabSumm, e)
	}, gwu.ETypeChange)

	filterHost.AddEHandlerFunc(func(e gwu.Event) {
		jaz.filterHost = filterHost.Text()
		refreshDeviceTable(jaz, t, tabSumm, e)
	}, gwu.ETypeChange)

	t.Add(filterModel, row, 0)
	t.Add(filterID, row, 1)
	t.Add(filterHost, row, 2)
	t.Add(gwu.NewLabel(""), row, 3)
	t.Add(gwu.NewLabel(""), row, 4)
	t.Add(gwu.NewLabel(""), row, 5)
	t.Add(gwu.NewLabel(""), row, 6)
	t.Add(gwu.NewLabel(""), row, 7)
	t.Add(gwu.NewLabel(""), row, 8)
	t.Add(gwu.NewLabel(""), row, 9)

	hostPort := gwu.NewLabel("Host:Port")
	hostPort.SetAttr("title", "Part ':Port' is optional")

	row = 1 // header
	t.Add(gwu.NewLabel("Model"), row, 0)
	t.Add(gwu.NewLabel("Device"), row, 1)
	t.Add(hostPort, row, 2)
	t.Add(gwu.NewLabel("Transport"), row, 3)
	t.Add(gwu.NewLabel("Last Status"), row, 4)
	t.Add(gwu.NewLabel("Elapsed"), row, 5)
	t.Add(gwu.NewLabel("Last Try"), row, 6)
	t.Add(gwu.NewLabel("Last Success"), row, 7)
	t.Add(gwu.NewLabel("Holdtime"), row, 8)
	t.Add(gwu.NewLabel("Run Now"), row, 9)

	devList := jaz.table.ListDevices()
	sort.Sort(sortByID{data: devList})

	now := time.Now()

	options := jaz.options.Get()

	row = 2
	for _, d := range devList {

		if !strings.Contains(d.Model(), filterModel.Text()) {
			continue
		}
		if !strings.Contains(d.ID, filterID.Text()) {
			continue
		}
		if !strings.Contains(d.HostPort, filterHost.Text()) {
			continue
		}

		labMod := gwu.NewLabel(d.Model())

		//devWin := deviceWinName(d.ID)
		//labID := gwu.NewLink(d.ID, devWin)
		buttonID := gwu.NewButton(d.ID)

		devID := d.ID // get dev id for closure below
		buttonID.AddEHandlerFunc(func(e gwu.Event) {
			winName := buildDeviceWindow(jaz, e, devID)
			e.ReloadWin(winName)
		}, gwu.ETypeClick)

		labHost := gwu.NewLabel(d.HostPort)
		labTransport := gwu.NewLabel(d.Transports)
		var imageLastStatus gwu.Image
		if d.LastStatus() {
			imageLastStatus = gwu.NewImage("Success", fmt.Sprintf("%s/ok-small.png", jaz.staticPath))
		} else {
			imageLastStatus = gwu.NewImage("Failure", fmt.Sprintf("%s/fail-small.png", jaz.staticPath))
		}
		labElapsed := gwu.NewLabel(durationSecString(d.LastElapsed()))
		labLastTry := gwu.NewLabel(timestampString(d.LastTry()))
		labLastSuccess := gwu.NewLabel(timestampString(d.LastSuccess()))
		h := d.Holdtime(now, options.Holdtime)
		if h < 0 {
			h = 0
		}
		labHoldtime := gwu.NewLabel(durationSecString(h))

		buttonRun := gwu.NewButton("Run")
		id := d.ID
		buttonRun.AddEHandlerFunc(func(e gwu.Event) {
			// run in a goroutine to not block the UI on channel write
			go runPriority(jaz, id)
		}, gwu.ETypeClick)

		t.Add(labMod, row, 0)
		t.Add(buttonID, row, 1)
		t.Add(labHost, row, 2)
		t.Add(labTransport, row, 3)
		t.Add(imageLastStatus, row, 4)
		t.Add(labElapsed, row, 5)
		t.Add(labLastTry, row, 6)
		t.Add(labLastSuccess, row, 7)
		t.Add(labHoldtime, row, 8)
		t.Add(buttonRun, row, 9)

		row++
	}

	for r := 0; r < row; r++ {
		for j := 0; j < COLS; j++ {
			t.CellFmt(r, j).Style().AddClass("device_table_cell")
		}
	}

	tabSumm.Clear()
	tabSumm.Add(gwu.NewLabel(fmt.Sprintf("Filter: %d selected from %d total devices", row-2, len(devList))))
}

func runPriority(jaz *app, id string) {
	jaz.logger.Printf("runPriority: device: %s", id)

	_, clearErr := dev.ClearDeviceStatus(jaz.table, id, jaz.logger, jaz.options.Get().Holdtime)
	if clearErr != nil {
		jaz.logger.Printf("runPriority: clear device %s status error: %v", id, clearErr)
		return
	}

	jaz.requestChan <- dev.FetchRequest{ID: id}
}

func refreshDeviceTable(jaz *app, t gwu.Table, tabSumm gwu.Panel, e gwu.Event) {
	t.Clear() // clear out table contents
	buildDeviceTable(jaz, e.Session(), t, tabSumm)
	e.MarkDirty(t)
	e.MarkDirty(tabSumm)
}

func buildHomeWin(jaz *app, s gwu.Session) {

	winName := fmt.Sprintf("%s home", appName)
	win := newWin(jaz, "home", winName)

	win.Add(jaz.apHome)

	l := gwu.NewLabel(winName)
	l.Style().SetFontWeight(gwu.FontWeightBold).SetFontSize("130%")
	win.Add(l)

	tableSumm := gwu.NewPanel()
	tableSumm.Add(gwu.NewLabel("table summary"))
	t := gwu.NewTable()
	t.Style().AddClass("device_table")

	createButton := gwu.NewButton("Create")

	refresh := func(e gwu.Event) {
		createButton.SetEnabled(userIsLogged(e.Session()))
		e.MarkDirty(createButton)
		refreshDeviceTable(jaz, t, tableSumm, e)
	}

	createDevPanel := buildCreateDevPanel(jaz, s, refresh, createButton)

	refreshButton := gwu.NewButton("Refresh")
	refreshButton.AddEHandlerFunc(refresh, gwu.ETypeClick)
	win.Add(refreshButton)

	win.AddEHandlerFunc(refresh, gwu.ETypeWinLoad)

	createDevExpander := gwu.NewExpander()
	createDevExpander.SetHeader(gwu.NewLabel("Create device"))
	createDevExpander.SetContent(createDevPanel)

	win.Add(createDevExpander)

	win.Add(gwu.NewLabel("Hint: fill in text boxes below to select matching subset of devices."))

	buildDeviceTable(jaz, s, t, tableSumm)

	win.Add(tableSumm)
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
	panelID := gwu.NewPanel()
	panelHost := gwu.NewPanel()
	panelTransport := gwu.NewPanel()
	panelUser := gwu.NewPanel()
	panelPass := gwu.NewPanel()
	panelEnable := gwu.NewPanel()
	button := createButton

	labelModel := gwu.NewLabel("Model")
	labelID := gwu.NewLabel("ID")
	labelHost := gwu.NewLabel("Host:Port")
	labelHost.SetAttr("title", "Part ':Port' is optional")
	labelTransport := gwu.NewLabel("Transports")
	labelUser := gwu.NewLabel("User")
	labelPass := gwu.NewLabel("Pass")
	labelEnable := gwu.NewLabel("Enable")

	autoIDPrefix := "auto"

	models := jaz.table.ListModels()
	sort.Strings(models)

	listModel := gwu.NewListBox(models)
	textID := gwu.NewTextBox(autoIDPrefix)
	textHost := gwu.NewTextBox("")
	textTransport := gwu.NewTextBox("ssh,telnet")
	textUser := gwu.NewTextBox("")
	textPass := gwu.NewTextBox("")
	textEnable := gwu.NewTextBox("")

	listModel.SetSelected(0, true)

	inputCols := 10
	textID.SetCols(inputCols)
	textHost.SetCols(inputCols)
	textTransport.SetCols(inputCols)
	textUser.SetCols(inputCols)
	textPass.SetCols(inputCols)
	textEnable.SetCols(inputCols)

	panelModel.Add(labelModel)
	panelModel.Add(listModel)
	panelID.Add(labelID)
	panelID.Add(textID)
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
	createPanel.Add(panelID)
	createPanel.Add(panelHost)
	createPanel.Add(panelTransport)
	createPanel.Add(panelUser)
	createPanel.Add(panelPass)
	createPanel.Add(panelEnable)
	createPanel.Add(button)

	createAutoID := func() {
		if strings.HasPrefix(textID.Text(), autoIDPrefix) {
			textID.SetText(jaz.table.FindDeviceFreeID(autoIDPrefix))
		}
	}

	button.AddEHandlerFunc(func(e gwu.Event) {

		if !userIsLogged(e.Session()) {
			return // refuse to create
		}

		id := textID.Text()

		if id == autoIDPrefix {
			id = jaz.table.FindDeviceFreeID(autoIDPrefix)
		}

		change := conf.Change{
			From: eventRemoteAddress(e),
			By:   sessionUsername(e.Session()),
			When: time.Now(),
		}

		mod := listModel.SelectedValue()
		if mod == "" {
			msg.SetText(fmt.Sprintf("Invalid model: [%s]", mod))
			e.MarkDirty(createDevPanel)
			return
		}

		host := strings.TrimSpace(textHost.Text())
		textHost.SetText(host)
		e.MarkDirty(textHost)

		if createErr := dev.CreateDevice(jaz.table, jaz.logger, mod, id, host, textTransport.Text(), textUser.Text(), textPass.Text(), textEnable.Text(), false, &change); createErr != nil {
			msg.SetText("Could not create device: " + createErr.Error())
			e.MarkDirty(createDevPanel)
			return
		}

		saveConfig(jaz, change)

		createAutoID() // prepare next auto id
		e.MarkDirty(textID)

		msg.SetText("Device created: " + id)
		e.MarkDirty(createDevPanel) // redraw msg
		refresh(e)                  // redraw device table
	}, gwu.ETypeClick)

	createAutoID() // first call

	return createDevPanel
}

func timestampString(ts time.Time) string {
	if ts.IsZero() {
		return "never"
	}
	return ts.Format("2006-01-02 15:04:05")
}

func durationSecString(d time.Duration) string {
	return fmt.Sprintf("%.3fs", d.Seconds())
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
	win.SetFocusedCompID(tb.ID())

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
	jaz.apHome = newAccPanel("", jaz.staticPath)
	jaz.apAdmin = newAccPanel("", jaz.staticPath)
	jaz.apLogout = newAccPanel("", jaz.staticPath)
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
			jaz.logger.Printf("buildAdminWin: could not find last config: %v", lastErr)
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

		// overwrite change record
		opt.LastChange.From = eventRemoteAddress(e)
		opt.LastChange.By = sessionUsername(e.Session())
		opt.LastChange.When = time.Now()

		jaz.options.Set(opt) // set all options from text field, including change record

		saveConfig(jaz, opt.LastChange) // will also update in-memory change record again

		refresh(e)

		settingsMsg.SetText("Saved.")

	}, gwu.ETypeClick)

	win.Add(settingsPanel)

	win.AddEHandlerFunc(refresh, gwu.ETypeWinLoad)

	s.AddWin(win)

	jaz.winAdmin = win
}
