package devices

import (
	"fmt"
	"strconv"
	"strings"

	mqtt "github.com/beanz/homeassistant-go/pkg/mqtt"
	ha "github.com/beanz/homeassistant-go/pkg/types"
	"github.com/beanz/udin2mqtt-go/pkg/types"
)

type RelayType int

const (
	MomentaryOpenClose RelayType = iota
	UnsupportedRelayType
)

func (r RelayType) String() string {
	switch r {
	case MomentaryOpenClose:
		return "momentaryopenclose"
	default:
		return "unsupportedrelaytype"
	}
}

type Device struct {
	Name    string
	Type    RelayType
	Def     []string
	Enabled bool
	Icon    string
}

type Action struct {
	Udin   string
	Relay  uint
	Action string
}

func (a *Action) String() string {
	return fmt.Sprintf("%s[%d].%s", a.Udin, a.Relay, a.Action)
}

func (d *Device) Command(cmd string) (*Action, error) {
	switch d.Type {
	case MomentaryOpenClose:
		var relay string
		switch strings.ToLower(cmd) {
		case "open":
			relay = d.Def[0]
		case "close":
			relay = d.Def[1]
		default:
			return nil, fmt.Errorf("invalid command on %s: %s", d.Name, cmd)
		}
		rs := strings.SplitN(relay, "-", 2)
		i, err := strconv.Atoi(rs[1][1:])
		if err != nil {
			return nil, fmt.Errorf("invalid instance %s: %w", rs[1], err)
		}
		return &Action{Udin: rs[0], Relay: uint(i), Action: "pulse"}, nil
	default:
		return nil, fmt.Errorf("unsupported device type for command on %s: %s",
			d.Name, d.Type)
	}
}

func (d *Device) DiscoveryMessage(cfg types.SimpleStringConfig) (*mqtt.Msg, error) {
	defaultVersion := fmt.Sprintf("%s v%s",
		cfg.GetString("App_Name"), cfg.GetString("Version"))
	defaultHADevice := ha.Device{
		Identifiers:      []string{d.Name},
		Name:             d.Name,
		SwVersion:        defaultVersion,
		ConfigurationURL: "http://" + cfg.GetString("UI_Advertise"),
	}
	defaultAvailability := []ha.Availability{
		{
			Topic: mqtt.AvailabilityTopic(
				cfg.GetString("Bridge_Topic"), "bridge",
			),
		},
	}
	icon := d.Icon
	if icon == "" {
		icon = "mdi:blinds"
	}
	switch d.Type {
	case MomentaryOpenClose:
		return &mqtt.Msg{
			Topic: fmt.Sprintf("%s/cover/%s/config",
				cfg.GetString("Discovery_Prefix"), d.Name),
			Body: ha.Cover{
				CommandTopic: fmt.Sprintf("%s/%s/set",
					cfg.GetString("Bridge_Topic"), d.Name),
				Device:       defaultHADevice,
				Availability: defaultAvailability,
				UniqueID:     d.Name,
				Icon:         icon,
			},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported device type on device %s: %v",
			d.Name, d.Type)
	}
}
