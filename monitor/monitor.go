// Package monitor provides core Blip components that, together, monitor one
// MySQL instance. Most monitoring logic happens in the package, but package
// metrics is closely related: this latter actually collect metrics, but it
// is driven by this package. Other Blip packages are mostly set up and support
// of monitors.
package monitor

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"

	"github.com/square/blip"
	"github.com/square/blip/event"
	"github.com/square/blip/ha"
	"github.com/square/blip/heartbeat"
	"github.com/square/blip/plan"
	"github.com/square/blip/prom"
	"github.com/square/blip/proto"
	"github.com/square/blip/status"
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
	config     blip.ConfigMonitor
	dbMaker    blip.DbFactory
	planLoader *plan.Loader
	sinks      []blip.Sink

	// Core components
	db      *sql.DB
	dsn     string
	engine  *Engine
	promAPI *prom.API
	lpc     LevelCollector
	lpa     LevelAdjuster
	hbw     *heartbeat.BlipWriter

	ctx context.Context

	// Control chans and sync
	stopChan     chan struct{}
	doneChanLPA  chan struct{}
	doneChanLPC  chan struct{}
	doneChanHBW  chan struct{}
	doneChanProm chan struct{}
	stopped      bool

	errMux *sync.Mutex
	err    error

	runMux *sync.RWMutex
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
	return &Monitor{
		monitorId:  args.Config.MonitorId,
		config:     args.Config,
		dbMaker:    args.DbMaker,
		planLoader: args.PlanLoader,
		sinks:      args.Sinks,
		// --
		stopChan:    make(chan struct{}),
		errMux:      &sync.Mutex{},
		runMux:      &sync.RWMutex{},
		doneChanLPC: make(chan struct{}),
	}
}

// MonitorId returns the monitor ID. This method implements *Monitor.
func (m *Monitor) MonitorId() string {
	return m.monitorId
}

// DB returns the low-level database connection. This method implements *Monitor.
//func (m *Monitor) DB() *sql.DB {
//	return m.db
//}

// Config returns the monitor config. This method implements *Monitor.
func (m *Monitor) Config() blip.ConfigMonitor {
	return m.config
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

func (m *Monitor) setErr(err error) {
	m.errMux.Lock()
	m.err = err
	m.errMux.Unlock()
}

// Boot sets up the monitor. If it does not return an error, then the caller
// should call Run to start monitoring. If it returns an error, do not call Run.
func (m *Monitor) Run() {
	blip.Debug("%s: Run called", m.monitorId)
	defer blip.Debug("%s: Run return", m.monitorId)

	defer m.Stop()

	defer func() {
		if err := recover(); err != nil {
			b := make([]byte, 4096)
			n := runtime.Stack(b, false)
			log.Printf("PANIC: %s\n%s", err, string(b[0:n]))
		}
	}()

	retry := backoff.NewExponentialBackOff()
	retry.MaxElapsedTime = 0

	for {
		status.Monitor(m.monitorId, "monitor", "making DB/DSN (not connecting)")
		db, dsn, err := m.dbMaker.Make(m.config)
		m.setErr(err)
		if err == nil { // success
			m.runMux.Lock()
			m.db = db
			m.dsn = dsn
			m.runMux.Unlock()
			break
		}
		if m.stop() {
			return
		}
		status.Monitor(m.monitorId, "monitor", "error making DB/DSN, sleep and retry: %s", err)
		time.Sleep(retry.NextBackOff())
	}

	m.runMux.Lock() // -- BOOT --------------------------------------------

	// ----------------------------------------------------------------------
	// Prometheus emulation

	if m.config.Exporter.Mode != "" {
		m.promAPI = prom.NewAPI(
			m.config.Exporter,
			m.monitorId,
			NewExporter(m.config.Exporter, NewEngine(m.monitorId, m.db)),
		)
		m.doneChanProm = make(chan struct{})
		go func() {
			defer close(m.doneChanProm)
			if err := recover(); err != nil {
				b := make([]byte, 4096)
				n := runtime.Stack(b, false)
				log.Printf("PANIC: %s\n%s", err, string(b[0:n]))
			}
			for {
				err := m.promAPI.Run()
				if err == nil { // shutdown
					blip.Debug("%s: prom api stopped", m.monitorId)
					return
				}
				blip.Debug("%s: prom api error: %s", m.monitorId, err.Error())
				time.Sleep(1 * time.Second)
				continue
			}
		}()
		if m.config.Exporter.Mode == blip.EXPORTER_MODE_LEGACY {
			blip.Debug("legacy mode")
			<-m.stopChan
			return
		}
	}

	// ----------------------------------------------------------------------
	// Level plans

	retry.Reset()
	for {
		status.Monitor(m.monitorId, "monitor", "loading plans")
		err := m.planLoader.LoadMonitor(m.config, m.dbMaker)
		m.setErr(err)
		if err == nil { // success
			break
		}
		if m.stop() {
			return
		}
		status.Monitor(m.monitorId, "monitor", "error loading plans, sleep and retry: %s", err)
		time.Sleep(retry.NextBackOff())
	}

	// ----------------------------------------------------------------------
	// Heartbeat

	// Run optional heartbeat monitor to monitor replication lag. When enabled,
	// the heartbeat (hb) writes a high-resolution timestamp to a row in a table
	// at the configured frequence: config.monitors.M.heartbeat.freq.
	if m.config.Heartbeat.Freq != "" {
		status.Monitor(m.monitorId, "monitor", "starting heartbeat")
		m.hbw = heartbeat.NewBlipWriter(m.monitorId, m.db, m.config.Heartbeat)
		m.doneChanHBW = make(chan struct{})
		go m.hbw.Write(m.stopChan, m.doneChanHBW)
	}

	// ----------------------------------------------------------------------
	// Level plan collector (LPC)

	m.engine = NewEngine(m.monitorId, m.db)
	m.lpc = NewLevelCollector(LevelCollectorArgs{
		MonitorId:  m.monitorId,
		Engine:     m.engine,
		PlanLoader: m.planLoader,
		Sinks:      m.sinks,
	})
	go m.lpc.Run(m.stopChan, m.doneChanLPC)

	// ----------------------------------------------------------------------
	// Level plan adjuster (LPA)

	status.Monitor(m.monitorId, "monitor", "setting first level")

	if m.config.Plans.Adjust.Enabled() {
		m.doneChanLPA = make(chan struct{})
		m.lpa = NewLevelAdjuster(LevelAdjusterArgs{
			MonitorId: m.monitorId,
			Config:    m.config.Plans.Adjust,
			DB:        m.db,
			LPC:       m.lpc,
			HA:        ha.Disabled,
		})
	}

	// Run option level plan adjuster (LPA). When enabled, the LPA checks the
	// state of MySQL . If the state changes, it calls lpc.ChangePlan to change
	// the plan as configured by config.monitors.M.plans.adjust.<state>.
	if m.lpa != nil {
		go m.lpa.Run(m.stopChan, m.doneChanLPA)
	} else {
		// When the lpa is not enabled, we need to get the party started by
		// setting the first (and only) plan: "". When lpc.ChangePlan passes that
		// along to planLoader.Plan, the plan loader will automatically find
		// and return the first plan by precedence: first plan from table, or
		// first plan file, or internal plan--trying monitor plans first, then
		// default plans. So it always finds something: the default internal plan,
		// if nothing else.
		//
		// Also, without an lpa, monitors default to active state.
		if err := m.lpc.ChangePlan(blip.STATE_ACTIVE, ""); err != nil {
			blip.Debug(err.Error())
			// @todo
		}
	}

	m.runMux.Unlock() // -- BOOT --------------------------------------------

	status.RemoveComponent(m.monitorId, "monitor")

	// Run level plan collector (LPC)
	select {
	case <-m.stopChan:
	case <-m.doneChanLPC:
	}
}

func (m *Monitor) stop() bool {
	m.runMux.Lock()
	defer m.runMux.Unlock()
	return m.stopped
}

func (m *Monitor) Stop() error {
	m.runMux.Lock()
	defer m.runMux.Unlock()
	if m.stopped {
		return nil
	}
	m.stopped = true

	defer event.Sendf(event.MONITOR_STOPPED, m.monitorId)

	close(m.stopChan)
	if m.db != nil {
		m.db.Close()
	}

	timeout := time.After(4 * time.Second)

	if m.promAPI != nil {
		select {
		case <-m.doneChanProm:
		case <-timeout:
			return fmt.Errorf("timeout waiting for prom API to stop")
		}
	}

	if m.lpa != nil {
		select {
		case <-m.doneChanLPA:
		case <-timeout:
			return fmt.Errorf("timeout waiting for level adjuster to stop")
		}
	}

	if m.doneChanHBW != nil {
		select {
		case <-m.doneChanHBW:
		case <-timeout:
			return fmt.Errorf("timeout waiting for heartbeat writer to stop")
		}

	}

	select {
	case <-m.doneChanLPC:
	case <-timeout:
		return fmt.Errorf("timeout waiting for collector to stop")
	}

	return nil
}
