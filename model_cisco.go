package main

import (
	"fmt"
	"time"
)

type model struct {
	name        string
	defaultAttr attributes
}

type attributes struct {
	loginChat                   bool     // expect login chat
	enableCommand               string   // enable
	usernamePromptPattern       string   // Username:
	passwordPromptPattern       string   // Password:
	enablePasswordPromptPattern string   // Password:
	disabledPromptPattern       string   // >
	enabledPromptPattern        string   // #
	commandList                 []string // show run
	disablePagerCommand         string   // term len 0
}

type device struct {
	devModel   *model
	id         string
	hostPort   string
	transports string

	loginUser      string
	loginPassword  string
	enablePassword string

	attr attributes
}

const (
	FETCH_ERR_NONE   = 0
	FETCH_ERR_TRANSP = 1
	FETCH_ERR_LOGIN  = 2
	FETCH_ERR_CHAT   = 3
	FETCH_ERR_OTHER  = 4
)

func (d *device) fetch(resultCh chan fetchResult, delay time.Duration) {
	modelName := d.devModel.name
	logger.Printf("fetch: %s %s %s %s delay=%dms", modelName, d.id, d.hostPort, d.transports, delay/time.Millisecond)

	if delay > 0 {
		time.Sleep(delay)
	}

	session, logged, err := openTransport(modelName, d.id, d.hostPort, d.transports, d.loginUser, d.loginPassword)
	if err != nil {
		resultCh <- fetchResult{model: modelName, devId: d.id, devHostPort: d.hostPort, msg: fmt.Sprintf("fetch transport: %v", err), code: FETCH_ERR_TRANSP}
		return
	}

	logger.Printf("fetch: %s %s %s - transport open session=%v logged=%v", modelName, d.id, d.hostPort, session, logged)

	capture := dialog{}

	if d.attr.loginChat && !logged {
		err1 := d.login(session, &capture)
		if err1 != nil {
			resultCh <- fetchResult{model: modelName, devId: d.id, devHostPort: d.hostPort, msg: fmt.Sprintf("fetch login: %v", err1), code: FETCH_ERR_LOGIN}
			return
		}
	}

	resultCh <- fetchResult{model: modelName, devId: d.id, devHostPort: d.hostPort, msg: "fetch: FIXME WRITEME", code: FETCH_ERR_OTHER}
}

func (d *device) login(t transp, capture *dialog) error {
	return fmt.Errorf("login: FIXME WRITEME")
}

type dialog struct {
}

func registerModelCiscoIOS(models map[string]*model) {
	modelName := "cisco-ios"
	m := &model{name: modelName}

	m.defaultAttr = attributes{
		loginChat:                   true,
		enableCommand:               "enable",
		usernamePromptPattern:       "Username: ",
		passwordPromptPattern:       "Password: ",
		enablePasswordPromptPattern: "Password: ",
		disabledPromptPattern:       "> ",
		enabledPromptPattern:        "# ",
		commandList:                 []string{"show run"},
		disablePagerCommand:         "term len 0",
	}

	models[modelName] = m

	logger.Printf("registerModelCiscoIOS: FIXME WRITEME program chat sequence")
}

func createDevice(jaz *app, modelName, id, hostPort, transports, user, pass, enable string) {
	logger.Printf("createDevice: %s %s %s %s", modelName, id, hostPort, transports)

	mod, ok := jaz.models[modelName]
	if !ok {
		logger.Printf("createDevice: could not find model '%s'", modelName)
	}

	dev := &device{devModel: mod, id: id, hostPort: hostPort, transports: transports, loginUser: user, loginPassword: pass, enablePassword: enable}

	dev.attr = mod.defaultAttr

	jaz.devices[id] = dev
}
