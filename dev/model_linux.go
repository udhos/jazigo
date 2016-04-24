package dev

import (
	"time"

	"github.com/udhos/jazigo/conf"
)

func registerModelLinux(logger hasPrintf, t *DeviceTable) {
	modelName := "linux"
	m := &Model{name: modelName}

	m.defaultAttr = conf.DevAttributes{
		NeedLoginChat:               true,
		NeedEnabledMode:             false,
		NeedPagingOff:               false,
		EnableCommand:               "",
		UsernamePromptPattern:       `Username:\s*$`,
		PasswordPromptPattern:       `Password:\s*$`,
		EnablePasswordPromptPattern: "",
		DisabledPromptPattern:       `\$\s*$`,
		EnabledPromptPattern:        `\$\s*$`,
		CommandList:                 []string{"", "/bin/uname -a", "/usr/bin/uptime", "/bin/ls"},
		DisablePagerCommand:         "",
		ReadTimeout:                 5 * time.Second,
		MatchTimeout:                10 * time.Second,
		SendTimeout:                 5 * time.Second,
		CommandReadTimeout:          10 * time.Second,
		CommandMatchTimeout:         10 * time.Second,
	}

	if err := t.SetModel(m, logger); err != nil {
		logger.Printf("registerModelLinux: %v", err)
	}
}
