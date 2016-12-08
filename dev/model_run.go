package dev

import (
	"time"

	"github.com/udhos/jazigo/conf"
)

func registerModelRun(logger hasPrintf, t *DeviceTable) {
	a := conf.NewDevAttr()

	a.RunProg = []string{"/bin/bash", "-c", "env | egrep ^JAZIGO_"}
	a.RunTimeout = 60 * time.Second
	a.EnabledPromptPattern = ""  // "" --> look for EOF
	a.CommandList = []string{""} // "" = dont send, wait for command prompt
	a.ReadTimeout = 5 * time.Second
	a.MatchTimeout = 10 * time.Second
	a.SendTimeout = 5 * time.Second
	a.CommandReadTimeout = 10 * time.Second
	a.CommandMatchTimeout = 10 * time.Second

	m := &Model{name: "run"}
	m.defaultAttr = a
	if err := t.SetModel(m, logger); err != nil {
		logger.Printf("registerModelRun: %v", err)
	}
}
