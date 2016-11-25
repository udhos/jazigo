package dev

import (
	"time"

	"github.com/udhos/jazigo/conf"
)

func registerModelCiscoIOS(logger hasPrintf, t *DeviceTable) {
	modelName := "cisco-ios"
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
		CommandList:                 []string{"show ver", "show run"},
		DisablePagerCommand:         "term len 0",
		ReadTimeout:                 10 * time.Second,
		MatchTimeout:                20 * time.Second,
		SendTimeout:                 5 * time.Second,
		CommandReadTimeout:          20 * time.Second, // larger timeout for slow 'sh run'
		CommandMatchTimeout:         30 * time.Second, // larger timeout for slow 'sh run'
		QuoteSentCommandsFormat:     `!![%s]`,
	}

	if err := t.SetModel(m, logger); err != nil {
		logger.Printf("registerModelCiscoIOS: %v", err)
	}
}
