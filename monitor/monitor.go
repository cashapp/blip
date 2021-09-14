// Package monitor provides the Monitor type that monitors one MySQL instnace.
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
	"github.com/square/blip/sink"
)

type factory struct {
	dbMaker    blip.DbFactory
	planLoader *plan.Loader
}

func NewFactory(dbMaker blip.DbFactory, planLoader *plan.Loader) factory {
	return factory{
		dbMaker:    dbMaker,
		planLoader: planLoader,
	}
}

func (f factory) Make(cfg blip.ConfigMonitor) blip.Monitor {
	return &monitor{
		monitorId:  blip.MonitorId(cfg),
		config:     cfg,
		dbMaker:    f.dbMaker,
		planLoader: f.planLoader,
	}
}

// --------------------------------------------------------------------------

// monitor implements Monitor.
type monitor struct {
	// Factory values
	monitorId  string
	config     blip.ConfigMonitor
	dbMaker    blip.DbFactory
	planLoader *plan.Loader

	// monitor and sub-components
	engine *Engine
	db     *sql.DB
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

var _ blip.Monitor = &monitor{}

// MonitorId returns the monitor ID. This method implements blip.monitor.
func (m *monitor) MonitorId() string {
	return m.monitorId
}

// DB returns the low-level database connection. This method implements blip.monitor.
func (m *monitor) DB() *sql.DB {
	return m.db
}

// Config returns the monitor config. This method implements blip.monitor.
func (m *monitor) Config() blip.ConfigMonitor {
	return m.config
}

// Start starts monitoring the database if no error is returned.
func (m *monitor) Start() error {
	var err error

	m.db, err = m.dbMaker.Make(m.config)
	if err != nil {
		return err // @todo
	}

	sinks := []blip.Sink{}
	for sinkName, opts := range m.config.Sinks {
		sink, err := sink.Make(sinkName, m.monitorId, opts)
		if err != nil {
			return err
		}
		sinks = append(sinks, sink)
		blip.Debug("%s sends to %s", m.monitorId, sinkName)
	}
	if len(sinks) == 0 && !blip.Strict {
		blip.Debug("using log sink")
		sink, _ := sink.Make("log", m.monitorId, map[string]string{})
		sinks = append(sinks, sink)
	}

	m.Mutex = &sync.Mutex{}
	m.stopChan = make(chan struct{})
	m.engine = NewEngine(m.monitorId, m.db)
	m.lpc = NewLevelCollector(LevelCollectorArgs{
		Engine:     m.engine,
		PlanLoader: m.planLoader,
		Sinks:      sinks,
	})
	go m.run()
	return nil
}

func (m *monitor) run() {
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

	// Run option level plan adjuster (lpa). When enabled, the lpa checks the
	// state of MySQL . If the state changes,
	// it calls lpc.ChangePlan to change the plan as configured by
	// config.monitors.M.plans.adjust.<state>.
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

func (m *monitor) Stop() error {
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
