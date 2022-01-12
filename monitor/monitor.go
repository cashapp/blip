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
	db      *sql.DB
	dsn     string
	engine  *Engine
	promAPI *prom.API
	lpc     LevelCollector
	lpa     LevelAdjuster
	hbw     *heartbeat.BlipWriter

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
		cfg:        args.Config,
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

func (m *Monitor) setErr(err error) {
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
		db, dsn, err := m.dbMaker.Make(m.cfg)
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

	if m.cfg.Exporter.Mode != "" {
		m.promAPI = prom.NewAPI(
			m.cfg.Exporter,
			m.monitorId,
			NewExporter(m.cfg.Exporter, NewEngine(m.monitorId, m.db)),
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
		if m.cfg.Exporter.Mode == blip.EXPORTER_MODE_LEGACY {
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
		err := m.planLoader.LoadMonitor(m.cfg, m.dbMaker)
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
	// at the configured frequence: cfg.monitors.M.heartbeat.freq.
	if m.cfg.Heartbeat.Freq != "" {
		status.Monitor(m.monitorId, "monitor", "starting heartbeat")
		m.hbw = heartbeat.NewBlipWriter(m.monitorId, m.db, m.cfg.Heartbeat)
		m.doneChanHBW = make(chan struct{})
		go m.hbw.Write(m.stopChan, m.doneChanHBW)
	}

	// ----------------------------------------------------------------------
	// Level plan collector (LPC)

	// LPC starts paused becuase there's no plan. The main loop in lpc.Run
	// does nothing until lpc.ChangePlan is called, which is done next either
	// indirectly via LPA or directly if LPA isn't enabled.
	m.engine = NewEngine(m.monitorId, m.db)
	m.lpc = NewLevelCollector(LevelCollectorArgs{
		Config:     m.cfg,
		Engine:     m.engine,
		PlanLoader: m.planLoader,
		Sinks:      m.sinks,
	})
	go m.lpc.Run(m.stopChan, m.doneChanLPC)

	// ----------------------------------------------------------------------
	// Level plan adjuster (LPA)

	status.Monitor(m.monitorId, "monitor", "setting first level")

	if m.cfg.Plans.Adjust.Enabled() {
		// Run option level plan adjuster (LPA). When enabled, the LPA checks
		// the state of MySQL. If the state changes, it calls lpc.ChangePlan
		// to change the plan as configured by config.monitors.M.plans.adjust.<state>.
		m.doneChanLPA = make(chan struct{})
		m.lpa = NewLevelAdjuster(LevelAdjusterArgs{
			MonitorId: m.monitorId,
			Config:    m.cfg.Plans.Adjust,
			DB:        m.db,
			LPC:       m.lpc,
			HA:        ha.Disabled,
		})
		go m.lpa.Run(m.stopChan, m.doneChanLPA) // start LPC indirectly
	} else {
		// When the LPA is not enabled, we must init the state and plan,
		// which are ACTIVE and first (""), respectively. Since LPA is
		// optional, this is the normal case: startup presuming MySQL is
		// active and use the monitor's first plan.
		//
		// Do need retry or error handling because ChangePlan tries forever,
		// or until called again.
		m.lpc.ChangePlan(blip.STATE_ACTIVE, "") // start LPC directly
	}

	m.runMux.Unlock() // -- BOOT --------------------------------------------

	status.RemoveComponent(m.monitorId, "monitor")

	// Block to keep monitor running until Stop is called
	<-m.stopChan
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
