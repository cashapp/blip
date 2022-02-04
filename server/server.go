// Copyright 2022 Block, Inc.

// Package server provides the core runtime for Blip.
package server

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/cashapp/blip"
	"github.com/cashapp/blip/aws"
	"github.com/cashapp/blip/dbconn"
	"github.com/cashapp/blip/event"
	"github.com/cashapp/blip/metrics"
	"github.com/cashapp/blip/monitor"
	"github.com/cashapp/blip/plan"
	"github.com/cashapp/blip/sink"
	"github.com/cashapp/blip/status"
)

// ControlChans is a convenience function to return arguments for Run.
func ControlChans() (stopChan, doneChan chan struct{}) {
	return make(chan struct{}), make(chan struct{})
}

// Defaults returns the default environment, plugins, and factories. It is used
// in main.go as the args to Server.Boot. Third-party integration will likely
// _not_ call this function and, instead, provide its own environment, plugins,
// or factories when calling Server.Boot.
func Defaults() (blip.Env, blip.Plugins, blip.Factories) {
	factories := blip.Factories{
		AWSConfig: &aws.ConfigFactory{},
		// DbConn made after loading config
		// HTTPClient made after loading config
	}
	env := blip.Env{
		Args: os.Args,
		Env:  os.Environ(),
	}
	return env, blip.Plugins{}, factories
}

type httpClientFactory struct {
	cfg blip.ConfigHTTP
}

func (f httpClientFactory) MakeForSink(sinkName, monitorId string, opts, tags map[string]string) (*http.Client, error) {
	client := &http.Client{}
	if f.cfg.Proxy != "" {
		proxyFunc := func(req *http.Request) (url *url.URL, err error) {
			return url.Parse(f.cfg.Proxy)
		}
		client.Transport = &http.Transport{Proxy: proxyFunc}
		blip.Debug("%s sink %s http proxy via %s", monitorId, sinkName, f.cfg.Proxy)
	}
	return client, nil
}

// --------------------------------------------------------------------------

// Server is the core runtime for one instance of Blip. It's responsible for
// starting and running everything: Boot and Run, respectively. As long as
// the server is running, Blip is running.
//
// See bin/blip/main.go for the simplest use case.
type Server struct {
	cfg           blip.Config
	cmdline       CommandLine
	planLoader    *plan.Loader
	monitorLoader *monitor.Loader
	api           *API
}

// Boot boots Blip. That means it loads, validates, and creates everything,
// but it doesn't start (run) anything--that happens when Run is called.
// If Boot returns nil, then Blip is ready to run; else, any error is fatal:
// sometimes is too wrong for Blip to run (for example, an invalid config).
//
// Boot must be called once before Run.
func (s *Server) Boot(env blip.Env, plugins blip.Plugins, factories blip.Factories) error {
	startTs := time.Now()
	fmt.Fprintf(os.Stdout, "blip %s\n", blip.VERSION)
	status.Blip("server", "booting")

	// ----------------------------------------------------------------------
	// Parse commad line options

	var err error
	s.cmdline, err = ParseCommandLine(env.Args)
	if err != nil {
		return err
	}
	if s.cmdline.Options.Version {
		fmt.Println("blip", blip.VERSION)
		return nil
	}
	if s.cmdline.Options.Help {
		printHelp()
		return nil
	}

	event.SetReceiver(event.Log{All: s.cmdline.Options.Log})
	event.Sendf(event.BOOT_START, "blip %s", blip.VERSION) // very first event

	// Set debug and strict from env vars. Do this very first because all code
	// uses blip.Debug() and blip.Strict (boolean).
	//
	// STRICT.md documents the effects of strict mode.
	if s.cmdline.Options.Debug {
		blip.Debugging = true
	}
	blip.Debug("cmdline: %+v", s.cmdline)
	if v := os.Getenv(blip.ENV_DEBUG); v != "" {
		switch strings.ToLower(v) {
		case "yes", "on", "enable", "1":
			blip.Debugging = true
		}
	}
	if s.cmdline.Options.Strict {
		blip.Strict = true
	}
	if v := os.Getenv(blip.ENV_STRICT); v != "" {
		switch strings.ToLower(v) {
		case "yes", "on", "enable", "1", "finch":
			blip.Strict = true
		}
	}

	// --print-domains and exit
	if s.cmdline.Options.PrintDomains {
		fmt.Fprintf(os.Stdout, metrics.PrintDomains())
		os.Exit(0)
	}

	// ----------------------------------------------------------------------
	// Load config
	event.Send(event.BOOT_CONFIG_LOADING)
	status.Blip("server", "boot: loading config")

	// Always start with a default config, else we'll lack some basic config
	// like the API addr:port to listen on. If strict, it's a zero config except
	// for the absolute most minimal/must-have config values. Else, the default
	// (not strict) config is a more realistic set of defaults.
	cfg := blip.DefaultConfig(blip.Strict)

	// User-provided LoadConfig plugin takes priority if set; else, use default
	// (built-in) LoadConfig func.
	if plugins.LoadConfig != nil {
		blip.Debug("call plugins.LoadConfig")
		cfg, err = plugins.LoadConfig(cfg)
	} else {
		cfg, err = blip.LoadConfig(s.cmdline.Options.Config, cfg)
	}
	if err != nil {
		event.Sendf(event.BOOT_ERROR, err.Error())
		return err
	}

	// Extensively validate the config. Once done, the config is immutable,
	// except for plans and monitors which might come from dynamic sources,
	// like tables.
	if err := cfg.Validate(); err != nil {
		event.Errorf(event.BOOT_CONFIG_INVALID, err.Error())
		return err
	}

	cfg.InterpolateEnvVars()
	s.cfg = cfg // final immutable config
	event.Send(event.BOOT_CONFIG_LOADED)

	if s.cmdline.Options.PrintConfig {
		printYAML(s.cfg)
	}

	// ----------------------------------------------------------------------
	// Finished and register Blip global factories

	// If HTTP factory not provided, then use default with config.http, which
	// is why its creation is delayed until now
	if factories.HTTPClient == nil {
		factories.HTTPClient = httpClientFactory{cfg: cfg.HTTP}
	}

	if factories.DbConn == nil {
		factories.DbConn = dbconn.NewConnFactory(factories.AWSConfig, plugins.ModifyDB)
	}

	sink.InitFactory(factories)
	metrics.InitFactory(factories)

	// ----------------------------------------------------------------------
	// Load level plans
	status.Blip("server", "boot: loading plans")

	// Get the built-in level plan loader singleton. It's used in two places:
	// here for initial plan loading, and level.Collector (LPC) to fetch the
	// plan and set the Monitor to use it.
	s.planLoader = plan.NewLoader(plugins.LoadPlans)

	if s.cmdline.Options.Plans != "" {
		plans := strings.Split(s.cmdline.Options.Plans, ",")
		blip.Debug("--plans override config.plans: %v -> %v", s.cfg.Plans.Files, plans)
		s.cfg.Plans.Files = plans
	}

	if err := s.planLoader.LoadShared(s.cfg.Plans, factories.DbConn); err != nil {
		event.Sendf(event.BOOT_ERROR, err.Error())
		return err
	}

	if s.cmdline.Options.PrintPlans {
		s.planLoader.Print()
	}

	// ----------------------------------------------------------------------
	// Database monitors
	status.Blip("server", "boot: load monitors")

	// Create, but don't start, database monitors. They're startTs later in Run.
	s.monitorLoader = monitor.NewLoader(monitor.LoaderArgs{
		Config:     s.cfg,
		Factories:  factories,
		Plugins:    plugins,
		PlanLoader: s.planLoader,
		RDSLoader:  aws.RDSLoader{ClientFactory: aws.NewRDSClientFactory(factories.AWSConfig)},
	})
	if err := s.monitorLoader.Load(context.Background()); err != nil {
		event.Sendf(event.BOOT_ERROR, err.Error())
		return err
	}

	if s.cmdline.Options.PrintMonitors {
		fmt.Println(s.monitorLoader.Print())
	}

	// ----------------------------------------------------------------------
	// API
	if !s.cfg.API.Disable {
		s.api = NewAPI(cfg.API, s.monitorLoader)
	} else {
		blip.Debug("API disabled")
	}

	event.Sendf(event.BOOT_SUCCESS, "booted in %s, loaded %d monitors", time.Now().Sub(startTs), s.monitorLoader.Count())
	return nil
}

func (s *Server) Run(stopChan, doneChan chan struct{}) error {
	defer close(doneChan)

	// Return if --run=false (boot blip/server but don't run)
	if !s.cmdline.Options.Run {
		return nil
	}
	event.Send(event.SERVER_RUN)

	stopReason := "unknown"
	defer func() {
		event.Errorf(event.SERVER_STOPPED, stopReason)
	}()

	// Start all monitors. Then if config.monitor-load.freq is specified, start
	// periodical monitor reloading.
	status.Blip("server", "loading monitors")
	s.monitorLoader.StartMonitors()
	if s.cfg.MonitorLoader.Freq != "" {

		for {
			go func() {
				defer func() { // catch panic in API
					if r := recover(); r != nil {
						b := make([]byte, 4096)
						n := runtime.Stack(b, false)
						err := fmt.Errorf("PANIC: monitor loader: %s\n%s", r, string(b[0:n]))
						event.Errorf(event.MONITOR_LOADER_PANIC, err.Error())
					}
				}()
				doneChan := make(chan struct{}) // ignored: goroutine dies with Server
				s.monitorLoader.Reload(stopChan, doneChan)
			}()
			time.Sleep(1 * time.Second) // between panic
		}
	}

	// Run API, restart on panic
	if !s.cfg.API.Disable {
		for {
			go func() {
				defer func() { // catch panic in API
					if r := recover(); r != nil {
						b := make([]byte, 4096)
						n := runtime.Stack(b, false)
						err := fmt.Errorf("PANIC: server API: %s\n%s", r, string(b[0:n]))
						event.Errorf(event.SERVER_API_PANIC, err.Error())
					}
				}()
				s.api.Run()
			}()
			time.Sleep(1 * time.Second) // between panic
		}
	}

	// Run until caller closes stopChan or blip process catches a signal
	status.Blip("server", "running since %s", time.Now())
	signalChan := make(chan os.Signal)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	select {
	case <-stopChan:
		stopReason = "server stopped"
	case <-signalChan:
		stopReason = "caught signal"
	}
	return nil
}
