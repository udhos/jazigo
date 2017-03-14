package dev

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
)

// DeviceTable is goroutine concurrency-safe list of devices.
// Data is fully copied when either entering or leaving DeviceTable.
// Data is not shared with pointers.
type DeviceTable struct {
	models  map[string]*Model  // label => model
	devices map[string]*Device // id => device
	lock    sync.RWMutex
}

// DeviceUpdater is helper interface for a device store which can provide and update device information.
type DeviceUpdater interface {
	GetDevice(id string) (*Device, error)
	UpdateDevice(d *Device) error
}

// NewDeviceTable creates a device table.
func NewDeviceTable() *DeviceTable {
	return &DeviceTable{models: map[string]*Model{}, devices: map[string]*Device{}, lock: sync.RWMutex{}}
}

// GetModel looks up a model in the device table.
func (t *DeviceTable) GetModel(modelName string) (*Model, error) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	if m, found := t.models[modelName]; found {
		m1 := *m // force copy data
		return &m1, nil
	}

	return nil, fmt.Errorf("DeviceTable.GetModel: not found")
}

// SetModel adds a model to the device table.
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

// GetDevice finds a device in the device table.
func (t *DeviceTable) GetDevice(id string) (*Device, error) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	if d, found := t.devices[id]; found {
		d1 := *d // force copy data
		return &d1, nil
	}

	return nil, fmt.Errorf("DeviceTable.GetDevice: not found")
}

// SetDevice adds a device into the device table.
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

// DeleteDevice sets a device as deleted in the device table.
func (t *DeviceTable) DeleteDevice(id string) {
	t.lock.Lock()
	defer t.lock.Unlock()

	if d, found := t.devices[id]; found {
		d.Deleted = true
	}
}

// PurgeDevice actually removes a device from the device table.
func (t *DeviceTable) PurgeDevice(id string) {
	t.lock.Lock()
	defer t.lock.Unlock()

	delete(t.devices, id)
}

// UpdateDevice updates device info in the device table.
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

// ListDevices gets the list of devices.
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

// FindDeviceFreeID finds an ID available for a new device.
func (t *DeviceTable) FindDeviceFreeID(prefix string) string {
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

// ListModels gets the list of models.
func (t *DeviceTable) ListModels() []string {
	t.lock.RLock()
	defer t.lock.RUnlock()

	models := make([]string, len(t.models))
	i := 0
	for name := range t.models {
		models[i] = name
		i++
	}
	return models
}
