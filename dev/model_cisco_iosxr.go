package dev

import (
	"github.com/udhos/jazigo/conf"
	"time"
)

func registerModelCiscoIOSXR(logger hasPrintf, t *DeviceTable) {
	modelName := "cisco-iosxr"
	m := &Model{name: modelName}

	m.defaultAttr = conf.DevAttributes{
		NeedLoginChat:               true,
		NeedEnabledMode:             true,
		NeedPagingOff:               true,
		EnableCommand:               "enable",
		UsernamePromptPattern:       `Username:\s*$`,
		PasswordPromptPattern:       `Password:\s*$`,
		EnablePasswordPromptPattern: `Password:\s*$`,
		DisabledPromptPattern:       `\S+>\s*$`,
		EnabledPromptPattern:        `\S+#\s*$`,
		CommandList:                 []string{"show ver br", "show run"},
		DisablePagerCommand:         "term len 0",
		ReadTimeout:                 10 * time.Second,
		MatchTimeout:                20 * time.Second,
		SendTimeout:                 5 * time.Second,
		CommandReadTimeout:          20 * time.Second, // larger timeout for slow 'sh run'
		CommandMatchTimeout:         30 * time.Second, // larger timeout for slow 'sh run'
		QuoteSentCommandsFormat:     `!![%s]`,
		LineFilter:                  "iosxr", // line filter name - applied to every saved line
	}

	if err := t.SetModel(m, logger); err != nil {
		logger.Printf("registerModelCiscoIOSXR: %v", err)
	}
}
