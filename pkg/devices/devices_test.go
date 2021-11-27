package devices

import (
	"testing"

	"github.com/beanz/udin2mqtt-go/pkg/udin"
	"github.com/stretchr/testify/assert"
)

func Test_Devices(t *testing.T) {
	u8r, err := udin.NewUdin("mock", nil)
	assert.NoError(t, err)
	u44, err := udin.NewUdin("mock:UDIN-44", nil)
	udins := map[string]*udin.UdinDevice{
		"udin_8r": u8r,
		"udin_44": u44,
	}
	devs := NewDevices(udins)
	assert.Equal(t, []string{
		"udin_44-r1",
		"udin_44-r2",
		"udin_44-r3",
		"udin_44-r4",
		"udin_8r-r1",
		"udin_8r-r2",
		"udin_8r-r3",
		"udin_8r-r4",
		"udin_8r-r5",
		"udin_8r-r6",
		"udin_8r-r7",
		"udin_8r-r8",
	}, devs.Relays())
	assert.Equal(t, []string{"MomentaryOpenClose"}, devs.Types())
}

func Test_Create(t *testing.T) {
	u8r, err := udin.NewUdin("mock", nil)
	assert.NoError(t, err)
	udins := map[string]*udin.UdinDevice{
		"udin_8r": u8r,
	}
	devs := NewDevices(udins)
	dev, err := devs.Create(
		[]string{"foobar", "0", "udin_8r-r1", "udin_8r-r2"}, false)
	assert.Equal(t, "foobar", dev.Name)

	devList := devs.Devices()
	assert.Equal(t, []*Device{dev}, devList)
}
