package udin

import (
	"bytes"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_UdinRequest_String(t *testing.T) {
	tests := []struct {
		req  UdinRequest
		want string
	}{
		{UdinRequest{Command: UdinQuery}, "?"},
		{UdinRequest{Command: UdinOn, Instance: 1}, "n1"},
		{UdinRequest{Command: UdinOff, Instance: 3}, "f3"},
		{UdinRequest{Command: UdinSet, Instance: 1}, "r1"},
		{UdinRequest{Command: UdinStatus, Instance: 4}, "s4"},
		{UdinRequest{Command: UdinInput, Instance: 2}, "i2"},
		{UdinRequest{Command: UdinCommand(99), Instance: 2},
			"unknown command"},
	}
	for _, test := range tests {
		assert.Equal(t, test.want, test.req.String())
	}
}

func Test_NumRelays(t *testing.T) {
	tests := []struct {
		mock string
		want uint
	}{
		{"mock", 8},
		{"mock:UDIN-8R 8 x Relay V1.0", 8},
		{"mock:UDIN-44", 4},
		{"mock:UDIN-8I", 0},
	}
	for _, test := range tests {
		t.Run(test.mock, func(t *testing.T) {
			u, err := NewUdin(test.mock, nil)
			assert.NoError(t, err)
			assert.Equal(t, test.want, u.NumRelays())
		})
	}
}

func Test_NumInputs(t *testing.T) {
	tests := []struct {
		mock string
		want uint
	}{
		{"mock", 0},
		{"mock:UDIN-8R 8 x Relay V1.0", 0},
		{"mock:UDIN-44", 4},
		{"mock:UDIN-8I", 8},
	}
	for _, test := range tests {
		t.Run(test.mock, func(t *testing.T) {
			u, err := NewUdin(test.mock, nil)
			assert.NoError(t, err)
			assert.Equal(t, test.want, u.NumInputs())
		})
	}
}

func Test_Name(t *testing.T) {
	tests := []struct {
		mock string
		want string
	}{
		{"mock", "udin-8r"},
		{"mock:UDIN-8R 8 x Relay V1.0", "udin-8r"},
		{"mock:UDIN-44", "udin-44"},
		{"mock:UDIN-8I", "udin-8i"},
	}
	for _, test := range tests {
		t.Run(test.mock, func(t *testing.T) {
			u, err := NewUdin(test.mock, nil)
			assert.NoError(t, err)
			assert.Equal(t, test.want, u.Name())
		})
	}
}

func Test_Model(t *testing.T) {
	tests := []struct {
		mock string
		want string
	}{
		{"mock", "UDIN-8R 8 x Relay V1.0"},
		{"mock:UDIN-8R 8 x Relay V1.0", "UDIN-8R 8 x Relay V1.0"},
		{"mock:UDIN-44", "UDIN-44"},
		{"mock:UDIN-8I", "UDIN-8I"},
	}
	for _, test := range tests {
		t.Run(test.mock, func(t *testing.T) {
			u, err := NewUdin(test.mock, nil)
			assert.NoError(t, err)
			assert.Equal(t, test.want, u.Model())
		})
	}
}

func Test_String(t *testing.T) {
	tests := []struct {
		mock string
		want string
	}{
		{"mock", "udin-8r: UDIN-8R 8 x Relay V1.0 (r=8 i=0)"},
		{"mock:UDIN-8R 8 x Relay V1.0",
			"udin-8r: UDIN-8R 8 x Relay V1.0 (r=8 i=0)"},
		{"mock:UDIN-44", "udin-44: UDIN-44 (r=4 i=4)"},
		{"mock:UDIN-8I", "udin-8i: UDIN-8I (r=0 i=8)"},
	}
	for _, test := range tests {
		t.Run(test.mock, func(t *testing.T) {
			u, err := NewUdin(test.mock, nil)
			assert.NoError(t, err)
			assert.Equal(t, test.want, u.String())
		})
	}
}

func Test_UnsupportedModel(t *testing.T) {
	_, err := NewUdin("mock:ACME-00", nil)
	assert.Error(t, err)
}

func Test_Udin(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)
	u, err := NewUdin("mock", logger)
	defer u.Close()
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
	assert.Equal(t, `read: ? [63 13 10]
read model: UDIN-8R 8 x Relay V1.0 [85 68 73 78 45 56 82 32 56 32 120 32 82 101 108 97 121 32 86 49 46 48 13 10]
found device udin-8r: UDIN-8R 8 x Relay V1.0
read: n1 [110 49 13 10]
read: f2 [102 50 13 10]
read: ? [63 13 10]
read model: UDIN-8R 8 x Relay V1.0 [85 68 73 78 45 56 82 32 56 32 120 32 82 101 108 97 121 32 86 49 46 48 13 10]
`,
		buf.String())
}
