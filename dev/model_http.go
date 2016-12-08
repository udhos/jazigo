package dev

import (
	"time"

	"github.com/udhos/jazigo/conf"
)

func registerModelHTTP(logger hasPrintf, t *DeviceTable) {
	a := conf.NewDevAttr()

	a.CommandList = []string{"GET / HTTP/1.0\r\n\r\n"}
	a.ReadTimeout = 5 * time.Second
	a.MatchTimeout = 10 * time.Second
	a.SendTimeout = 5 * time.Second
	a.CommandReadTimeout = 5 * time.Second   // larger timeout for slow 'sh run'
	a.CommandMatchTimeout = 10 * time.Second // larger timeout for slow 'sh run'
	a.SupressAutoLF = true
	a.QuoteSentCommandsFormat = `[%s]`

	m := &Model{name: "http"}
	m.defaultAttr = a
	if err := t.SetModel(m, logger); err != nil {
		logger.Printf("registerModelHTTP: %v", err)
	}
}
