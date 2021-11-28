package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	devs "github.com/beanz/udin2mqtt-go/pkg/devices"
	"github.com/beanz/udin2mqtt-go/pkg/udin"
	"github.com/beanz/udin2mqtt-go/pkg/ui"

	mqtt "github.com/beanz/homeassistant-go/pkg/mqtt"
	// ha "github.com/beanz/homeassistant-go/pkg/types"

	"github.com/spf13/viper"
)

const appName = "udin2mqtt"

// Version is overridden at build time
var Version = "0.0.1+Dev"

const (
	// exitFail is the exit code if the program
	// fails.
	exitFail = 1
)

func main() {
	if err := run(os.Args, os.Stdout, viper.New()); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(exitFail)
	}
}

func run(args []string, stdout io.Writer, v *viper.Viper) error {
	if len(args) == 2 && args[1] == "--version" {
		fmt.Fprintf(stdout, "%s v%s\n", appName, Version)
		os.Exit(0)
	}
	v.Set("Version", Version)
	v.SetDefault("Broker", "tcp://127.0.0.1:1883")
	v.SetDefault("UI", "0.0.0.0:8094")
	v.SetDefault("UI_Advertise", "")
	v.SetDefault("Discovery_Prefix", "homeassistant")
	v.SetDefault("Devices", []string{"mock"})
	v.SetDefault("Bridge_Topic", appName)
	v.SetDefault("Resend_Time", time.Minute*10)
	v.SetDefault("Client_ID", appName)
	v.SetDefault("KeepAlive", 30)
	v.SetDefault("Connect_Retry_Delay", 10*time.Second)
	v.SetDefault("Verbose", 0)
	v.SetDefault("App_Name", appName)
	v.SetDefault("device", map[string]interface{}{})
	v.SetConfigName(appName)
	v.SetConfigType("yaml")
	v.AddConfigPath("/etc/" + appName)
	v.AddConfigPath(".")
	v.AutomaticEnv()
	err := v.ReadInConfig() // Find and read the config file
	if err != nil {         // Handle errors reading the config file
		return fmt.Errorf("config file error: %+v", err)
	}

	if v.GetString("UI_Advertise") == "" {
		v.Set("UI_Advertise", v.GetString("UI"))
	}

	if v.GetInt("Verbose") > 0 {
		settings, err := json.MarshalIndent(v.AllSettings(), "", "  ")
		if err != nil {
			return fmt.Errorf("config marshal error: %+v", err)
		}
		fmt.Fprintf(stdout, "Settings:\n%s\n", settings)
	}
	logger := log.New(stdout, "", log.Ldate|log.Ltime|log.Lmicroseconds)

	udinTtys := v.GetStringSlice("Devices")
	udins := make(map[string]*udin.UdinDevice, len(udinTtys))
	var udinLogger *log.Logger
	if v.GetInt("Verbose") > 0 {
		udinLogger = logger
	}
	for _, tty := range udinTtys {
		u, err := udin.NewUdin(tty, udinLogger)
		if err != nil {
			return fmt.Errorf("failed to open udin device %s: %+v", tty, err)
		}
		name := uidSafe(u.Name())
		logger.Printf("found UDIN device %s\n", u)
		udins[name] = u
	}

	// Set up channel on which to send signal notifications.
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt)
	signal.Notify(sigc, syscall.SIGTERM)

	uic := make(chan ui.UIEvent, 10)
	msgp := make(chan *mqtt.Msg, 300)
	msgs := make(chan *mqtt.Msg, 50)
	errCh := make(chan error, 1)

	devices := devs.NewDevices(udins)
	for name := range v.GetStringMap("device") {
		args := []string{name, v.GetString("device." + name + ".kind")}
		args = append(args, v.GetStringSlice("device."+name+".def")...)
		enabled := v.GetBool("device." + name + ".enabled")
		icon := v.GetString("device." + name + ".icon")
		dev, err := devices.Create(args, enabled, icon)
		if err != nil {
			return fmt.Errorf("unable to create device %s: %+v", name, err)
		}
		logger.Printf("loaded device %v\n", dev)
		if !enabled {
			continue
		}
		msg, err := dev.DiscoveryMessage(v)
		if err != nil {
			return fmt.Errorf("failed to generate discovery message: %w",
				err)
		}
		msg.Retain = true
		msgp <- msg
	}

	uiRouter := ui.NewUI(devices, Version,
		time.Now().Unix()).CreateRouter(stdout, uic)

	srv := &http.Server{
		Addr:           v.GetString("UI"),
		Handler:        uiRouter,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			logger.Printf("UI server error: %v\n", err)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func(ctx context.Context, errCh chan error) {
		mqttc, err := mqtt.NewClient(&mqtt.ClientConfig{
			AppName:              v.GetString("App_Name"),
			Version:              Version,
			Debug:                v.GetInt("Verbose") > 0,
			Log:                  logger,
			Broker:               v.GetString("Broker"),
			ClientID:             v.GetString("Client_ID"),
			DataTopicPrefix:      v.GetString("Bridge_Topic"),
			DiscoveryTopicPrefix: v.GetString("Discovery_Prefix"),
			ConnectRetryDelay:    v.GetDuration("Connect_Retry_Delay"),
			KeepAlive:            int16(v.GetInt("KeepAlive")),
			Subs: []mqtt.Sub{
				{
					Topic: v.GetString("Bridge_Topic") + "/+/set",
					QoS:   1,
				},
			},
		}, logger)
		if err != nil {
			errCh <- fmt.Errorf("Failed to create MQTT client: %w", err)
			return
		}
		errCh <- mqttc.Run(ctx, msgp, msgs)
	}(ctx, errCh)

LOOP:
	for {
		select {
		case <-sigc:
			err = nil
			cancel()
			break LOOP
		case err = <-errCh:
			break LOOP
		case uie := <-uic:
			switch uie.Kind {
			case ui.UIEnableEvent:
				logger.Printf("enable %s %s\n", uie.Args[0], uie.Args[1])
				val := uie.Args[1] == "true"
				devices.EnableDisable(uie.Args[0], val)
				v.Set("device."+uie.Args[0]+".enabled", val)
				err = v.WriteConfig()
				if err != nil {
					return fmt.Errorf("failed to write config: %+v", err)
				}
				if !val {
					continue
				}
				msg, err := devices.Device(uie.Args[0]).DiscoveryMessage(v)
				if err != nil {
					logger.Printf("failed to generate discovery message: %s",
						err)
					continue
				}
				msg.Retain = true
				msgp <- msg
			case ui.UICreateEvent:
				dev, err := devices.Create(uie.Args, false, "")
				if err != nil {
					fmt.Fprintf(stdout, "failed to create device: %+v\n", err)
					continue
				}
				v.Set("device."+dev.Name+".kind", dev.Type.String())
				v.Set("device."+dev.Name+".def", dev.Def)
				err = v.WriteConfig()
				if err != nil {
					return fmt.Errorf("failed to write config: %+v", err)
				}
				logger.Printf("loaded device %v\n", dev)
			}
		case msg := <-msgs:

			topic := msg.Topic
			cmd := string(msg.Body.([]byte))
			logger.Printf("mqtt < %s: %s\n", topic, cmd)
			ts := strings.Split(topic, "/")
			devName := ts[len(ts)-2]
			act, err := devices.ActionForDevice(devName, cmd)
			if err != nil {
				logger.Printf("command failed: %s\n", err)
				continue
			}
			logger.Printf("Found action: %s\n", act)
			u := udins[act.Udin]
			if u == nil {
				logger.Printf("invalid UDIN %s for %s\n", act.Udin, devName)
				continue
			}
			switch act.Action {
			case "pulse":
				go func() {
					err := u.Pulse(act.Relay, time.Second)
					if err != nil {
						logger.Printf("failed to pulse relay %d on %s: %s\n",
							act.Relay, act.Udin, err)
					}
				}()
			default:
				logger.Printf("invalid UDIN action %s for %s\n",
					act.Action, devName)
				continue
			}
		}
	}

	logger.Println("shutting down")

	if err != nil {
		return err
	}

	// TOFIX: shutdown the ui
	if err := srv.Shutdown(context.Background()); err != nil {
		// Error from closing listeners, or context timeout:
		logger.Printf("error shutting down UI: %v\n", err)
	}

	return nil
}

func uidSafe(s string) string {
	r := strings.ReplaceAll(s, "/", "_slash_")
	r = strings.ReplaceAll(r, "#", "_hash_")
	r = strings.ReplaceAll(r, "+", "_plus_")
	r = strings.ReplaceAll(r, "-", "_")
	r = strings.TrimLeft(r, "_")
	r = strings.TrimRight(r, "_")
	return r
}
