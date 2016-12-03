package dev

import (
	"time"

	"github.com/udhos/jazigo/conf"
)

func registerModelRun(logger hasPrintf, t *DeviceTable) {
	m := &Model{name: "run"}

	m.defaultAttr = conf.DevAttributes{
		//Run:                  []string{"/usr/bin/printf", "hello\nworld\n"},
		//Run:                  []string{"/usr/bin/env"},
		Run:                  []string{"/bin/bash", "-c", "env | egrep ^JAZIGO_"},
		EnabledPromptPattern: "",           // "" --> look for EOF
		CommandList:          []string{""}, // "" = dont send, wait for command prompt
		ReadTimeout:          5 * time.Second,
		MatchTimeout:         10 * time.Second,
		SendTimeout:          5 * time.Second,
		CommandReadTimeout:   10 * time.Second,
		CommandMatchTimeout:  10 * time.Second,
	}

	if err := t.SetModel(m, logger); err != nil {
		logger.Printf("registerModelRun: %v", err)
	}
}
