package dev

import (
	"time"

	"github.com/udhos/jazigo/conf"
)

func registerModelJunOS(logger hasPrintf, t *DeviceTable) {
	a := conf.NewDevAttr()

	a.NeedLoginChat = true
	a.NeedEnabledMode = false
	a.NeedPagingOff = true
	a.EnableCommand = ""
	a.UsernamePromptPattern = `login:\s*$`
	a.PasswordPromptPattern = `Password:\s*$`
	a.EnablePasswordPromptPattern = ""
	a.DisabledPromptPattern = `\S+>\s*$`
	a.EnabledPromptPattern = `\S+>\s*$`
	a.CommandList = []string{"show ver", "show conf | disp set"}
	a.DisablePagerCommand = "set cli screen-length 0"
	a.ReadTimeout = 10 * time.Second
	a.MatchTimeout = 20 * time.Second
	a.SendTimeout = 5 * time.Second
	a.CommandReadTimeout = 20 * time.Second  // larger timeout for slow 'sh run'
	a.CommandMatchTimeout = 30 * time.Second // larger timeout for slow 'sh run'
	a.QuoteSentCommandsFormat = `##[%s]`
	a.S3ContentType = "detect"

	m := &Model{name: "junos"}
	m.defaultAttr = a
	if err := t.SetModel(m, logger); err != nil {
		logger.Printf("registerModelJunOS: %v", err)
	}
}
