package dev

import (
	"time"

	"github.com/udhos/jazigo/conf"
)

func registerModelLinux(logger hasPrintf, t *DeviceTable) {
	a := conf.NewDevAttr()

	a.NeedLoginChat = true
	a.NeedEnabledMode = false
	a.NeedPagingOff = false
	a.EnableCommand = ""
	a.UsernamePromptPattern = `Username:\s*$`
	a.PasswordPromptPattern = `Password:\s*$`
	a.EnablePasswordPromptPattern = ""
	a.DisabledPromptPattern = `\$\s*$`
	a.EnabledPromptPattern = `\$\s*$`
	a.CommandList = []string{"", "/bin/uname -a", "/usr/bin/uptime", "/bin/ls"} // "" = dont send, wait for command prompt
	a.DisablePagerCommand = ""
	a.ReadTimeout = 5 * time.Second
	a.MatchTimeout = 10 * time.Second
	a.SendTimeout = 5 * time.Second
	a.CommandReadTimeout = 10 * time.Second
	a.CommandMatchTimeout = 10 * time.Second
	a.QuoteSentCommandsFormat = `##[%s]`

	m := &Model{name: "linux"}
	m.defaultAttr = a
	if err := t.SetModel(m, logger); err != nil {
		logger.Printf("registerModelLinux: %v", err)
	}
}
