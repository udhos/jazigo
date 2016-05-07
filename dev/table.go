package dev

import (
	"fmt"
	"strconv"
	"strings"
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

type DeviceUpdater interface {
	GetDevice(id string) (*Device, error)
	UpdateDevice(d *Device) error
}

func NewDeviceTable() *DeviceTable {
	return &DeviceTable{models: map[string]*Model{}, devices: map[string]*Device{}, lock: sync.RWMutex{}}
}

func (t *DeviceTable) GetModel(modelName string) (*Model, error) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	if m, found := t.models[modelName]; found {
		m1 := *m // force copy data
		return &m1, nil
	}

	return nil, fmt.Errorf("DeviceTable.GetModel: not found")
}

func (t *DeviceTable) SetModel(m *Model, logger hasPrintf) error {

	logger.Printf("DeviceTable.SetModel: registering model: '%s'", m.name)

	t.lock.Lock()
	defer t.lock.Unlock()

	if _, found := t.models[m.name]; found {
		return fmt.Errorf("DeviceTable.SetModel: found")
	}

	m1 := *m // force copy data
	t.models[m1.name] = &m1
	return nil
}

func (t *DeviceTable) GetDevice(id string) (*Device, error) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	if d, found := t.devices[id]; found {
		d1 := *d // force copy data
		return &d1, nil
	}

	return nil, fmt.Errorf("DeviceTable.GetDevice: not found")
}

func (t *DeviceTable) SetDevice(d *Device) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	id := d.Id

	if _, found := t.devices[id]; found {
		return fmt.Errorf("DeviceTable.SetDevice: found")
	}
	d1 := *d // force copy data
	t.devices[id] = &d1
	return nil
}

func (t *DeviceTable) KillDevice(id string) {
	t.lock.Lock()
	defer t.lock.Unlock()

	delete(t.devices, id)
}

func (t *DeviceTable) UpdateDevice(d *Device) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	id := d.Id

	if _, found := t.devices[id]; !found {
		return fmt.Errorf("DeviceTable.UpdateDevice: not found")
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

func (t *DeviceTable) FindDeviceFreeId(prefix string) string {
	pLen := len(prefix)
	devices := t.ListDevices()
	highest := 0
	for _, d := range devices {
		id := d.Id
		if !strings.HasPrefix(id, prefix) {
			continue
		}
		suffix := id[pLen:]
		value, err := strconv.Atoi(suffix)
		if err != nil {
			continue
		}
		if value > highest {
			highest = value
		}
	}
	free := highest + 1
	return prefix + strconv.Itoa(free)
}
