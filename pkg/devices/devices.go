package devices

import (
	"fmt"
	"sort"
	"sync"

	"github.com/beanz/udin2mqtt-go/pkg/udin"
)

type Devices struct {
	relays []string
	types  []string
	dev    map[string]*Device
	mu     sync.Mutex
}

func NewDevices(udins map[string]*udin.UdinDevice) *Devices {
	relays := []string{}
	for name, dev := range udins {
		var i uint
		for i = 1; i <= dev.NumRelays(); i++ {
			relays = append(relays, fmt.Sprintf("%s-r%d", name, i))
		}
	}
	sort.Strings(relays)
	return &Devices{
		relays: relays,
		types:  []string{"MomentaryOpenClose"},
		dev:    make(map[string]*Device),
	}
}

func kindFromArg(kind string) (RelayType, error) {
	switch kind {
	case "0", "momentaryopenclose":
		return MomentaryOpenClose, nil
	}
	return UnsupportedRelayType, fmt.Errorf("invalid relay type: %s", kind)
}

func (d *Devices) Create(def []string, enabled bool, icon string) (*Device, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	name := def[0]
	t, err := kindFromArg(def[1])
	if err != nil {
		return nil, err
	}
	d.dev[name] = &Device{
		Name:    name,
		Type:    t,
		Def:     def[2:],
		Enabled: enabled,
		Icon:    icon,
	}
	return d.dev[name], nil
}

func (d *Devices) Update(n Device) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.dev[n.Name] = &n
}

func (d *Devices) EnableDisable(name string, val bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.dev[name].Enabled = val
}

func (d *Devices) Device(name string) *Device {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.dev[name]
}

func (d *Devices) Devices() []*Device {
	res := []*Device{}
	for _, dev := range d.dev {
		res = append(res, dev)
	}
	return res
}

func (d *Devices) Relays() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.relays
}

func (d *Devices) Types() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.types
}

func (d *Devices) ActionForDevice(name, cmd string) (*Action, error) {
	dev := d.Device(name)
	if dev == nil {
		return nil, fmt.Errorf("invalid device %s", name)
	}
	act, err := dev.Command(cmd)
	if err != nil {
		return nil, fmt.Errorf("invalid action on device %s: %w", name, err)
	}
	return act, nil
}
