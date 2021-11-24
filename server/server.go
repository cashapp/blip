// Package server provides the core runtime for Blip.
package server

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/square/blip"
	"github.com/square/blip/aws"
	"github.com/square/blip/dbconn"
	"github.com/square/blip/event"
	"github.com/square/blip/monitor"
	"github.com/square/blip/plan"
	"github.com/square/blip/status"
)

// ControlChans is a convenience function to return arguments for Run.
func ControlChans() (stopChan, doneChan chan struct{}) {
	return make(chan struct{}), make(chan struct{})
}

func Defaults() (blip.Env, blip.Plugins, blip.Factories) {
	// Plugins are optional, but factories are required
	awsConfig := aws.NewConfigFactory()
	dbMaker := dbconn.NewConnFactory(awsConfig, nil)
	factories := blip.Factories{
		AWSConfig:  awsConfig,
		DbConn:     dbMaker,
		HTTPClient: httpClientFactory{},
	}
	env := blip.Env{
		Args: os.Args,
		Env:  os.Environ(),
	}
	return env, blip.Plugins{}, factories
}

type httpClientFactory struct{}

func (f httpClientFactory) Make(cfg blip.ConfigHTTP, usedFor string) (*http.Client, error) {
	client := &http.Client{}
	if cfg.Proxy != "" {
		proxyFunc := func(req *http.Request) (url *url.URL, err error) {
			return url.Parse(cfg.Proxy)
		}
		client.Transport = &http.Transport{Proxy: proxyFunc}
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
	stopChan      chan struct{}
	doneChan      chan struct{}
}

// Boot boots Blip. That means it loads, validates, and creates everything,
// but it doesn't start (run) anything--that happens when Run is called.
// If Boot returns nil, then Blip is ready to run; else, any error is fatal:
// sometimes is too wrong for Blip to run (for example, an invalid config).
//
// Boot must be called once before Run.
func (s *Server) Boot(env blip.Env, plugin blip.Plugins, factory blip.Factories) error {
	event.Sendf(event.BOOT, "blip %s", blip.VERSION) // very first event
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
		cfg, err = blip.LoadConfig(s.cmdline.Options.Config, cfg)
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

	cfg.InterpolateEnvVars()
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
	s.planLoader = plan.NewLoader(plugin.LoadLevelPlans)

	if err := s.planLoader.LoadShared(s.cfg.Plans, factory.DbConn); err != nil {
		event.Sendf(event.BOOT_PLANS_ERROR, err.Error())
		return err
	}
	event.Send(event.BOOT_PLANS_LOADED)

	if s.cmdline.Options.PrintPlans {
		s.planLoader.Print()
	}

	// ----------------------------------------------------------------------
	// Database monitors

	// Create, but don't start, database monitors. They're started later in Run.
	s.monitorLoader = monitor.NewLoader(
		s.cfg,
		plugin.LoadMonitors,
		factory.DbConn,
		s.planLoader,
	)
	if err := s.monitorLoader.Load(context.Background()); err != nil {
		event.Sendf(event.BOOT_MONITORS_ERROR, err.Error())
		return err
	}

	if s.cmdline.Options.PrintMonitors {
		fmt.Println(s.monitorLoader.Print())
	}

	// ----------------------------------------------------------------------
	// API
	s.api = NewAPI(cfg.API, s.monitorLoader)

	return nil
}

func (s *Server) Run(stopChan, doneChan chan struct{}) error {
	if !s.cmdline.Options.Run {
		return nil
	}

	status.Blip("server", "running")

	defer close(doneChan)

	s.stopChan = stopChan
	s.doneChan = doneChan

	event.Send(event.SERVER_RUN)

	stopReason := "unknown"
	defer func() {
		event.Sendf(event.SERVER_RUN_STOP, stopReason)
	}()

	go s.monitorLoader.Run()

	go s.api.Run()

	// Run until stopped by Shutdown or signal
	event.Sendf(event.SERVER_RUN_WAIT, s.cfg.API.Bind)
	signalChan := make(chan os.Signal)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	select {
	case <-stopChan:
		stopReason = "Server.Shutdown called"
	case <-signalChan:
		stopReason = "caught signal"
	}
	return nil
}

func (s *Server) Shutdown() {
	select {
	case <-s.stopChan:
		// Already stopped
	default:
		status.Blip("server", "shutting down")
		close(s.stopChan)
		select {
		case <-s.doneChan:
		case <-time.After(3 * time.Second):
		}
	}
}
