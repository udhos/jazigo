package dev

import (
	"time"

	"github.com/udhos/jazigo/conf"
)

func registerModelCiscoNGA(logger hasPrintf, t *DeviceTable) {
	a := conf.NewDevAttr()

	promptPattern := `\S+#\s*$`

	a.DisabledPromptPattern = promptPattern
	a.EnabledPromptPattern = promptPattern
	a.CommandList = []string{"terminal length 0", "show ver", "show conf"}
	a.ReadTimeout = 10 * time.Second
	a.MatchTimeout = 20 * time.Second
	a.SendTimeout = 5 * time.Second
	a.CommandReadTimeout = 15 * time.Second  // larger timeout for slow 'sh run'
	a.CommandMatchTimeout = 25 * time.Second // larger timeout for slow 'sh run'
	a.QuoteSentCommandsFormat = `!![%s]`

	m := &Model{name: "cisco-nga"}
	m.defaultAttr = a
	if err := t.SetModel(m, logger); err != nil {
		logger.Printf("registerModelCiscoNGA: %v", err)
	}
}
