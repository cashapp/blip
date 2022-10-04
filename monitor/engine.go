// Copyright 2022 Block, Inc.

package monitor

import (
	"context"
	"database/sql"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cashapp/blip"
	"github.com/cashapp/blip/event"
	"github.com/cashapp/blip/metrics"
	"github.com/cashapp/blip/proto"
	"github.com/cashapp/blip/status"
)

// CollectParallel sets how many domains to collect in parallel. Currently, this
// is not configurable via Blip config; it can only be changed via integration.
var CollectParallel = 2

// amc represents one active metric collector, including its cleanup func (if any)
// and its last error. There's only one mc per domain, even if the domain is used
// at multiple levels (because the mc prepares itself for each level). When plans
// change, we start over (we don't reset/reuse mcs): discard all old mc (calling
// their cleanup funcs), then create a new list of mcs.
type amc struct {
	c       blip.Collector
	cleanup func()
	err     error
}

// Engine does the real work: collect metrics.
type Engine struct {
	cfg       blip.ConfigMonitor
	db        *sql.DB
	monitorId string
	// --
	event event.MonitorReceiver
	sem   chan bool // semaphore for CollectParallel

	planMux *sync.RWMutex
	plan    blip.Plan
	atLevel map[string][]blip.Collector // keyed on level

	mcMux  *sync.Mutex
	mcList map[string]*amc // keyed on domain

	statusMux *sync.Mutex
	status    proto.MonitorEngineStatus

	collectAll  uint64
	collectSome uint64
	collectFail uint64
}

func NewEngine(cfg blip.ConfigMonitor, db *sql.DB) *Engine {
	sem := make(chan bool, CollectParallel)
	for i := 0; i < CollectParallel; i++ {
		sem <- true
	}

	return &Engine{
		cfg:       cfg,
		db:        db,
		monitorId: cfg.MonitorId,
		// --
		event: event.MonitorReceiver{MonitorId: cfg.MonitorId},
		sem:   sem,

		planMux: &sync.RWMutex{},
		atLevel: map[string][]blip.Collector{},

		mcMux:  &sync.Mutex{},
		mcList: map[string]*amc{},

		statusMux: &sync.Mutex{},
		status:    proto.MonitorEngineStatus{},
	}
}

func (e *Engine) MonitorId() string {
	return e.monitorId
}

func (e *Engine) DB() *sql.DB {
	return e.db
}

func (e *Engine) Status() proto.MonitorEngineStatus {
	e.statusMux.Lock()
	cp := e.status // copy
	e.statusMux.Unlock()

	cp.CollectAll = atomic.LoadUint64(&e.collectAll)
	cp.CollectSome = atomic.LoadUint64(&e.collectSome)
	cp.CollectFail = atomic.LoadUint64(&e.collectFail)

	e.mcMux.Lock()
	errs := map[string]string{}
	for domain := range e.mcList {
		if e.mcList[domain].err == nil {
			continue
		}
		errs[domain] = e.mcList[domain].err.Error()
	}
	e.mcMux.Unlock()

	if len(errs) > 0 {
		cp.CollectorErrors = errs
	}

	return cp
}

// Prepare prepares the monitor to collect metrics for the plan. The monitor
// must be successfully prepared for Collect() to work because Prepare()
// initializes metrics collectors for every level of the plan. Prepare() can
// be called again when, for example, the LevelAdjuster (LPA) detects a state
// change and calls the LevelCollector (LPC) to change plans, which than calls
// this func with the new state plan. (Each monitor has its own LPA and LPC.)
//
// Do not call this func concurrently! It does not guard against concurrent
// calls. Instead, serialization is handled by the only caller: LevelCollector.ChangePlan().
func (e *Engine) Prepare(ctx context.Context, plan blip.Plan, before, after func()) error {
	blip.Debug("%s: prepare %s (%s)", e.monitorId, plan.Name, plan.Source)
	e.event.Sendf(event.ENGINE_PREPARE, plan.Name)
	status.Monitor(e.monitorId, "engine-prepare", "preparing plan %s", plan.Name)

	// Report last error, if any
	var lerr error
	defer func() {
		e.statusMux.Lock()
		if lerr != nil {
			e.status.Error = lerr.Error()
			e.event.Errorf(event.ENGINE_PREPARE_ERROR, lerr.Error())
			status.Monitor(e.monitorId, "engine-prepare", "error: %s", lerr)
		} else {
			// success
			status.RemoveComponent(e.monitorId, "engine-prepare")
			e.status.Error = ""
			e.event.Sendf(event.ENGINE_PREPARE_SUCCESS, plan.Name)
		}
		e.statusMux.Unlock()
	}()

	e.statusMux.Lock()
	e.status.Connected = false
	e.statusMux.Unlock()

	// Connect to MySQL. DO NOT loop and retry; try once and return on error
	// to let the caller (a LevelCollector.changePlan goroutine) retry with backoff.
	status.Monitor(e.monitorId, "engine-prepare", "connecting to MySQL")
	dbctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	err := e.db.PingContext(dbctx)
	cancel()
	if err != nil {
		lerr = fmt.Errorf("while connecting to MySQL: %s", err)
		return lerr
	}

	e.statusMux.Lock()
	e.status.Connected = true
	e.statusMux.Unlock()

	// Create and prepare metric collectors for every level. Return on error
	// because the error might be fatal, e.g. something misconfigured and the
	// plan cannot work.
	mcNew := map[string]*amc{} // keyed on domain
	atLevel := map[string][]blip.Collector{}
	for levelName, level := range plan.Levels {
		for domain, _ := range level.Collect {

			// Make collector if needed
			mc, ok := mcNew[domain]
			if !ok {
				// Make and prepare collector once because collectors prepare
				// themselves for all levels in the plan
				c, err := metrics.Make(
					domain,
					blip.CollectorFactoryArgs{
						Config:    e.cfg,
						DB:        e.db,
						MonitorId: e.monitorId,
					},
				)
				if err != nil {
					lerr = fmt.Errorf("while making %s collector: %s", domain, err)
					return lerr
				}

				status.Monitor(e.monitorId, "engine-prepare", "preparing plan %s level %s collector %s",
					plan.Name, levelName, domain)

				cleanup, err := c.Prepare(ctx, plan)
				if err != nil {
					lerr = fmt.Errorf("while preparing %s/%s/%s: %s", plan.Name, levelName, domain, err)
					return lerr
				}

				mc = &amc{
					c:       c,
					cleanup: cleanup,
				}
				mcNew[domain] = mc
			}

			// At this level, collect from this domain
			atLevel[levelName] = append(atLevel[levelName], mc.c)
		}
	}

	// Successfully prepared the plan
	status.Monitor(e.monitorId, "engine-prepare", "finalizing plan %s", plan.Name)

	before() // notify caller (LPC.changePlan) that we're about to swap the plan

	e.planMux.Lock() // LOCK plan -------------------------------------------
	e.mcMux.Lock()   // LOCK mc

	// Clean up old mc before swapping lists. Currently, the repl collector
	// uses this to stop its heartbeat.BlipReader goroutine.
	for _, mc := range e.mcList {
		if mc.cleanup != nil {
			blip.Debug("%s cleanup", mc.c.Domain())
			mc.cleanup()
		}
	}
	e.mcList = mcNew    // new mcs
	e.plan = plan       // new plan
	e.atLevel = atLevel // new levels

	e.mcMux.Unlock()   // UNLCOK mc
	e.planMux.Unlock() // UNLOCK plan ---------------------------------------

	blip.Debug("changed plan to %s", plan.Name)

	after() // notify caller (LPC.changePlan) that we have swapped the plan

	e.statusMux.Lock()
	e.status.Plan = plan.Name
	e.statusMux.Unlock()

	return nil
}

func (e *Engine) Collect(ctx context.Context, levelName string) (*blip.Metrics, error) {
	blip.Debug("collecting plan %s level %s", e.plan.Name, levelName)
	engineNo := status.MonitorMulti(e.monitorId, "engine", "collecting at %s/%s", e.plan.Name, levelName)
	defer status.RemoveComponent(e.monitorId, engineNo)

	//
	// *** This func can run concurrently! ***
	//
	// READ lock while collecting so Prepare cannot change plan while using it.
	// Must be read lock to allow concurrent calls.
	e.planMux.RLock()
	defer e.planMux.RUnlock()

	// All metric collectors at this level
	collectors := e.atLevel[levelName]
	if collectors == nil {
		blip.Debug("%s: no collectors at %s/%s, ignoring", e.monitorId, e.plan.Name, levelName)
		return nil, nil
	}

	// Serialize writes to metrics struct because CollectParallel number of collectors
	// run in parallel
	mux := &sync.Mutex{}
	metrics := &blip.Metrics{
		Plan:      e.plan.Name,
		Level:     levelName,
		MonitorId: e.monitorId,
		Values:    make(map[string][]blip.MetricValue, len(collectors)),
		Begin:     time.Now(),
	}
	errs := map[string]error{}

	// Collect metrics for each domain in parallel (limit: CollectParallel)
	var wg sync.WaitGroup
	for i := range collectors {
		<-e.sem
		wg.Add(1)
		go func(mc blip.Collector) {
			defer func() {
				e.sem <- true
				wg.Done()

				// Handle collector panic
				if r := recover(); r != nil {
					b := make([]byte, 4096)
					n := runtime.Stack(b, false)
					perr := fmt.Errorf("PANIC: monitor ID %s: %v\n%s", e.monitorId, r, string(b[0:n]))
					e.event.Errorf(event.COLLECTOR_PANIC, perr.Error())
					mux.Lock()
					errs[mc.Domain()] = perr
					mux.Unlock()
				}
			}()

			// **************************************************************
			// COLLECT METRICS
			//
			// Collect metrics in this domain. This is where metrics collection
			// happens: this domain-specific blip.Collector queries MySQL and
			// returns blip.Metrics at this level.
			vals, err := mc.Collect(ctx, levelName)
			// **************************************************************

			mux.Lock()
			errs[mc.Domain()] = err // clear or set error
			if len(vals) > 0 {      // save metrics, if any
				metrics.Values[mc.Domain()] = vals
			}
			mux.Unlock()
		}(collectors[i])
	}

	// Wait for all collectors to finish, then record end time
	wg.Wait()
	metrics.End = time.Now()

	// Process collector errors, if any
	errCount := 0
	e.mcMux.Lock()
	for domain, err := range errs {
		// Update MonitorEngineStatus: set new error or clear old error
		if err == nil {
			e.mcList[domain].err = nil
			continue
		}
		errCount += 1
		errMsg := fmt.Sprintf("error collecting %s/%s/%s: %s", e.plan.Name, levelName, domain, err)
		e.mcList[domain].err = fmt.Errorf("[%s] %s", metrics.Begin, errMsg) // status
		e.event.Errorf(event.COLLECTOR_ERROR, errMsg)                       // log by default
	}
	e.mcMux.Unlock()

	// Total success? Yes if no errors.
	if errCount == 0 {
		atomic.AddUint64(&e.collectAll, 1)
		return metrics, nil
	}

	// Partial success? Yes if there are some metrics values.
	if len(metrics.Values) > 0 {
		atomic.AddUint64(&e.collectSome, 1)
		return metrics, fmt.Errorf("%d errors collecting %s/%s: some metrics were not collected",
			errCount, e.plan.Name, levelName)
	}

	// Errors and zero metrics: all collectors failee
	atomic.AddUint64(&e.collectFail, 1)
	return nil, fmt.Errorf("failed to collect %s/%s", e.plan.Name, levelName)
}

// Stop the engine and cleanup any metrics associated with it.
// TODO: There is a possible race condition when this is called. Since
// Engine.Collect is called as a go-routine, we could have an invocation
// of the function block waiting for Engine.Stop to runlock planMux,
// after which Collect would run after cleanup has been called.
// This could result in a panic, though that should be caught and logged.
// Since the monitor is stopping anyway this isn't a huge issue.
func (e *Engine) Stop() {
	blip.Debug("Stopping engine...")
	e.planMux.Lock()
	defer e.planMux.Unlock()

	e.mcMux.Lock()
	defer e.mcMux.Unlock()

	// Clean up the monitors
	for _, mc := range e.mcList {
		if mc.cleanup != nil {
			blip.Debug("%s cleanup", mc.c.Domain())
			mc.cleanup()
		}
	}
}
