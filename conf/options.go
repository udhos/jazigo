package conf

import (
	"sync"
)

type Options struct {
	options AppConfig
	lock    sync.RWMutex
}

func NewOptions() *Options {
	return &Options{}
}

func (o *Options) Get() *AppConfig {
	o.lock.RLock()
	defer o.lock.RUnlock()
	opt := o.options // clone
	return &opt
}

func (o *Options) Set(c *AppConfig) {
	o.lock.Lock()
	defer o.lock.Unlock()
	o.options = *c // clone
}
