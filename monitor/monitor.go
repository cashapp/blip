// Copyright 2022 Block, Inc.

// Package monitor provides core Blip components that, together, monitor one
// MySQL instance. Most monitoring logic happens in the package, but package
// metrics is closely related: this latter actually collect metrics, but it
// is driven by this package. Other Blip packages are mostly set up and support
// of monitors.
package monitor

import (
	"database/sql"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"

	"github.com/cashapp/blip"
	"github.com/cashapp/blip/event"
	"github.com/cashapp/blip/ha"
	"github.com/cashapp/blip/heartbeat"
	"github.com/cashapp/blip/plan"
	"github.com/cashapp/blip/prom"
	"github.com/cashapp/blip/proto"
	"github.com/cashapp/blip/status"
)

// Monitor monitors one MySQL instance. The monitor is a high-level component
// that runs (and keeps running) four monitor subsystems:
//   - Level plan collector (LPC)
//   - Level plan adjuster (LPA)
//   - Blip heartbeat writer
//   - Exporter (Prometheus)
//
// Each subsystem is optional based on the config, but LPC runs by default
// because it contains the Engine component that does actual metrics collection.
// If any subsystem crashes (returns for any reason or panics), the monitor
// stops and restarts all subsystems. The monitor doesn't stop until Stop is
// called. Consequently, if a monitor is not configured correctly (for example,
// it can't connect to MySQL), it tries and reports every forever.
//
// Monitors are loaded, created, and initially started only by the MonitorLoader.
// A monitor can be stopped and started (again) via the server API.
//
// A monitor is uniquely identified by its monitor ID, which should be included
// in all output by the monitor and its subsystems. The monitor ID is set when
// loaded by the MonitoLoad, which calls blip.MonitorId to determine the value.
//
// A monitor is completely self-contained and independent. For example, each monitor
// has its own LPC, engine, and metric collectors.
type Monitor struct {
	// Required to create; created in Loader.makeMonitor()
	monitorId       string
	cfg             blip.ConfigMonitor
	dbMaker         blip.DbFactory
	planLoader      *plan.Loader
	sinks           []blip.Sink
	transformMetric func(metrics *blip.Metrics) error

	// Core components
	runMux  *sync.RWMutex
	db      *sql.DB
	dsn     string // redacted (no password)
	promAPI *prom.API
	lpc     LevelCollector
	lpa     LevelAdjuster
	hbw     *heartbeat.Writer

	// Control chans and sync
	runLoopChan chan struct{} // Stop(): stop the monitor
	runChan     chan struct{} // stop goroutines run by monitor
	wg          sync.WaitGroup

	errMux *sync.Mutex
	err    error

	event event.MonitorReceiver
	retry *backoff.ExponentialBackOff
}

// MonitorArgs are required arguments to NewMonitor.
type MonitorArgs struct {
	Config          blip.ConfigMonitor
	DbMaker         blip.DbFactory
	PlanLoader      *plan.Loader
	Sinks           []blip.Sink
	TransformMetric func(metrics *blip.Metrics) error
}

// NewMonitor creates a new Monitor with the given arguments. The caller must
// call Boot then, if that does not return an error, Run to start monitoring
// the MySQL instance.
func NewMonitor(args MonitorArgs) *Monitor {
	retry := backoff.NewExponentialBackOff()
	retry.MaxElapsedTime = 0
	retry.MaxInterval = 20 * time.Second
	return &Monitor{
		monitorId:       args.Config.MonitorId,
		cfg:             args.Config,
		dbMaker:         args.DbMaker,
		planLoader:      args.PlanLoader,
		sinks:           args.Sinks,
		transformMetric: args.TransformMetric,
		// --
		errMux: &sync.Mutex{},
		runMux: &sync.RWMutex{},
		wg:     sync.WaitGroup{},
		event:  event.MonitorReceiver{MonitorId: args.Config.MonitorId},
		retry:  retry,
	}
}

// MonitorId returns the monitor ID.
func (m *Monitor) MonitorId() string {
	return m.monitorId
}

// Config returns the monitor config.
func (m *Monitor) Config() blip.ConfigMonitor {
	return m.cfg
}

// Status returns the real-time monitor status. See proto.MonitorStatus for details.
func (m *Monitor) Status() proto.MonitorStatus {
	m.runMux.RLock()
	status := proto.MonitorStatus{
		MonitorId: m.monitorId,
	}
	if m.dsn != "" {
		status.DSN = m.dsn
	}
	if m.lpc != nil {
		status.Collector = m.lpc.Status()
	}
	if m.lpa != nil {
		lpaStatus := m.lpa.Status()
		status.Adjuster = &lpaStatus
	}
	m.runMux.RUnlock()

	m.errMux.Lock()
	if m.err != nil {
		status.Error = m.err.Error()
	}
	m.errMux.Unlock()

	return status
}

// Stop stops the monitor. It is idempotent and thread-safe.
func (m *Monitor) Stop() error {
	m.runMux.Lock()
	defer m.runMux.Unlock()

	blip.Debug("%s: Stop call", m.monitorId)
	defer blip.Debug("%s: Stop return", m.monitorId)

	// Stop runLoop() _first_, else it will restart run()
	select {
	case <-m.runLoopChan: // not running
		blip.Debug("%s: already stopped", m.monitorId)
		return nil
	default: // running
	}

	// Stop runLoop so it won't restart everything
	close(m.runLoopChan)

	// Stop and wait for monitor subsystems
	m.stop(false, "Stop")

	// Everything should be stopped now, so close db connection
	if m.db != nil {
		m.db.Close()
	}

	event.Sendf(event.MONITOR_STOPPED, m.monitorId)
	status.Monitor(m.monitorId, "monitor", "stopped at %s", time.Now())
	return nil
}

// Start starts the monitor. If it's already running, it returns an error.
// It can be called again after calling Stop.
func (m *Monitor) Start() error {
	m.runMux.Lock()
	defer m.runMux.Unlock()
	select {
	case <-m.runLoopChan:
		// not running
		blip.Debug("%s: start (again)", m.monitorId)
	default:
		if m.runLoopChan != nil { // running
			return fmt.Errorf("ready running")
		}
		// first start
		blip.Debug("%s: start (first)", m.monitorId)
	}
	m.runLoopChan = make(chan struct{})
	go m.runLoop()
	return nil
}

// runnLoop starts and keeps the monitor subsystems running by calling startup.
// If any subsystem crashes, it calls startup again. It doesn't stop until Stop
// is called.
//
// runLoop is called only by Start, which guards (serializes) it.
func (m *Monitor) runLoop() {
	defer blip.Debug("%s: runLoop return", m.monitorId)
	for {
		// New runChan for every iteration; it can only be used/closed once
		m.runMux.Lock()
		m.runChan = make(chan struct{})
		m.runMux.Unlock()

		// Run monitor startup sequence to start all (enabled) monitor subsystems.
		// If successful, the monitor is running but that does _not_ mean metrics
		// are collecting because collectors can fail for different reasons.
		err := m.startup()
		m.setErr(err, false)
		if err != nil {
			time.Sleep(m.retry.NextBackOff())
			continue
		}

		// Monitor is running. Wait for either Stop (which closes m.runLoopChan)
		// or one of the monitor subsystems to return/panic (which closes m.runChan).
		// On Stop, return immediately: user is stopping the monitor completely.
		// On m.runChan close (via stop func), we restart almost immediately because
		// Blip never stops trying to send metrics.
		m.retry.Reset()
		m.event.Sendf(event.MONITOR_STARTED, m.dsn)
		status.Monitor(m.monitorId, "monitor", "running since %s", time.Now())
		select {
		case <-m.runLoopChan: // Stop called
			return
		case <-m.runChan: // internal failure
			blip.Debug("%s: runChan closed; restarting", m.monitorId)
			time.Sleep(1 * time.Second) // between monitor restarts
		}
	}
}

// startup starts the monitor subsystems, which are optional depending on config:
// heartbeat writer, exporter API (Prometheus emulation), LPA, and LPC.
// The monitor is running once these have started. If any subsystem crashes
// (or returns for any reason), it calls stop() to stop the other subsystems,
// then runLoop() calls startup again to restart monitoring.
//
// startup is called only by runLoop, which guards (serializes) and monitors it.
func (m *Monitor) startup() error {
	blip.Debug("%s: startup call", m.monitorId)
	defer blip.Debug("%s: startup return", m.monitorId)

	// Catch panic in this func, pretty much just the DB-plan loop because
	// each monitor subsystems goroutine has its own defer/recover.
	defer func() {
		if r := recover(); r != nil {
			m.panic(r)
			m.stop(true, "startup") // stop monitor subsystems
		}
	}()

	// //////////////////////////////////////////////////////////////////////
	// DB-plan loop
	// //////////////////////////////////////////////////////////////////////

	// ----------------------------------------------------------------------
	// Make DSN and *sql.DB. This does NOT connect to MySQL.
	for {
		status.Monitor(m.monitorId, "monitor", "making DB/DSN (not connecting)")
		db, dsnRedacted, err := m.dbMaker.Make(m.cfg)
		m.setErr(err, false)
		if err == nil { // success
			m.runMux.Lock()
			m.db = db
			m.dsn = dsnRedacted
			m.runMux.Unlock()
			break
		}
		select {
		case <-m.runLoopChan:
			return nil // runLoop stopped (Stop called)
		default:
		}
		status.Monitor(m.monitorId, "monitor", "error making DB/DSN, sleep and retry: %s", err)
		time.Sleep(m.retry.NextBackOff())
	}

	// ----------------------------------------------------------------------
	// Load monitor plans, if any. This MIGHT connect to MySQL if the plan
	// is stored in a table.
	for {
		status.Monitor(m.monitorId, "monitor", "loading plans")
		err := m.planLoader.LoadMonitor(m.cfg, m.dbMaker)
		m.setErr(err, false)
		if err == nil { // success
			break
		}
		select {
		case <-m.runLoopChan:
		default:
			return nil // // runLoop stopped (Stop called)
		}
		status.Monitor(m.monitorId, "monitor", "error loading plans, sleep and retry: %s", err)
		time.Sleep(m.retry.NextBackOff())
	}

	// //////////////////////////////////////////////////////////////////////
	// Start monitor subsystems
	// //////////////////////////////////////////////////////////////////////

	m.runMux.Lock()
	defer m.runMux.Unlock()

	// ----------------------------------------------------------------------
	// Heartbeat

	// Run optional heartbeat write. When enabled (by setting heartbeat.freq),
	// Blip writes millisecond-precision timestamps to a table that the repl.lag
	// metric collector uses to report sub-second replication lag.
	if m.cfg.Heartbeat.Freq != "" {
		status.Monitor(m.monitorId, "monitor", "starting heartbeat")
		m.hbw = heartbeat.NewWriter(m.monitorId, m.db, m.cfg.Heartbeat)
		m.wg.Add(1)
		go func() {
			defer m.stop(true, "heartbeat.Writer") // stop monitor subsystems
			defer m.wg.Done()                      // notify stop()
			defer func() {                         // catch panic in heartbeat.Writer
				if r := recover(); r != nil {
					m.panic(r)
				}
			}()
			doneChan := make(chan struct{}) // Monitor uses wg
			m.hbw.Write(m.runChan, doneChan)
		}()
	}

	// ----------------------------------------------------------------------
	// Exporter API (Prometheus emulation)

	if m.cfg.Exporter.Mode != "" {
		status.Monitor(m.monitorId, "monitor", "starting exporter")

		// Get default plan for monitor. It's possible user provided a prom-compatible.
		// If not, this returns the Blip default plan, but ExporterPlan will discard
		// that and choose the correct internal plan based on any exporter flags.
		defaultPlan, err := m.planLoader.Plan(m.monitorId, "", nil)
		if err != nil {
			blip.Debug("%s: %s", m.monitorId, err.Error())
			status.Monitor(m.monitorId, "exporter", "not running: error loading plans: %s", err)
			return err
		}

		// Determine actual prom plan: either the default if it's user-provide
		// (i.e. not the default blip plan), or the provided plan. Then validate and
		// tweak based on config.exporter.flags.
		promPlan, err := ExporterPlan(m.cfg.Exporter, defaultPlan)
		if err != nil {
			blip.Debug("%s: %s", m.monitorId, err.Error())
			status.Monitor(m.monitorId, "exporter", "not running: invalid plan: %s", err)
			return err
		}
		blip.Debug("%s: exporter plan: %s (%s)", m.monitorId, promPlan.Name, promPlan.Source)

		// Run API to emulate an exporter, responding to GET /metrics
		m.promAPI = prom.NewAPI(
			m.cfg.Exporter,
			m.monitorId,
			NewExporter(m.cfg.Exporter, promPlan, NewEngine(m.cfg, m.db)),
		)

		m.wg.Add(1)
		go func() {
			defer status.RemoveComponent(m.monitorId, "exporter")
			defer m.stop(true, "prom.API") // stop monitor subsystems
			defer m.wg.Done()              // notify stop()
			defer func() {                 // catch panic in exporter API
				if r := recover(); r != nil {
					m.panic(r)
				}
			}()
			err := m.promAPI.Run()
			if err == nil { // shutdown
				blip.Debug("%s: prom api stopped", m.monitorId)
				return
			}
			blip.Debug("%s: prom api error: %s", m.monitorId, err.Error())
			status.Monitor(m.monitorId, "exporter", "API error (restart in 1s): %s", err)
		}()

		if m.cfg.Exporter.Mode == blip.EXPORTER_MODE_LEGACY {
			blip.Debug("%s: legacy mode", m.monitorId)
			status.Monitor(m.monitorId, "monitor", "running in exporter legacy mode")
			return nil
		}
	}

	// ----------------------------------------------------------------------
	// Level plan collector (LPC)

	// Start LPC before LPA because the latter calls the former on state change.
	// The LPC starts paused (engine not running) until a plan is set by calling
	// lpc.ChangePlan. If the LPA is enabled, it will do this; if it's not enabled,
	// we'll do it as the last startup step.
	status.Monitor(m.monitorId, "monitor", "starting LPC")
	m.lpc = NewLevelCollector(LevelCollectorArgs{
		Config:           m.cfg,
		Engine:           NewEngine(m.cfg, m.db),
		PlanLoader:       m.planLoader,
		Sinks:            m.sinks,
		TransformMetrics: m.transformMetric,
	})

	m.wg.Add(1)
	go func() {
		defer m.stop(true, "LPC") // stop monitor subsystems
		defer m.wg.Done()         // notify stop()
		defer func() {            // catch panic in LPC
			if r := recover(); r != nil {
				m.panic(r)
			}
		}()
		doneChan := make(chan struct{}) // Monitor uses wg
		m.lpc.Run(m.runChan, doneChan)
	}()

	// ----------------------------------------------------------------------
	// Level plan adjuster (LPA)

	if m.cfg.Plans.Adjust.Enabled() {
		// Run option level plan adjuster (LPA). When enabled, the LPA checks
		// the state of MySQL. If the state changes, it calls lpc.ChangePlan
		// to change the plan as configured by config.monitors.plans.adjust.<state>.
		status.Monitor(m.monitorId, "monitor", "starting LPA")
		m.lpa = NewLevelAdjuster(LevelAdjusterArgs{
			MonitorId: m.monitorId,
			Config:    m.cfg.Plans.Adjust,
			DB:        m.db,
			LPC:       m.lpc,
			HA:        ha.Disabled,
		})

		m.wg.Add(1)
		go func() {
			defer m.stop(true, "LPA") // stop monitor subsystems
			defer m.wg.Done()         // notify stop()
			defer func() {            // catch panic in LPA
				if r := recover(); r != nil {
					m.panic(r)
				}
			}()
			doneChan := make(chan struct{}) // Monitor uses wg
			m.lpa.Run(m.runChan, doneChan)  // start LPC indirectly
		}()
	} else {
		// When the LPA is not enabled, we must init the state and plan,
		// which are ACTIVE and first (""), respectively. Since LPA is
		// optional, this is the normal case: startup presuming MySQL is
		// active and use the monitor's first plan.
		//
		// Do need retry or error handling because ChangePlan tries forever,
		// or until called again.
		status.Monitor(m.monitorId, "monitor", "setting state active")
		m.lpc.ChangePlan(blip.STATE_ACTIVE, "") // start LPC directly
	}

	return nil
}

// stop stops the monitor subsystems started in startup. It does not stop the
// monitor; Stop does that. Stopping only the monitor subsystems causes runLoop
// to restart them.
func (m *Monitor) stop(lock bool, caller string) {
	if lock {
		m.runMux.Lock()
		defer m.runMux.Unlock()
	}

	// Already stopped?
	select {
	case <-m.runChan:
		blip.Debug("%s: stop called by %s (noop)", m.monitorId, caller)
		return // already stopped
	default:
		blip.Debug("%s: stop called by %s (first)", m.monitorId, caller)
		defer blip.Debug("%s: stop return for %s", m.monitorId, caller)
	}

	// Stop the monitor subsystem goroutines (except exporter/Prom API)
	close(m.runChan)

	// Stop exporter API; this one doesn't use stop/done control chans because
	// it's running an http.Server
	if m.promAPI != nil {
		m.promAPI.Stop()
	}

	// Wait for monitor subsystem goroutines to return
	status.Monitor(m.monitorId, "monitor", "stopping goroutines")
	m.wg.Wait()
}

func (m *Monitor) setErr(err error, isPanic bool) {
	if err != nil {
		m.event.Errorf(event.MONITOR_ERROR, err.Error())
		status.Monitor(m.monitorId, "monitor", "error: %s", err)
	}
	m.errMux.Lock()
	m.err = err
	m.errMux.Unlock()
}

func (m *Monitor) panic(r interface{}) {
	b := make([]byte, 4096)
	n := runtime.Stack(b, false)
	errMsg := fmt.Errorf("PANIC: %s: %s\n%s", m.monitorId, r, string(b[0:n]))
	m.setErr(errMsg, true)
}
