package server

import (
	"database/sql"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/square/blip"
	"github.com/square/blip/collect"
	"github.com/square/blip/dbconn"
	"github.com/square/blip/event"
	"github.com/square/blip/heartbeat"
	"github.com/square/blip/level"
	"github.com/square/blip/metrics"
	"github.com/square/blip/monitor"
	"github.com/square/blip/sink"
)

type DbMonFactory interface {
	Make(blip.ConfigMonitor) *DbMon
}

var _ DbMonFactory = dbmonFactory{}

type dbmonFactory struct {
	mcMaker    metrics.CollectorFactory
	dbMaker    dbconn.Factory
	planLoader *collect.PlanLoader
	sinks      []sink.Sink
}

func (f dbmonFactory) Make(cfg blip.ConfigMonitor) *DbMon {
	return &DbMon{
		MonitorId:  dbconn.MonitorId(cfg),
		Config:     cfg,
		MCMaker:    f.mcMaker,
		DBMaker:    f.dbMaker,
		PlanLoader: f.planLoader,
		Sinks:      f.sinks,
	}
}

type DbMon struct {
	MonitorId  string
	Config     blip.ConfigMonitor
	DB         *sql.DB
	MCMaker    metrics.CollectorFactory
	DBMaker    dbconn.Factory
	PlanLoader *collect.PlanLoader
	Sinks      []sink.Sink
	// --
	Monitor *monitor.Monitor
	LPC     level.Collector
	LPA     level.Adjuster
	HB      heartbeat.Monitor
	// --
	*sync.Mutex
	metronome   *sync.Cond
	stopChan    chan struct{}
	doneChanLPA chan struct{}
	doneChanLPC chan struct{}
	doneChanHB  chan struct{}
	stopped     bool
}

func (dbmon *DbMon) Start() error {
	db, err := dbmon.DBMaker.Make(dbmon.Config)
	if err != nil {
		return err // @todo
	}
	dbmon.DB = db
	dbmon.Mutex = &sync.Mutex{}
	dbmon.metronome = sync.NewCond(&sync.Mutex{})
	dbmon.stopChan = make(chan struct{})
	dbmon.Monitor = monitor.NewMonitor(dbmon.MonitorId, dbmon.DB, dbmon.MCMaker)
	dbmon.LPC = level.NewCollector(level.CollectorArgs{
		Monitor:    dbmon.Monitor,
		Metronome:  dbmon.metronome,
		PlanLoader: dbmon.PlanLoader,
		Sinks:      dbmon.Sinks,
	})
	go dbmon.run()
	return nil
}

func (dbmon *DbMon) run() {
	defer func() {
		if err := recover(); err != nil {
			b := make([]byte, 4096)
			n := runtime.Stack(b, false)
			log.Printf("PANIC: %s\n%s", err, string(b[0:n]))
		}
		dbmon.Stop()
	}()

	// Run level plan collector (LPC). This is the foundation of dbmon.
	// It's rock'n out with the metronome to invoke the Monitor at each level
	// frequency, which is how metrics are collected according to level plan.
	dbmon.doneChanLPC = make(chan struct{})
	go dbmon.LPC.Run(dbmon.stopChan, dbmon.doneChanLPC)

	// Run option level plan adjuster (LPA). When enabled, the LPA checks the
	// state of MySQL on every metronome tick (every 500ms). If the state changes,
	// it calls LPC.ChangePlan to change the plan as configured by
	// config.monitors.M.plans.adjust.<state>.
	if dbmon.Config.Plans.Adjust.Freq != "" {
		dbmon.doneChanLPA = make(chan struct{})
		dbmon.LPA = level.NewAdjuster(dbmon.Monitor, dbmon.metronome, dbmon.LPC)
		go dbmon.LPA.Run(dbmon.stopChan, dbmon.doneChanLPA)
	} else {
		// When the LPA is not enabled, we need to get the party started by
		// setting the first (and only) plan: "". When LPC.ChangePlan passes that
		// along to PlanLoader.Plan, the plan loader will automatically find
		// and return the first plan by precedence: first plan from table, or
		// first plan file, or internal plan--trying monitor plans first, then
		// default plans. So it always finds something: the default internal plan,
		// if nothing else.
		//
		// Also, without an LPA, monitors default to active state.
		dbmon.LPC.ChangePlan(blip.STATE_ACTIVE, "")
	}

	// Run optional heartbeat monitor to monitor replication lag. When enabled,
	// the heartbeat (HB) writes a high-resolution timestamp to a row in a table
	// at the configured frequence: config.monitors.M.heartbeat.freq.
	if dbmon.Config.Heartbeat.Freq != "" {
		dbmon.doneChanHB = make(chan struct{})
		dbmon.HB = heartbeat.NewMonitor(dbmon.Config.Heartbeat, dbmon.DB) // @todo
		go dbmon.HB.Run(dbmon.stopChan, dbmon.doneChanHB)
	}

	// @todo inconsequential race condition

	// Run the metronome that is ithe secret force behind everything:
	// the LPC, LPA,and HB work only when the metronome ticks.
	// In between ticks, these components contemplate 500 billion picoseconds of silence.
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			dbmon.metronome.Broadcast()
		case <-dbmon.stopChan:
			blip.Debug("%s: metronome stopped", dbmon.MonitorId)
			return

		}
	}
}

func (dbmon *DbMon) Stop() {
	dbmon.Lock()
	defer dbmon.Unlock()
	if dbmon.stopped {
		return
	}
	dbmon.stopped = true

	defer event.Sendf(event.MONITOR_STOPPED, dbmon.MonitorId)

	close(dbmon.stopChan)
	dbmon.DB.Close()

	running := 0
	if dbmon.doneChanLPC != nil {
		running += 1 // LPC
	}
	if dbmon.doneChanLPA != nil {
		running += 1 // LPA
	}
	if dbmon.doneChanHB != nil {
		running += 1 // + Heartbeat
	}

WAIT_LOOP:
	for running > 0 {
		blip.Debug("%s: %d running", dbmon.MonitorId, running)
		select {
		case <-dbmon.doneChanLPA:
			blip.Debug("%s: LPA done", dbmon.MonitorId)
			dbmon.doneChanLPA = nil
			running -= 1
		case <-dbmon.doneChanLPC:
			blip.Debug("%s: LPC done", dbmon.MonitorId)
			dbmon.doneChanLPC = nil
			running -= 1
		case <-dbmon.doneChanHB:
			blip.Debug("%s: HB done", dbmon.MonitorId)
			dbmon.doneChanHB = nil
			running -= 1
		case <-time.After(2 * time.Second):
			// @todo
			break WAIT_LOOP
		}
	}
}
