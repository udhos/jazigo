package dev

import (
	"time"
)

func registerModelLinux(logger hasPrintf, models map[string]*Model) {
	modelName := "linux"
	m := &Model{name: modelName}

	m.defaultAttr = attributes{
		needLoginChat:               true,
		needEnabledMode:             false,
		needPagingOff:               false,
		enableCommand:               "",
		usernamePromptPattern:       `Username:\s*$`,
		passwordPromptPattern:       `Password:\s*$`,
		enablePasswordPromptPattern: "",
		disabledPromptPattern:       `\$\s*$`,
		enabledPromptPattern:        `\$\s*$`,
		commandList:                 []string{"/bin/bash -c '/bin/uname -a; echo prompt$'"}, // echo prompt$ --> trick to issue prompt after uname
		disablePagerCommand:         "",
		readTimeout:                 5 * time.Second,
		matchTimeout:                10 * time.Second,
		sendTimeout:                 5 * time.Second,
		commandReadTimeout:          10 * time.Second,
		commandMatchTimeout:         10 * time.Second,
	}

	models[modelName] = m
}
