package devices

import (
	"strings"
	"testing"

	mqtt "github.com/beanz/homeassistant-go/pkg/mqtt"
	ha "github.com/beanz/homeassistant-go/pkg/types"

	"github.com/stretchr/testify/assert"
)

type MockCfg map[string]string

func (cfg MockCfg) GetString(k string) string {
	return cfg[k]
}

func Test_DiscoveryMessage(t *testing.T) {
	tests := []struct {
		name    string
		dev     Device
		cfg     []string
		idx     int
		want    *mqtt.Msg
		wantErr bool
	}{
		{
			name: "simple blind",
			dev: Device{
				Name: "blind1",
			},
			cfg: []string{
				"App_Name=app",
				"Version=0.0.1",
				"Bridge_Topic=foo",
				"Discovery_Prefix=baz",
				"UI_Advertise=10.0.0.1:8094",
			},
			want: &mqtt.Msg{
				Topic: "baz/cover/blind1/config",
				Body: ha.Cover{
					Availability: []ha.Availability{
						{
							Topic: "foo/bridge/availability",
						},
					},
					Device: ha.Device{
						Identifiers:      []string{"blind1"},
						Name:             "blind1",
						ConfigurationURL: "http://10.0.0.1:8094",
						SwVersion:        "app v0.0.1",
					},
					UniqueID:     "blind1",
					CommandTopic: "foo/blind1/set",
				},
			},
		},
	}
	for _, tc := range tests {
		cfg := make(MockCfg)
		for _, c := range tc.cfg {
			kv := strings.Split(c, "=")
			cfg[kv[0]] = kv[1]
		}
		t.Run(tc.name, func(t *testing.T) {
			msg, err := tc.dev.DiscoveryMessage(cfg)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, msg, tc.want)
		})
	}
}
