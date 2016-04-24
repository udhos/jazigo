package dev

import (
	"time"

	"github.com/udhos/jazigo/conf"
)

func registerModelJunOS(logger hasPrintf, t *DeviceTable) {
	modelName := "junos"
	m := &Model{name: modelName}

	m.defaultAttr = conf.DevAttributes{
		NeedLoginChat:               true,
		NeedEnabledMode:             false,
		NeedPagingOff:               true,
		EnableCommand:               "",
		UsernamePromptPattern:       `login:\s*$`,
		PasswordPromptPattern:       `Password:\s*$`,
		EnablePasswordPromptPattern: "",
		DisabledPromptPattern:       `\S+>\s*$`,
		EnabledPromptPattern:        `\S+>\s*$`,
		CommandList:                 []string{"show ver", "show conf | disp set"},
		DisablePagerCommand:         "set cli screen-length 0",
		ReadTimeout:                 10 * time.Second,
		MatchTimeout:                20 * time.Second,
		SendTimeout:                 5 * time.Second,
		CommandReadTimeout:          20 * time.Second, // larger timeout for slow 'sh run'
		CommandMatchTimeout:         30 * time.Second, // larger timeout for slow 'sh run'
	}

	if err := t.SetModel(m, logger); err != nil {
		logger.Printf("registerModelJunOS: %v", err)
	}
}
