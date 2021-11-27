package ui

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strings"

	"github.com/beanz/udin2mqtt-go/pkg/devices"

	"github.com/go-chi/chi"
)

var templates = template.Must(template.ParseFiles("index.html"))

type UI struct {
	Devices *devices.Devices
	Version string
	Seed    int64
}

func NewUI(d *devices.Devices, ver string, seed int64) *UI {
	return &UI{
		Devices: d,
		Version: ver,
		Seed:    seed,
	}
}

func (ui *UI) CreateRouter(stdout io.Writer, ch chan UIEvent) *chi.Mux {
	router := chi.NewRouter()
	router.Get("/", ui.getIndexHandler(stdout))
	router.Route("/api", func(r chi.Router) {
		r.Get("/create/{def}", ui.getCreateHandler(stdout, ch))
		r.Get("/{device}/enable/{val}", ui.getEnableDisableHandler(stdout, ch))
		r.Get("/{device}/alias/{val}", ui.getAliasHandler(stdout, ch))
	})
	fs := http.FileServer(http.Dir("static"))
	router.Handle("/static/*", http.StripPrefix("/static/", fs))
	return router
}

func (ui *UI) getIndexHandler(stdout io.Writer) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		err := templates.ExecuteTemplate(w, "index.html", ui)
		if err != nil {
			fmt.Fprintf(stdout, "http error: %+v\n", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
	}
}

func (ui *UI) getCreateHandler(stdout io.Writer, ch chan UIEvent) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		def := strings.Split(chi.URLParam(r, "def"), ",")
		ch <- NewUIEvent(UICreateEvent, def...)
		_, err := w.Write([]byte(
			"{\"status\":\"ok\",\"message\":\"creating device\"}"))
		if err != nil {
			fmt.Fprintf(stdout,
				"enable/disable request write failed: %+v\n", err)
		}
	}
}

func (ui *UI) getEnableDisableHandler(stdout io.Writer, ch chan UIEvent) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		device := chi.URLParam(r, "device")
		val := chi.URLParam(r, "val")
		var action string
		if val == "true" {
			action = "enabled"
		} else {
			action = "disabled"
		}
		ch <- NewUIEvent(UIEnableEvent, device, val)
		_, err := w.Write([]byte(fmt.Sprintf(
			"{\"status\":\"ok\",\"message\":\"device %s %s\"}",
			device, action)))
		if err != nil {
			fmt.Fprintf(stdout,
				"enable/disable request write failed: %+v\n", err)
		}
	}
}

func (ui *UI) getAliasHandler(stdout io.Writer, ch chan UIEvent) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		device := chi.URLParam(r, "device")
		val := chi.URLParam(r, "val")
		ch <- NewUIEvent(UIRenameEvent, device, val)
		_, err := w.Write([]byte(fmt.Sprintf(
			"{\"status\":\"ok\",\"message\":\"set name of %s to %s\"}",
			device, val)))
		if err != nil {
			fmt.Fprintf(stdout, "rename request write failed: %+v\n", err)
		}
	}
}
