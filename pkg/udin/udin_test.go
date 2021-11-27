package udin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Udin(t *testing.T) {
	u, err := NewUdin("mock", nil)
	assert.NoError(t, err)
	resp, err := u.Send(UdinRequest{UdinOn, 1})
	assert.NoError(t, err)
	assert.Equal(t, "n1", resp)
	resp, err = u.Send(UdinRequest{UdinOff, 2})
	assert.NoError(t, err)
	assert.Equal(t, "f2", resp)
	resp, err = u.Send(UdinRequest{UdinQuery, 0})
	assert.NoError(t, err)
	assert.Equal(t, "UDIN-8R 8 x Relay V1.0", resp)
}
