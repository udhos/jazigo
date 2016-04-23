package dev

import (
	"time"
)

func registerModelCiscoIOSXR(logger hasPrintf, t *DeviceTable) {
	modelName := "cisco-iosxr"
	m := &Model{name: modelName}

	m.defaultAttr = attributes{
		needLoginChat:               true,
		needEnabledMode:             true,
		needPagingOff:               true,
		enableCommand:               "enable",
		usernamePromptPattern:       `Username:\s*$`,
		passwordPromptPattern:       `Password:\s*$`,
		enablePasswordPromptPattern: `Password:\s*$`,
		disabledPromptPattern:       `\S+>\s*$`,
		enabledPromptPattern:        `\S+#\s*$`,
		commandList:                 []string{"show clock det", "show ver", "show run"},
		disablePagerCommand:         "term len 0",
		readTimeout:                 10 * time.Second,
		matchTimeout:                20 * time.Second,
		sendTimeout:                 5 * time.Second,
		commandReadTimeout:          20 * time.Second, // larger timeout for slow 'sh run'
		commandMatchTimeout:         30 * time.Second, // larger timeout for slow 'sh run'
	}

	if err := t.SetModel(m, logger); err != nil {
		logger.Printf("registerModelCiscoIOSXR: %v", err)
	}
}
