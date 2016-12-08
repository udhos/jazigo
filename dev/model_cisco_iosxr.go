package dev

import (
	"github.com/udhos/jazigo/conf"
	"time"
)

func registerModelCiscoIOSXR(logger hasPrintf, t *DeviceTable) {
	a := conf.NewDevAttr()

	a.NeedLoginChat = true
	a.NeedEnabledMode = true
	a.NeedPagingOff = true
	a.EnableCommand = "enable"
	a.UsernamePromptPattern = `Username:\s*$`
	a.PasswordPromptPattern = `Password:\s*$`
	a.EnablePasswordPromptPattern = `Password:\s*$`
	a.DisabledPromptPattern = `\S+>\s*$`
	a.EnabledPromptPattern = `\S+#\s*$`
	a.CommandList = []string{"show ver br", "show run"}
	a.DisablePagerCommand = "term len 0"
	a.ReadTimeout = 10 * time.Second
	a.MatchTimeout = 20 * time.Second
	a.SendTimeout = 5 * time.Second
	a.CommandReadTimeout = 20 * time.Second  // larger timeout for slow 'sh run'
	a.CommandMatchTimeout = 30 * time.Second // larger timeout for slow 'sh run'
	a.QuoteSentCommandsFormat = `!![%s]`
	a.LineFilter = "iosxr" // line filter name - applied to every saved line

	m := &Model{name: "cisco-iosxr"}
	m.defaultAttr = a
	if err := t.SetModel(m, logger); err != nil {
		logger.Printf("registerModelCiscoIOSXR: %v", err)
	}
}
