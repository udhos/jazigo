package main

import (
	"log"
)

type model struct {
	name        string
	defaultAttr attributes
}

type attributes struct {
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

func registerModelCiscoIOS(models map[string]*model) {
	modelName := "cisco-ios"
	m := &model{name: modelName}

	m.defaultAttr = attributes{
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

	log.Printf("registerModelCiscoIOS: FIXME WRITEME program chat sequence")
}

func createDevice(jaz *app, modelName, id, hostPort, transports, user, pass, enable string) {
	log.Printf("createDevice: %s %s %s %s", modelName, id, hostPort, transports)

	mod, ok := jaz.models[modelName]
	if !ok {
		log.Printf("createDevice: could not find model '%s'", modelName)
	}

	dev := &device{devModel: mod, id: id, hostPort: hostPort, transports: transports, loginUser: user, loginPassword: pass, enablePassword: enable}

	dev.attr = mod.defaultAttr

	jaz.devices[id] = dev
}
