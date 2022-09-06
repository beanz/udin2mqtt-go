package ui

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/beanz/udin2mqtt-go/pkg/devices"
	"github.com/beanz/udin2mqtt-go/pkg/udin"

	"github.com/stretchr/testify/assert"
)

type BrokenWriter struct{}

func (w BrokenWriter) Header() http.Header {
	return http.Header{}
}

func (w BrokenWriter) Write(p []byte) (n int, err error) {
	return 0, io.ErrClosedPipe
}

func (w BrokenWriter) WriteHeader(statusCode int) {}

func (w BrokenWriter) Result() *http.Response {
	return nil
}

func Test_Router(t *testing.T) {
	tests := []struct {
		name   string
		uri    string
		checks func(*testing.T, *UI, string, chan UIEvent)
	}{
		{
			name: "index test",
			uri:  "/",
			checks: func(t *testing.T, ui *UI, body string, ch chan UIEvent) {
				assert.Contains(t, body,
					"<link rel=\"stylesheet\" href=\"/static/style.css?v=987654321\" />",
					"must contain stylesheet with seed",
				)
				assert.Contains(t, body,
					"src=\"/static/form.js?v=987654321\"",
					"must contain javascript reference with seed",
				)
				assert.Contains(t, body,
					"<div>App v0.0.1</div>", "must contain version reference")
			},
		},
		{
			name: "javascript test",
			uri:  "/static/form.js?v=1234",
			checks: func(t *testing.T, ui *UI, body string, ch chan UIEvent) {
				assert.Contains(t, body, "function load")
				assert.Empty(t, ch, "event channel should be empty")
			},
		},
		{
			name: "enable request",
			uri:  "/api/foo/enable/true",
			checks: func(t *testing.T, ui *UI, body string, ch chan UIEvent) {
				assert.Equal(t,
					"{\"status\":\"ok\",\"message\":\"device foo enabled\"}",
					body)
				assert.NotEmpty(t, ch, "event channel should not be empty")
				assert.Equal(t, <-ch,
					NewUIEvent(UIEnableEvent, "foo", "true"))
			},
		},
		{
			name: "disable request",
			uri:  "/api/bar/enable/false",
			checks: func(t *testing.T, ui *UI, body string, ch chan UIEvent) {
				assert.Equal(t,
					"{\"status\":\"ok\",\"message\":\"device bar disabled\"}",
					body)
				assert.NotEmpty(t, ch, "event channel should not be empty")
				assert.Equal(t, <-ch,
					NewUIEvent(UIEnableEvent, "bar", "false"))
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := devices.NewDevices(map[string]*udin.UdinDevice{})
			d.Update(devices.Device{Name: "foo"})
			d.Update(devices.Device{Name: "bar", Enabled: true})
			ui := NewUI(d, "0.0.1", 987654321)
			req := httptest.NewRequest(http.MethodGet, tc.uri, nil)
			w := httptest.NewRecorder()
			var buf bytes.Buffer
			ch := make(chan UIEvent, 1)
			router := ui.CreateRouter(&buf, ch)
			router.ServeHTTP(w, req)
			res := w.Result()
			defer res.Body.Close()
			data, err := ioutil.ReadAll(res.Body)
			assert.NoError(t, err)
			assert.Equal(t, "", buf.String())
			body := string(data)
			tc.checks(t, ui, body, ch)
		})
	}
}

func Test_Errors(t *testing.T) {
	tests := []struct {
		name  string
		uri   string
		error string
	}{
		{
			name:  "index error",
			uri:   "/",
			error: "http error:",
		},
		{
			name:  "enable error",
			uri:   "/api/foo/enable/true",
			error: "enable/disable request write failed",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := devices.NewDevices(map[string]*udin.UdinDevice{})
			d.Update(devices.Device{Name: "foo"})
			d.Update(devices.Device{Name: "bar", Enabled: true})
			ui := NewUI(d, "0.0.1", 987654321)
			req := httptest.NewRequest(http.MethodGet, tc.uri, nil)
			w := BrokenWriter{}
			var buf bytes.Buffer
			ch := make(chan UIEvent, 1)
			router := ui.CreateRouter(&buf, ch)
			router.ServeHTTP(w, req)
			assert.Contains(t, buf.String(), tc.error)
		})
	}
}
