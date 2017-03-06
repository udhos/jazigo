package dev

import (
	"time"

	"github.com/udhos/jazigo/conf"
)

func registerModelFortiOS(logger hasPrintf, t *DeviceTable) {
	a := conf.NewDevAttr()

	promptPattern := `\S+\s#\s$` // "hostname # "

	a.NeedPagingOff = true
	a.DisablePagerCommand = "config system console\nset output standard\nend"
	a.DisablePagerExtraPromptCount = 2
	a.DisabledPromptPattern = promptPattern
	a.EnabledPromptPattern = promptPattern
	a.CommandList = []string{"get system status", "show"}
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
