package dev

import (
	"fmt"
	"sync"
)

// DeviceTable: goroutine concurrency-safe DeviceTable
type DeviceTable struct {
	models  map[string]*Model  // label => model
	devices map[string]*Device // id => device
	lock    sync.RWMutex
}

func NewDeviceTable() *DeviceTable {
	return &DeviceTable{models: map[string]*Model{}, devices: map[string]*Device{}, lock: sync.RWMutex{}}
}

func (t *DeviceTable) GetModel(modelName string) (*Model, error) {
	if m, ok := t.models[modelName]; ok {
		return m, nil
	}
	return nil, fmt.Errorf("GetModel: not found")
}

func (t *DeviceTable) SetModel(m *Model) error {
	if _, found := t.models[m.name]; found {
		return fmt.Errorf("app.SetModel: found")
	}
	t.models[m.name] = m
	return nil
}

func (t *DeviceTable) SetDevice(id string, d *Device) error {
	if _, found := t.devices[id]; found {
		return fmt.Errorf("app.SetDevice: found")
	}
	t.devices[id] = d
	return nil
}

func (t *DeviceTable) ListDevices() []*Device {
	return DeviceMapToSlice(t.devices)
}
