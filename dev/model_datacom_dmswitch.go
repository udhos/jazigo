package dev

import (
	"time"

	"github.com/udhos/jazigo/conf"
)

func registerModelDatacomDmswitch(logger hasPrintf, t *DeviceTable) {
	a := conf.NewDevAttr()

	a.CommandList = []string{
		"no terminal paging", // disable paging
		"show system",
		"show firmware",
		"show running-config",
		"terminal paging", // enable paging
	}

	a.DisabledPromptPattern = `[^#\s]+#$`

	a.NeedLoginChat = true
	a.UsernamePromptPattern = `login:\s*$`
	a.PasswordPromptPattern = `Password:\s*$`
	a.ReadTimeout = 10 * time.Second
	a.MatchTimeout = 20 * time.Second
	a.SendTimeout = 5 * time.Second
	a.CommandReadTimeout = 15 * time.Second  // larger timeout for slow 'sh run'
	a.CommandMatchTimeout = 25 * time.Second // larger timeout for slow 'sh run'
	a.QuoteSentCommandsFormat = `!![%s]`

	m := &Model{name: "dmswitch"}
	m.defaultAttr = a
	if err := t.SetModel(m, logger); err != nil {
		logger.Printf("registerModelDatacomDmswitch: %v", err)
	}
}
