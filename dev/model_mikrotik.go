package dev

import (
	"time"

	"github.com/udhos/jazigo/conf"
)

func registerModelMikrotik(logger hasPrintf, t *DeviceTable) {
	a := conf.NewDevAttr()

	promptPattern := `\[[^\[\]]+]\]\s*>\s*`

	a.NeedLoginChat = true
	a.UsernamePromptPattern = `Login:\s*$`
	a.PasswordPromptPattern = `Password:\s*$`
	a.SendExtraPostPasswordNewline = true
	a.DisabledPromptPattern = promptPattern
	a.EnabledPromptPattern = promptPattern
	a.CommandList = []string{"/system resource print", "/export", "/export verbose"}
	a.DisablePagerCommand = "term len 0"
	a.ReadTimeout = 10 * time.Second
	a.MatchTimeout = 20 * time.Second
	a.SendTimeout = 5 * time.Second
	a.CommandReadTimeout = 20 * time.Second  // larger timeout for slow 'sh run'
	a.CommandMatchTimeout = 30 * time.Second // larger timeout for slow 'sh run'
	a.QuoteSentCommandsFormat = `##[%s]`

	m := &Model{name: "mikrotik"}
	m.defaultAttr = a
	if err := t.SetModel(m, logger); err != nil {
		logger.Printf("registerModelMikrotik: %v", err)
	}
}
