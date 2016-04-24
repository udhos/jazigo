package dev

import (
	"time"

	"github.com/udhos/jazigo/conf"
)

func registerModelHTTP(logger hasPrintf, t *DeviceTable) {
	modelName := "http"
	m := &Model{name: modelName}

	m.defaultAttr = conf.DevAttributes{
		NeedLoginChat:               false,
		NeedEnabledMode:             false,
		NeedPagingOff:               false,
		EnableCommand:               "",
		UsernamePromptPattern:       "",
		PasswordPromptPattern:       "",
		EnablePasswordPromptPattern: "",
		DisabledPromptPattern:       "",
		EnabledPromptPattern:        "",
		CommandList:                 []string{"GET / HTTP/1.0\r\n\r\n"},
		DisablePagerCommand:         "",
		ReadTimeout:                 5 * time.Second,
		MatchTimeout:                10 * time.Second,
		SendTimeout:                 5 * time.Second,
		CommandReadTimeout:          5 * time.Second,  // larger timeout for slow 'sh run'
		CommandMatchTimeout:         10 * time.Second, // larger timeout for slow 'sh run'
		SupressAutoLF:               true,
	}

	if err := t.SetModel(m, logger); err != nil {
		logger.Printf("registerModelHTTP: %v", err)
	}
}
