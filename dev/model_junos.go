package dev

import (
	"time"
)

func registerModelJunOS(logger hasPrintf, t *DeviceTable) {
	modelName := "junos"
	m := &Model{name: modelName}

	m.defaultAttr = attributes{
		needLoginChat:               true,
		needEnabledMode:             false,
		needPagingOff:               true,
		enableCommand:               "",
		usernamePromptPattern:       `login:\s*$`,
		passwordPromptPattern:       `Password:\s*$`,
		enablePasswordPromptPattern: "",
		disabledPromptPattern:       `\S+>\s*$`,
		enabledPromptPattern:        `\S+>\s*$`,
		commandList:                 []string{"show configuration | display set"},
		disablePagerCommand:         "set cli screen-length 0",
		readTimeout:                 10 * time.Second,
		matchTimeout:                20 * time.Second,
		sendTimeout:                 5 * time.Second,
		commandReadTimeout:          20 * time.Second, // larger timeout for slow 'sh run'
		commandMatchTimeout:         30 * time.Second, // larger timeout for slow 'sh run'
	}

	if err := t.SetModel(m); err != nil {
		logger.Printf("registerModelJunOS: %v", err)
	}
}
