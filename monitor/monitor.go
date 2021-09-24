// Package monitor provides the Monitor type that monitors one MySQL instnace.
// All monitoring activity happens in the package. The rest of Blip is mostly
// to set up and support Monitor instances.
//
// The most important types in this package are Engine, LevelCollector, and
// LevelAdjuster.
package monitor

import (
	"database/sql"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/square/blip"
	"github.com/square/blip/event"
	"github.com/square/blip/ha"
	"github.com/square/blip/heartbeat"
	"github.com/square/blip/plan"
	"github.com/square/blip/prom"
)

// Monitor monitors one MySQL instance. A monitor is completely self-contained;
// monitors share nothing. Therefore, each monitor is completely indepedent, too.
//
// The Monitor type is just an "outer wrapper" that runs the core components
// that do real work: Engine, LevelCollector (LPC), optional LevelAdjuster (LPA),
// and optinoal Heartbeat (HB). You can view it like:
//
//   Blip -> Monitor[ Engine + LPA + LPC + HB ]
//
// Blip only calls/uses the Monitor; nothing outside the Monitor can access those
// core components.
//
// Loading monitors (Loader type) is dyanmic, but individual Monitor are not.
// You can start/stop a Monitor, but to change it, you must stop, discard, and
// create a new Monitor via the monitor Loader.
type Monitor struct {
	monitorId  string
	config     blip.ConfigMonitor
	planLoader *plan.Loader
	sinks      []blip.Sink
	dbMaker    blip.DbFactory
	db         *sql.DB

	// Core components
	engine *Engine
	lpc    LevelCollector
	lpa    LevelAdjuster
	hbw    heartbeat.Writer
	hbr    heartbeat.Reader

	// Control chans and sync
	*sync.Mutex
	stopChan    chan struct{}
	doneChanLPA chan struct{}
	doneChanLPC chan struct{}
	doneChanHBW chan struct{}
	doneChanHBR chan struct{}
	stopped     bool
}

// MonitorId returns the monitor ID. This method implements *Monitor.
func (m *Monitor) MonitorId() string {
	return m.monitorId
}

// DB returns the low-level database connection. This method implements *Monitor.
func (m *Monitor) DB() *sql.DB {
	return m.db
}

// Config returns the monitor config. This method implements *Monitor.
func (m *Monitor) Config() blip.ConfigMonitor {
	return m.config
}

func (m *Monitor) Status() string {
	return "todo"
}

// Start starts monitoring the database if no error is returned.
func (m *Monitor) Start() error {
	var err error

	m.db, err = m.dbMaker.Make(m.config)
	if err != nil {
		return err // @todo
	}

	m.Mutex = &sync.Mutex{}
	m.stopChan = make(chan struct{})
	m.engine = NewEngine(m.monitorId, m.db)
	m.lpc = NewLevelCollector(LevelCollectorArgs{
		Engine:     m.engine,
		PlanLoader: m.planLoader,
		Sinks:      m.sinks,
	})
	go m.run()
	return nil
}

func (m *Monitor) run() {
	defer func() {
		if err := recover(); err != nil {
			b := make([]byte, 4096)
			n := runtime.Stack(b, false)
			log.Printf("PANIC: %s\n%s", err, string(b[0:n]))
		}
		m.Stop()
	}()

	if m.config.Exporter.Bind != "" {
		exp := prom.NewExporter(
			m.monitorId,
			m.db,
		)
		if err := exp.Prepare(blip.PromPlan()); err != nil {
			// @todo move to Boot
			blip.Debug(err.Error())
			return
		}
		api := prom.NewAPI(m.config.Exporter.Bind, m.monitorId, exp)
		go api.Run()
		if blip.True(m.config.Exporter.Legacy) {
			blip.Debug("legacy mode")
			<-m.stopChan
			return
		}
	}

	// Run option level plan adjuster (LPA). When enabled, the LPA checks the
	// state of MySQL . If the state changes, it calls lpc.ChangePlan to change
	// the plan as configured by config.monitors.M.plans.adjust.<state>.
	if m.config.Plans.Adjust.Enabled() {
		m.doneChanLPA = make(chan struct{})
		m.lpa = NewLevelAdjuster(LevelAdjusterArgs{
			MonitorId: m.monitorId,
			Config:    m.config.Plans.Adjust,
			DB:        m.db,
			LPC:       m.lpc,
			HA:        ha.Disabled,
		})
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

	// Run optional heartbeat monitor to monitor replication lag. When enabled,
	// the heartbeat (hb) writes a high-resolution timestamp to a row in a table
	// at the configured frequence: config.monitors.M.heartbeat.freq.
	if !blip.True(m.config.Heartbeat.Disable) {

		if !blip.True(m.config.Heartbeat.DisableWrite) {
			m.hbw = heartbeat.NewWriter(m.monitorId, m.db)
			m.doneChanHBW = make(chan struct{})
			go m.hbw.Write(m.stopChan, m.doneChanHBW)
		}

		if !!blip.True(m.config.Heartbeat.DisableRead) &&
			(len(m.config.Heartbeat.Source) > 0 || !blip.True(m.config.Heartbeat.DisableAutoSource)) {
			var sf heartbeat.SourceFinder
			if len(m.config.Heartbeat.Source) > 0 {
				sf = heartbeat.NewStaticSourceList(m.config.Heartbeat.Source, m.db)
			} else if !blip.True(m.config.Heartbeat.DisableAutoSource) {
				sf = heartbeat.NewAutoSourceFinder() // @todo
			} else {
				panic("no repl sources and auto-source disable")
			}
			m.hbr = heartbeat.NewReader(
				m.config,
				m.db,
				heartbeat.NewSlowFastWaiter(),
				sf,
			)
			m.doneChanHBR = make(chan struct{})
			go m.hbr.Read(m.stopChan, m.doneChanHBR)
		} else {
			blip.Debug("heartbeat read disabled: no sources, aut-source dissabled")
		}
	}

	// @todo inconsequential race condition

	// Run level plan collector (LPC)
	m.doneChanLPC = make(chan struct{})
	if err := m.lpc.Run(m.stopChan, m.doneChanLPC); err != nil {
		blip.Debug(err.Error())
		// @todo
	}
}

func (m *Monitor) Stop() error {
	m.Lock()
	defer m.Unlock()
	if m.stopped {
		return nil
	}
	m.stopped = true

	defer event.Sendf(event.MONITOR_STOPPED, m.monitorId)

	close(m.stopChan)
	m.db.Close()

	running := 0
	if m.doneChanLPC != nil {
		running += 1 // lpc
	}
	if m.doneChanLPA != nil {
		running += 1 // lpa
	}
	if m.doneChanHBW != nil {
		running += 1 // + Heartbeat writer
	}
	if m.doneChanHBR != nil {
		running += 1 // + Heartbeat reader
	}

WAIT_LOOP:
	for running > 0 {
		blip.Debug("%s: %d running", m.monitorId, running)
		select {
		case <-m.doneChanLPA:
			blip.Debug("%s: lpa done", m.monitorId)
			m.doneChanLPA = nil
			running -= 1
		case <-m.doneChanLPC:
			blip.Debug("%s: lpc done", m.monitorId)
			m.doneChanLPC = nil
			running -= 1
		case <-m.doneChanHBW:
			blip.Debug("%s: hb writer done", m.monitorId)
			m.doneChanHBW = nil
			running -= 1
		case <-m.doneChanHBR:
			blip.Debug("%s: hb reader done", m.monitorId)
			m.doneChanHBR = nil
			running -= 1
		case <-time.After(2 * time.Second):
			// @todo
			break WAIT_LOOP
		}
	}

	return nil
}
