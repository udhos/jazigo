package dev

import (
	"time"

	"github.com/udhos/jazigo/conf"
)

func registerModelCiscoAPIC(logger hasPrintf, t *DeviceTable) {
	a := conf.NewDevAttr()

	promptPattern := `\S+#\s*$`

	a.DisabledPromptPattern = promptPattern
	a.EnabledPromptPattern = promptPattern
	a.CommandList = []string{"show ver", "conf", "terminal length 0", "show running-config"}
	a.ReadTimeout = 10 * time.Second
	a.MatchTimeout = 20 * time.Second
	a.SendTimeout = 5 * time.Second
	a.CommandReadTimeout = 20 * time.Second  // larger timeout for slow 'sh run'
	a.CommandMatchTimeout = 60 * time.Second // larger timeout for slow 'sh run'
	a.QuoteSentCommandsFormat = `!![%s]`

	m := &Model{name: "cisco-apic"}
	m.defaultAttr = a
	if err := t.SetModel(m, logger); err != nil {
		logger.Printf("registerModelCiscoAPIC: %v", err)
	}
}
