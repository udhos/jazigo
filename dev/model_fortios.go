package dev

import (
	"time"

	"github.com/udhos/jazigo/conf"
)

func registerModelFortiOS(logger hasPrintf, t *DeviceTable) {
	a := conf.NewDevAttr()

	// old method for disabling pager
	/*
		a.NeedPagingOff = true
		a.DisablePagerCommand = "config system console\nset output standard\nend"
		a.DisablePagerExtraPromptCount = 2
		a.CommandList = []string{"get system status", "show"}
	*/

	// preferred method for disabling pager
	a.CommandList = []string{
		"config system global",  // enter config: valid only for vdom
		"config system console", // enter config: valid only for non-vdom
		"set output standard",   // disable paging
		"end",                   // exit config
		"get system status",     // system information
		"show",                  // get configuration
	}

	promptPattern := `\S+\s#\s$` // "hostname # "
	a.DisabledPromptPattern = promptPattern
	a.EnabledPromptPattern = promptPattern
	a.NeedLoginChat = true
	a.UsernamePromptPattern = `login:\s*$`
	a.PasswordPromptPattern = `Password:\s*$`
	a.ReadTimeout = 10 * time.Second
	a.MatchTimeout = 20 * time.Second
	a.SendTimeout = 5 * time.Second
	a.CommandReadTimeout = 20 * time.Second  // larger timeout for slow 'sh run'
	a.CommandMatchTimeout = 60 * time.Second // larger timeout for slow 'sh run'
	a.QuoteSentCommandsFormat = `##[%s]`

	m := &Model{name: "fortios"}
	m.defaultAttr = a
	if err := t.SetModel(m, logger); err != nil {
		logger.Printf("registerModelFortiOS: %v", err)
	}
}
