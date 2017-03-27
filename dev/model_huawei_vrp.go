package dev

import (
	"time"

	"github.com/udhos/jazigo/conf"
)

func registerModelHuaweiVRP(logger hasPrintf, t *DeviceTable) {
	a := conf.NewDevAttr()

	/*
		a.CommandList = []string{
			"user-interface vty 0 4",
			"screen-length 0", // disable paging
			"quit",
			"disp ver",  // get system information
			"disp curr", // get configuration
			"user-interface vty 0 4",
			"screen-length 24", // restore paging
		}
		a.NeedEnabledMode = true
		a.EnableCommand = "sys"
		a.EnabledPromptPattern = `\[[^\[\]]+\]$`
	*/

	a.CommandList = []string{
		"screen-length 0 temporary", // disable paging
		"disp ver",                  // get system information
		"disp curr",                 // get configuration
	}

	promptPattern := `<[^<>]+>$`
	a.DisabledPromptPattern = promptPattern
	a.EnabledPromptPattern = promptPattern

	a.NeedLoginChat = true
	a.UsernamePromptPattern = `Username:$`
	a.PasswordPromptPattern = `Password:$`
	a.ReadTimeout = 10 * time.Second
	a.MatchTimeout = 20 * time.Second
	a.SendTimeout = 5 * time.Second
	a.CommandReadTimeout = 15 * time.Second  // larger timeout for slow 'sh run'
	a.CommandMatchTimeout = 25 * time.Second // larger timeout for slow 'sh run'
	a.QuoteSentCommandsFormat = `##[%s]`

	m := &Model{name: "huawei-vrp"}
	m.defaultAttr = a
	if err := t.SetModel(m, logger); err != nil {
		logger.Printf("registerModelHuaweiVRP: %v", err)
	}
}
