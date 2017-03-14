package conf

import (
	"sync"
)

// Options provides concurrency-safe access to AppConfig.
type Options struct {
	options AppConfig
	lock    sync.RWMutex
}

// NewOptions creates a new set of options.
func NewOptions() *Options {
	return &Options{}
}

// Get creates a copy of AppConfig.
func (o *Options) Get() *AppConfig {
	o.lock.RLock()
	defer o.lock.RUnlock()
	opt := o.options // clone
	return &opt
}

// Set updates the AppConfig from a copy.
func (o *Options) Set(c *AppConfig) {
	o.lock.Lock()
	defer o.lock.Unlock()
	o.options = *c // clone
}
