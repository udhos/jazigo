package dev

import (
	"fmt"
	"sync"
)

// DeviceTable: goroutine concurrency-safe DeviceTable.
// Data is fully copied when either entering or leaving DeviceTable.
// Data is not shared with pointers.
type DeviceTable struct {
	models  map[string]*Model  // label => model
	devices map[string]*Device // id => device
	lock    sync.RWMutex
}

func NewDeviceTable() *DeviceTable {
	return &DeviceTable{models: map[string]*Model{}, devices: map[string]*Device{}, lock: sync.RWMutex{}}
}

func (t *DeviceTable) GetModel(modelName string) (*Model, error) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	if m, ok := t.models[modelName]; ok {
		m1 := *m // force copy data
		return &m1, nil
	}

	return nil, fmt.Errorf("CopyModel: not found")
}

func (t *DeviceTable) SetModel(m *Model) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	if _, found := t.models[m.name]; found {
		return fmt.Errorf("app.SetModel: found")
	}
	m1 := *m // force copy data
	t.models[m1.name] = &m1
	return nil
}

func (t *DeviceTable) SetDevice(id string, d *Device) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	if _, found := t.devices[id]; found {
		return fmt.Errorf("app.SetDevice: found")
	}
	d1 := *d // force copy data
	t.devices[id] = &d1
	return nil
}

func (t *DeviceTable) ListDevices() []*Device {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return copyDeviceMapToSlice(t.devices)
}

func copyDeviceMapToSlice(m map[string]*Device) []*Device {
	devices := make([]*Device, len(m))
	i := 0
	for _, d := range m {
		d1 := *d // force copy data
		devices[i] = &d1
		i++
	}
	return devices
}
