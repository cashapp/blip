// Package server provides the core runtime for Blip.
package server

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/square/blip"
	"github.com/square/blip/collect"
	"github.com/square/blip/event"
	"github.com/square/blip/sink"
)

// ControlChans is a convenience function to return arguments for Run.
func ControlChans() (stopChan, doneChan chan struct{}) {
	return make(chan struct{}), make(chan struct{})
}

// Server is the core runtime for one instance of Blip. It's responsible for
// starting and running everything: Boot and Run, respectively. As long as
// the server is running, Blip is running.
//
// See bin/blip/main.go for the simplest use case.
type Server struct {
	cfg           blip.Config
	cmdline       CommandLine
	monitorLoader *MonitorLoader
	stopChan      chan struct{}
	doneChan      chan struct{}
}

// Boot boots Blip. That means it loads, validates, and creates everything,
// but it doesn't start (run) anything--that happens when Run is called.
// If Boot returns nil, then Blip is ready to run; else, any error is fatal:
// sometimes is too wrong for Blip to run (for example, an invalid config).
//
// Boot must be called once before Run.
func (s *Server) Boot(plugin Plugins, factory Factories) error {
	// ----------------------------------------------------------------------
	// Parse commad line options

	var err error
	s.cmdline, err = ParseCommandLine()
	if err != nil {
		return err
	}
	if s.cmdline.Options.Version {
		fmt.Println("blip", blip.VERSION)
		os.Exit(0)
	}
	if s.cmdline.Options.Help {
		printHelp()
		os.Exit(0)
	}

	// Set debug and strict from env vars. Do this very first because all code
	// uses blip.Debug() and blip.Strict (boolean).
	//
	// STRICT.md documents the effects of strict mode.
	if s.cmdline.Options.Debug {
		blip.Debugging = true
	}
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

	// ----------------------------------------------------------------------
	// Event sinks

	// Init event sink. Do this second-ish because all code sends events.
	// As with all plugins, the plugin takes priority and is used exclusively
	// if set by the user. If this plugin isn't set, the default even sink is
	// event.ToSTDOUT, which simply prints events to STDOUT.
	if plugin.InitEventSink != nil {
		blip.Debug("call plugin.InitEventSink")
		event.SetSink(plugin.InitEventSink())
	}
	event.Sendf(event.BOOT, "blip %s", blip.VERSION) // very first event

	// ----------------------------------------------------------------------
	// Load config
	event.Send(event.BOOT_CONFIG_LOADING)

	// Always start with a default config, else we'll lack some basic config
	// like the API addr:port to listen on. If strict, it's a zero config except
	// for the absolute most minimal/must-have config values. Else, the default
	// (not strict) config is a more realistic set of defaults.
	cfg := blip.DefaultConfig(blip.Strict)

	// User-provided LoadConfig plugin takes priority if set; else, use default
	// (built-in) LoadConfig func.
	if plugin.LoadConfig != nil {
		blip.Debug("call plugin.LoadConfig")
		cfg, err = plugin.LoadConfig(cfg)
	} else {
		cfg, err = LoadConfig(s.cmdline.Options.Config, cfg)
	}
	if err != nil {
		event.Sendf(event.BOOT_CONFIG_ERROR, err.Error())
		return err
	}

	// Extensively validate the config. Once done, the config is immutable,
	// except for plans and monitors which might come from dynamic sources,
	// like tables.
	if err := cfg.Validate(); err != nil {
		event.Sendf(event.BOOT_CONFIG_INVALID, err.Error())
		return err
	}

	s.cfg = cfg // final immutable config
	event.Send(event.BOOT_CONFIG_LOADED)

	if s.cmdline.Options.PrintConfig {
		printYAML(s.cfg)
	}

	// ----------------------------------------------------------------------
	// Load level plans

	// Get the built-in level plan loader singleton. It's used in two places:
	// here for initial plan loading, and level.Collector (LPC) to fetch the
	// plan and set the Monitor to use it.
	planLoader := collect.DefaultPlanLoader()

	// User-provided LoadLevelPlans plugin takes priority if set; else, use
	// the built-in default plan loader.
	if plugin.LoadLevelPlans != nil {
		blip.Debug("call plugin.LoadLevelPlans")
		plans, err := plugin.LoadLevelPlans(cfg)
		if err != nil {
			return err
		}
		err = planLoader.SetPlans(plans)
	} else {
		err = planLoader.LoadPlans(cfg, factory.MakeDbConn)
	}
	if err != nil {
		event.Sendf(event.BOOT_PLANS_ERROR, err.Error())
		return err
	}
	event.Send(event.BOOT_PLANS_LOADED)

	if s.cmdline.Options.PrintPlans {
		planLoader.Print()
	}

	// ----------------------------------------------------------------------
	// Sinks

	sinks := []sink.Sink{}
	if plugin.LoadMetricSinks != nil {
		blip.Debug("call plugin.LoadMetricSinks")
		sinks, err = plugin.LoadMetricSinks(cfg)
	} else {
		for sinkName, opts := range cfg.Sinks {
			sink, err := factory.MakeMetricSink.Make(sinkName, opts)
			if err != nil {
				// @todo
			}
			sinks = append(sinks, sink)
		}
		if len(sinks) == 0 && !blip.Strict {
			sink, _ := factory.MakeMetricSink.Make("log", nil)
			sinks = append(sinks, sink)
			blip.Debug("using default log sink")
		}
	}

	if plugin.TransformMetrics != nil {
		allSinks := sinks
		tf := sink.TransformMetrics{
			Plugin: plugin.TransformMetrics,
			Sinks:  allSinks,
		}
		sinks = []sink.Sink{tf}
		blip.Debug("using plugin.TransformMetrics")
	}

	// Make deferred dbmon factory
	if factory.MakeDbMon == nil {
		factory.MakeDbMon = &dbmonFactory{
			mcMaker:    factory.MakeMetricsCollector,
			dbMaker:    factory.MakeDbConn,
			planLoader: collect.DefaultPlanLoader(),
			sinks:      sinks,
		}
	}

	// ----------------------------------------------------------------------
	// Database monitors (dbmon)

	// Create, but don't start, database monitors. They're started later in Run.
	s.monitorLoader = NewMonitorLoader(cfg, plugin, factory)
	if _, err := s.monitorLoader.Load(context.Background()); err != nil {
		event.Sendf(event.BOOT_MONITORS_ERROR, err.Error())
		return err
	}

	if !s.cmdline.Options.BootCheck && (s.cmdline.Options.PrintConfig || s.cmdline.Options.PrintPlans) {
		os.Exit(0)
	}

	// ----------------------------------------------------------------------
	// API

	// Exit if boot check, else return and caller should call Run
	if s.cmdline.Options.BootCheck {
		os.Exit(0)
	}
	return nil
}

func (s *Server) Run(stopChan, doneChan chan struct{}) error {
	defer close(doneChan)

	s.stopChan = stopChan
	s.doneChan = doneChan

	event.Send(event.SERVER_RUN)
	defer event.Send(event.SERVER_RUN_STOP)

	monitors := s.monitorLoader.Monitors()

	// Space out dbmon so their clocks don't tick at the same time.
	// We don't want, for example, 25 dbmon simultaneously waking up,
	// connecting to MySQL, processing metrics. That'll make Blip
	// CPU/net usage unnecessarily spiky.
	var space time.Duration
	if len(monitors) < 25 {
		space = 20 * time.Millisecond
	} else {
		space = 10 * time.Millisecond
	}

	// Start database monitors (dbmon), which starts metrics collection
	for _, dbmon := range monitors {
		time.Sleep(space)
		if err := dbmon.Start(); err != nil {
			return err // @todo
		}
	}

	event.Send(event.SERVER_RUN_WAIT)

	signalChan := make(chan os.Signal)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	select {
	case <-stopChan:
	case <-signalChan:
	}

	return nil
}

func (s *Server) Shutdown() {
	select {
	case <-s.stopChan:
		// Already stopped
	default:
		close(s.stopChan)
		select {
		case <-s.doneChan:
		case <-time.After(3 * time.Second):
		}
	}
}
