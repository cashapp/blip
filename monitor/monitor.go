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
	"log"
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

// Monitor monitors one MySQL instance. A monitor is completely self-contained;
// monitors share nothing. Therefore, each monitor is completely independent, too.
//
// The Monitor type is a server that boots and runs the core components that do
// the real work: Engine, LevelCollector (LPC), optional LevelAdjuster (LPA),
// and optional Heartbeat (HB). You can view it like:
//
//   Blip -> Monitor 1[ Engine + LPA + LPC + HB ] -> MySQL 1
//        ...
//        -> Monitor N[ Engine + LPA + LPC + HB ] -> MySQL N
//
// Blip only calls/uses the Monitor; nothing outside the Monitor can access those
// core components.
type Monitor struct {
	// Required to create; created in Loader.makeMonitor()
	monitorId  string
	cfg        blip.ConfigMonitor
	dbMaker    blip.DbFactory
	planLoader *plan.Loader
	sinks      []blip.Sink

	// Core components
	runMux  *sync.RWMutex
	db      *sql.DB
	dsn     string
	engine  *Engine
	promAPI *prom.API
	lpc     LevelCollector
	lpa     LevelAdjuster
	hbw     *heartbeat.Writer

	// Control chans and sync
	stopMonitorChan chan struct{} // Stop(): stop the monitor
	stopRunChan     chan struct{} // stop goroutines run by monitor
	wg              sync.WaitGroup

	errMux *sync.Mutex
	err    error

	event event.MonitorReceiver
	retry *backoff.ExponentialBackOff
}

// MonitorArgs are required arguments to NewMonitor.
type MonitorArgs struct {
	Config     blip.ConfigMonitor
	DbMaker    blip.DbFactory
	PlanLoader *plan.Loader
	Sinks      []blip.Sink
}

// NewMonitor creates a new Monitor with the given arguments. The caller must
// call Boot then, if that does not return an error, Run to start monitoring
// the MySQL instance.
func NewMonitor(args MonitorArgs) *Monitor {
	retry := backoff.NewExponentialBackOff()
	retry.MaxElapsedTime = 0
	retry.MaxInterval = 20 * time.Second
	return &Monitor{
		monitorId:  args.Config.MonitorId,
		cfg:        args.Config,
		dbMaker:    args.DbMaker,
		planLoader: args.PlanLoader,
		sinks:      args.Sinks,
		// --
		stopMonitorChan: make(chan struct{}),
		stopRunChan:     make(chan struct{}),
		errMux:          &sync.Mutex{},
		runMux:          &sync.RWMutex{},
		wg:              sync.WaitGroup{},
		event:           event.MonitorReceiver{MonitorId: args.Config.MonitorId},
		retry:           retry,
	}
}

// MonitorId returns the monitor ID. This method implements *Monitor.
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
		status.Engine = m.engine.Status()
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

func (m *Monitor) setErr(err error, isPanic bool) {
	if err != nil {
		if isPanic {
			log.Println(err) // extra logging on panic
			m.event.Errorf(event.MONITOR_PANIC, err.Error())
		} else {
			m.event.Errorf(event.MONITOR_ERROR, err.Error())
		}
		status.Monitor(m.monitorId, "monitor", "error: %s", err)
	}
	m.errMux.Lock()
	m.err = err
	m.errMux.Unlock()
}

// Run runs the monitor. This function could return after starting the monitor
// components, but it blocks until Stop is called for its defer/recover, i.e.
// to catch panics. The only caller is Loader.Run: the monitor loader is the only
// component that runs monitors. It runs this function as a goroutine, which is
// why the defer/recover works here.
func (m *Monitor) Run() {
	for {
		// New stopRunChan for every iteration; it can only be used/closed once
		m.runMux.Lock()
		m.stopRunChan = make(chan struct{})
		m.runMux.Unlock()

		// Run monitor startup sequence. If successful, the monitor is running
		// but that doesn't mean metrics are collecting because collectors can
		// fail (and retry) for different reasons.
		err := m.start()
		m.setErr(err, false)
		if err != nil {
			time.Sleep(m.retry.NextBackOff())
			continue
		}

		// Monitor is running. Wait for either Stop (which closes m.stopMonitorChan)
		// or one of the monitor goroutines to return/panic (which closes m.stopRunChan).
		// On Stop, return immediately: user is stopping the monitor completely.
		// On m.stopRunChan close (via stop func), we restart almost immediately because
		// Blip never stops trying to send metrics.
		m.retry.Reset()
		status.Monitor(m.monitorId, "monitor", "running since %s", time.Now())
		select {
		case <-m.stopMonitorChan:
			blip.Debug("%s: monitor stopped", m.monitorId)
			status.Monitor(m.monitorId, "monitor", "stopped at %s", time.Now())
			return
		case <-m.stopRunChan:
			blip.Debug("%s: stopRunChan closed; restarting", m.monitorId)
			time.Sleep(1 * time.Second) // between monitor restarts
		}
	}
}

// start starts the four monitor goroutines (all are optional depending on config):
// heartbeat writer, exporter API (Prometheus emulation), LPA, and LPC.
// The monitoring is running once these have started. If any one goroutine fails
// (panic), then stopRunhChan is closed (because all goroutines call stop()),
// which is detected by Run, which restarts the monitor.
func (m *Monitor) start() error {
	blip.Debug("%s: start called", m.monitorId)
	defer blip.Debug("%s: start return", m.monitorId)

	// Catch panic in this func, pretty much just the start-wait loops because
	// the monitor goroutines run in separate goroutines, so they have their own
	// defer.
	defer func() {
		if r := recover(); r != nil {
			b := make([]byte, 4096)
			n := runtime.Stack(b, false)
			errMsg := fmt.Errorf("PANIC: %s: %s\n%s", m.monitorId, r, string(b[0:n]))
			m.setErr(errMsg, true)

			m.stop(true, "start") // stop monitor goroutines
		}
	}()

	// //////////////////////////////////////////////////////////////////////
	// Start-wait loops
	// //////////////////////////////////////////////////////////////////////

	// ----------------------------------------------------------------------
	// Make DSN and *sql.DB
	for {
		status.Monitor(m.monitorId, "monitor", "making DB/DSN (not connecting)")
		db, dsn, err := m.dbMaker.Make(m.cfg)
		m.setErr(err, false)
		if err == nil { // success
			m.runMux.Lock()
			m.db = db
			m.dsn = dsn
			m.runMux.Unlock()
			break
		}
		select {
		case <-m.stopMonitorChan:
		default:
			return nil // monitor stopped
		}
		status.Monitor(m.monitorId, "monitor", "error making DB/DSN, sleep and retry: %s", err)
		time.Sleep(m.retry.NextBackOff())
	}

	// ----------------------------------------------------------------------
	// Load monitor plans, if any
	for {
		status.Monitor(m.monitorId, "monitor", "loading plans")
		err := m.planLoader.LoadMonitor(m.cfg, m.dbMaker)
		m.setErr(err, false)
		if err == nil { // success
			break
		}
		select {
		case <-m.stopMonitorChan:
		default:
			return nil // monitor stopped
		}
		status.Monitor(m.monitorId, "monitor", "error loading plans, sleep and retry: %s", err)
		time.Sleep(m.retry.NextBackOff())
	}

	// //////////////////////////////////////////////////////////////////////
	// Monitor goroutines
	// //////////////////////////////////////////////////////////////////////
	m.runMux.Lock()
	defer m.runMux.Unlock()

	// ----------------------------------------------------------------------
	// Heartbeat

	// Run optional heartbeat monitor to monitor replication lag. When enabled,
	// the heartbeat (hb) writes a high-resolution timestamp to a row in a table
	// at the configured frequence: cfg.monitors.M.heartbeat.freq.
	if m.cfg.Heartbeat.Freq != "" {
		status.Monitor(m.monitorId, "monitor", "starting heartbeat")
		m.hbw = heartbeat.NewWriter(m.monitorId, m.db, m.cfg.Heartbeat)
		m.wg.Add(1)
		go func() {
			defer m.stop(true, "heartbeat.Writer") // stop monitor goroutines
			defer m.wg.Done()                      // notify stop()
			defer func() {                         // catch panic in heartbeat.Writer
				if r := recover(); r != nil {
					b := make([]byte, 4096)
					n := runtime.Stack(b, false)
					errMsg := fmt.Errorf("PANIC: %s: %s\n%s", m.monitorId, r, string(b[0:n]))
					m.setErr(errMsg, true)
				}
			}()
			doneChan := make(chan struct{}) // Monitor uses wg
			m.hbw.Write(m.stopRunChan, doneChan)
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
			blip.Debug(err.Error())
			status.Monitor(m.monitorId, "exporter", "not running: error loading plans: %s", err)
			return err
		}

		// Determine actual prom plan: either the default if it's user-provide
		// (i.e. not the default blip plan), or the provided plan. Then validate and
		// tweak based on config.exporter.flags.
		promPlan, err := ExporterPlan(m.cfg.Exporter, defaultPlan)
		if err != nil {
			blip.Debug(err.Error())
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
			defer m.stop(true, "prom.API") // stop monitor goroutines
			defer m.wg.Done()              // notify stop()
			defer func() {                 // catch panic in exporter API
				if r := recover(); r != nil {
					b := make([]byte, 4096)
					n := runtime.Stack(b, false)
					errMsg := fmt.Errorf("PANIC: %s: %s\n%s", m.monitorId, r, string(b[0:n]))
					m.setErr(errMsg, true)
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

	// LPC starts paused becuase there's no plan. The main loop in lpc.Run
	// does nothing until lpc.ChangePlan is called, which is done next either
	// indirectly via LPA or directly if LPA isn't enabled.
	status.Monitor(m.monitorId, "monitor", "starting LPC")
	m.engine = NewEngine(m.cfg, m.db)
	m.lpc = NewLevelCollector(LevelCollectorArgs{
		Config:     m.cfg,
		Engine:     m.engine,
		PlanLoader: m.planLoader,
		Sinks:      m.sinks,
	})

	m.wg.Add(1)
	go func() {
		defer m.stop(true, "LPC") // stop monitor goroutines
		defer m.wg.Done()         // notify stop()
		defer func() {            // catch panic in LPC
			if r := recover(); r != nil {
				b := make([]byte, 4096)
				n := runtime.Stack(b, false)
				errMsg := fmt.Errorf("PANIC: %s: %s\n%s", m.monitorId, r, string(b[0:n]))
				m.setErr(errMsg, true)
			}
		}()
		doneChan := make(chan struct{}) // Monitor uses wg
		m.lpc.Run(m.stopRunChan, doneChan)
	}()

	// ----------------------------------------------------------------------
	// Level plan adjuster (LPA)

	if m.cfg.Plans.Adjust.Enabled() {
		// Run option level plan adjuster (LPA). When enabled, the LPA checks
		// the state of MySQL. If the state changes, it calls lpc.ChangePlan
		// to change the plan as configured by config.monitors.M.plans.adjust.<state>.
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
			defer m.stop(true, "LPA") // stop monitor goroutines
			defer m.wg.Done()         // notify stop()
			defer func() {            // catch panic in LPA
				if r := recover(); r != nil {
					b := make([]byte, 4096)
					n := runtime.Stack(b, false)
					errMsg := fmt.Errorf("PANIC: %s: %s\n%s", m.monitorId, r, string(b[0:n]))
					m.setErr(errMsg, true)
				}
			}()
			doneChan := make(chan struct{})    // Monitor uses wg
			m.lpa.Run(m.stopRunChan, doneChan) // start LPC indirectly
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

// stop stops the monitor goroutines started in start. It does not stop the
// monitor; Stop does that. Stopping the monitor goroutines causes Run to
// restart them.
func (m *Monitor) stop(lock bool, caller string) {
	if lock {
		m.runMux.Lock()
		defer m.runMux.Unlock()
	}

	// Already stopped?
	select {
	case <-m.stopRunChan:
		blip.Debug("%s: stop called by %s (noop)", m.monitorId, caller)
		return // already stopped
	default:
		blip.Debug("%s: stop called by %s (first)", m.monitorId, caller)
		defer blip.Debug("%s: stop return for %s", m.monitorId, caller)
	}

	// Stop most of the monitor goroutines
	close(m.stopRunChan)

	// Stop exporter API; this one doesn't use stop/done control chans because
	// it's running an http.Server
	if m.promAPI != nil {
		m.promAPI.Stop()
	}

	// Wait for monitor gourtines to return
	status.Monitor(m.monitorId, "monitor", "stopping goroutines")
	m.wg.Wait()
}

// Restart restarts the monitor.
func (m *Monitor) Restart() error {
	m.runMux.Lock()
	defer m.runMux.Unlock()

	// Stop and wait for monitor goroutines
	m.stop(false, "Stop")

	// Everything should be stopped now, so close db connection
	if m.db != nil {
		m.db.Close()
	}

	return nil
}

// Stop stops the monitor.
func (m *Monitor) Stop() error {
	m.runMux.Lock()
	defer m.runMux.Unlock()

	// Stop Run loop (whole monitor)
	select {
	case <-m.stopMonitorChan:
		blip.Debug("%s: already stopped", m.monitorId)
		return nil
	default:
	}

	blip.Debug("%s: Stop call", m.monitorId)
	defer blip.Debug("%s: Stop return", m.monitorId)

	// Stop Run loop so it won't restart everything
	close(m.stopMonitorChan)

	// Stop and wait for monitor goroutines
	m.stop(false, "Stop")

	// Everything should be stopped now, so close db connection
	if m.db != nil {
		m.db.Close()
	}

	event.Sendf(event.MONITOR_STOPPED, m.monitorId)
	status.Monitor(m.monitorId, "monitor", "stopped at %s", time.Now())
	return nil
}
