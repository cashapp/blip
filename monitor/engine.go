// Copyright 2023 Block, Inc.

package monitor

import (
	"context"
	"database/sql"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/cashapp/blip"
	"github.com/cashapp/blip/event"
	"github.com/cashapp/blip/metrics"
	"github.com/cashapp/blip/status"
)

// CollectParallel sets how many domains to collect in parallel. Currently, this
// is not configurable via Blip config; it can only be changed via integration.
var CollectParallel = 2

// amc represents one active metric collector and its cleanup func (if any).
// There's only one mc per domain, even if the domain is used at multiple levels
// (because the mc prepares itself for each level). When plans change, we start
// over (we don't reset/reuse mcs): discard all old mc (calling their cleanup funcs)
// then create a new list of mcs.
type amc struct {
	c       blip.Collector
	cleanup func()
}

// Engine does the real work: collect metrics.
type Engine struct {
	cfg       blip.ConfigMonitor
	db        *sql.DB
	monitorId string
	// --
	event event.MonitorReceiver

	planMux *sync.RWMutex
	plan    blip.Plan
	atLevel map[string][]blip.Collector // keyed on level

	mcMux  *sync.Mutex
	mcList map[string]*amc // keyed on domain

	ctxList  map[string]context.CancelFunc
	domainWg map[string]*sync.WaitGroup

	mcContext  context.Context
	mcCancelFn context.CancelFunc

	wg sync.WaitGroup
}

func NewEngine(cfg blip.ConfigMonitor, db *sql.DB) *Engine {
	return &Engine{
		cfg:       cfg,
		db:        db,
		monitorId: cfg.MonitorId,
		// --
		event: event.MonitorReceiver{MonitorId: cfg.MonitorId},

		planMux: &sync.RWMutex{},
		atLevel: map[string][]blip.Collector{},

		mcMux:    &sync.Mutex{},
		mcList:   map[string]*amc{},
		ctxList:  map[string]context.CancelFunc{},
		domainWg: map[string]*sync.WaitGroup{},
	}
}

func (e *Engine) MonitorId() string {
	return e.monitorId
}

func (e *Engine) DB() *sql.DB {
	return e.db
}

// Prepare prepares the monitor to collect metrics for the plan. The monitor
// must be successfully prepared for Collect() to work because Prepare()
// initializes metrics collectors for every level of the plan. Prepare() can
// be called again when, for example, the PlanChanger detects a state change
// and calls the LevelCollector to change plans, which than calls this func with
// the new state plan.
//
// Do not call this func concurrently! It does not guard against concurrent
// calls. Serialization is handled by the only caller: LevelCollector.ChangePlan().
func (e *Engine) Prepare(ctx context.Context, plan blip.Plan, before, after func()) error {
	blip.Debug("%s: prepare %s (%s)", e.monitorId, plan.Name, plan.Source)
	e.event.Sendf(event.ENGINE_PREPARE, plan.Name)
	status.Monitor(e.monitorId, status.ENGINE_PREPARE, plan.Name)
	defer status.RemoveComponent(e.monitorId, status.ENGINE_PREPARE)

	// Report last error, if any
	var lerr error
	defer func() {
		if lerr != nil {
			e.event.Errorf(event.ENGINE_PREPARE_ERROR, lerr.Error())
			status.Monitor(e.monitorId, "error:"+status.ENGINE_PREPARE, lerr.Error())
		} else {
			// success
			status.RemoveComponent(e.monitorId, "error:"+status.ENGINE_PREPARE)
		}
	}()

	// Connect to MySQL. DO NOT loop and retry; try once and return on error
	// to let the caller (a LevelCollector.changePlan goroutine) retry with backoff.
	status.Monitor(e.monitorId, status.ENGINE_PREPARE, "%s: connect to MySQL", plan.Name)
	dbctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	err := e.db.PingContext(dbctx)
	cancel()
	if err != nil {
		lerr = fmt.Errorf("while connecting to MySQL: %s", err)
		return lerr
	}

	// Create and prepare metric collectors for every level. Return on error
	// because the error might be fatal, e.g. something misconfigured and the
	// plan cannot work.
	mcNew := map[string]*amc{} // keyed on domain
	atLevel := map[string][]blip.Collector{}
	for levelName, level := range plan.Levels {
		for domain := range level.Collect {

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

				status.Monitor(e.monitorId, status.ENGINE_PREPARE, "%s: prepare collector %s", plan.Name, domain)
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
	status.Monitor(e.monitorId, status.ENGINE_PREPARE, "%s: level-collector before callback", plan.Name)
	before() // notify caller (lco.changePlan) that we're about to swap the plan

	status.Monitor(e.monitorId, status.ENGINE_PREPARE, "%s: finalize", plan.Name)
	e.planMux.Lock() // LOCK plan -------------------------------------------
	e.mcMux.Lock()   // LOCK mc

	// Cancel prior collector contexts, if any
	if e.mcCancelFn != nil {
		e.mcCancelFn()
	}

	// Clean up old mc before swapping lists. Currently, the repl collector
	// uses this to stop its heartbeat.BlipReader goroutine.
	for _, mc := range e.mcList {
		if mc.cleanup != nil {
			blip.Debug("%s: %s cleanup", e.monitorId, mc.c.Domain())
			status.Monitor(e.monitorId, status.ENGINE_PREPARE, "%s: finalize (%s cleanup)", plan.Name, mc.c.Domain())
			mc.cleanup()
		}
	}
	e.mcList = mcNew                                                     // new mcs
	e.plan = plan                                                        // new plan
	e.atLevel = atLevel                                                  // new levels
	e.mcContext, e.mcCancelFn = context.WithCancel(context.Background()) // New context for collectors

	e.mcMux.Unlock()   // UNLOCK mc
	e.planMux.Unlock() // UNLOCK plan ---------------------------------------

	status.Monitor(e.monitorId, status.ENGINE_PLAN, plan.Name)
	e.event.Sendf(event.ENGINE_PREPARE_SUCCESS, plan.Name)

	status.Monitor(e.monitorId, status.ENGINE_PREPARE, "%s: level-collector after callback", plan.Name)
	after() // notify caller (lco.changePlan) that we have swapped the plan

	return nil
}

func (e *Engine) CollectSynchronous(ctx context.Context, levelName string) (*blip.Metrics, error) {
	blip.Debug("%s: collect plan %s level %s", e.monitorId, e.plan.Name, levelName)
	collectNo := status.MonitorMulti(e.monitorId, status.ENGINE_COLLECT, "%s/%s: acquire read lock", e.plan.Name, levelName)
	defer status.RemoveComponent(e.monitorId, collectNo)

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

	status.Monitor(e.monitorId, collectNo, "%s/%s: run collectors", e.plan.Name, levelName)

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
	sem := make(chan bool, CollectParallel) // semaphore for CollectParallel
	for i := 0; i < CollectParallel; i++ {
		sem <- true
	}
	var wg sync.WaitGroup
	for i := range collectors {
		<-sem
		wg.Add(1)
		go func(mc blip.Collector) {
			defer func() {
				sem <- true
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
			dbctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			vals, err := mc.Collect(dbctx, levelName)
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
	status.Monitor(e.monitorId, collectNo, "%s/%s: wait for collectors", e.plan.Name, levelName)
	wg.Wait()
	metrics.End = time.Now()

	// Process collector errors, if any
	status.Monitor(e.monitorId, collectNo, "%s/%s: log errors", e.plan.Name, levelName)
	errCount := 0
	for domain, err := range errs {
		// Update MonitorEngineStatus: set new error or clear old error
		if err != nil {
			errCount += 1
			errMsg := fmt.Sprintf("error collecting %s/%s/%s: %s", e.plan.Name, levelName, domain, err)
			status.Monitor(e.monitorId, "error:"+domain, "at %s: %s", metrics.Begin, errMsg)
			e.event.Errorf(event.COLLECTOR_ERROR, errMsg) // log by default
		} else {
			status.RemoveComponent(e.monitorId, "error:"+domain)
		}
	}

	status.Monitor(e.monitorId, collectNo, "%s/%s: return metrics", e.plan.Name, levelName)

	// Total success? Yes if no errors.
	if errCount == 0 {
		return metrics, nil
	}

	// Partial success? Yes if there are some metrics values.
	if len(metrics.Values) > 0 {
		return metrics, fmt.Errorf("%d errors collecting %s/%s: some metrics were not collected",
			errCount, e.plan.Name, levelName)
	}

	// Errors and zero metrics: all collectors failee
	return nil, fmt.Errorf("failed to collect %s/%s", e.plan.Name, levelName)
}

// We need a context that wraps a larger context for plan changes so that we can cancel collections that are
// in process for plan changes

func (e *Engine) Collect(levelName string, sinkChan chan<- sinkPayload) {
	blip.Debug("%s: collect plan %s level %s", e.monitorId, e.plan.Name, levelName)
	collectNo := status.MonitorMulti(e.monitorId, status.ENGINE_COLLECT, "%s/%s: acquire read lock", e.plan.Name, levelName)
	defer status.RemoveComponent(e.monitorId, collectNo)

	//
	// *** This func can run concurrently! ***
	//
	// READ lock while collecting so Prepare cannot change plan while using it.
	// Must be read lock to allow concurrent calls.
	e.planMux.RLock()
	defer e.planMux.RUnlock()

	// Did the engine context expire while we were waiting for the lock?
	// If so this could be due to a plan change or shutting down.
	select {
	case <-e.mcContext.Done():
		return
	default:
	}

	// All metric collectors at this level
	collectors := e.atLevel[levelName]
	if collectors == nil {
		blip.Debug("%s: no collectors at %s/%s, ignoring", e.monitorId, e.plan.Name, levelName)
		return
	}

	status.Monitor(e.monitorId, collectNo, "%s/%s: run collectors", e.plan.Name, levelName)

	for i := range collectors {
		domain := collectors[i].Domain()
		if cancelFn := e.ctxList[domain]; cancelFn != nil {
			cancelFn()
		}

		oldWg := e.domainWg[domain]
		newWg := &sync.WaitGroup{}
		e.domainWg[domain] = newWg

		domainCtx, domainCancelFn := context.WithCancel(e.mcContext)
		e.ctxList[domain] = domainCancelFn

		// Indicate that we have a collector running in the engine
		e.wg.Add(1)
		// Indicate that we have a collector running for the domain
		newWg.Add(1)
		go func(ctx context.Context, mc blip.Collector, priorWg *sync.WaitGroup, wg *sync.WaitGroup) {
			defer e.wg.Done()

			// Wait for any prior invocation of the collection for this domain to be done
			if priorWg != nil {
				priorWg.Wait()
			}
			defer wg.Done()

			defer func() {
				// Handle collector panic
				if r := recover(); r != nil {
					b := make([]byte, 4096)
					n := runtime.Stack(b, false)
					perr := fmt.Errorf("PANIC: monitor ID %s: %v\n%s", e.monitorId, r, string(b[0:n]))
					e.event.Errorf(event.COLLECTOR_PANIC, perr.Error())
					// TODO
					/*mux.Lock()
					errs[mc.Domain()] = perr
					mux.Unlock()*/
				}
			}()

			// TODO: Should we use a single starting time for all domains
			// or set it specific to each domain?
			ts := time.Now()

			// **************************************************************
			// COLLECT METRICS
			//
			// Collect metrics in this domain. This is where metrics collection
			// happens: this domain-specific blip.Collector queries MySQL and
			// returns blip.Metrics at this level.
			vals, err := mc.Collect(ctx, levelName)
			// TODO: How do we handle errors when the return of
			// metrics is not synchronized across the domains?
			if err != nil {
				errMsg := fmt.Sprintf("error collecting %s/%s/%s: %s", e.plan.Name, levelName, domain, err)
				status.Monitor(e.monitorId, "error:"+domain, "at %s: %s", ts, errMsg)
				e.event.Errorf(event.COLLECTOR_ERROR, errMsg) // log by default
			} else {
				status.RemoveComponent(e.monitorId, "error:"+domain)
			}

			if len(vals) > 0 { // save metrics, if any
				metrics := &blip.Metrics{
					Plan:      e.plan.Name,
					Level:     levelName,
					MonitorId: e.monitorId,
					Begin:     ts,
					End:       time.Now(),
					Values: map[string][]blip.MetricValue{
						mc.Domain(): vals,
					},
				}

				sinkChan <- sinkPayload{
					metrics:   metrics,
					collectNo: "temp",
				}

			}
		}(domainCtx, collectors[i], oldWg, newWg)
	}

	// Note that we started all the collectors
	status.Monitor(e.monitorId, collectNo, "%s/%s: done starting collectors", e.plan.Name, levelName)
}

// Stop the engine and cleanup any metrics associated with it.
// TODO: There is a possible race condition when this is called. Since
// Engine.Collect is called as a go-routine, we could have an invocation
// of the function block waiting for Engine.Stop to unlock planMux,
// after which Collect would run after cleanup has been called.
// This could result in a panic, though that should be caught and logged.
// Since the monitor is stopping anyway this isn't a huge issue.
func (e *Engine) Stop() <-chan struct{} {
	blip.Debug("Stopping engine...")
	e.planMux.Lock()
	defer e.planMux.Unlock()

	e.mcMux.Lock()
	defer e.mcMux.Unlock()

	// Stop any collectors that are running
	if e.mcCancelFn != nil {
		e.mcCancelFn()
	}

	// Clean up the monitors
	for _, mc := range e.mcList {
		if mc.cleanup != nil {
			blip.Debug("%s cleanup", mc.c.Domain())
			mc.cleanup()
		}
	}

	doneChan := make(chan struct{})
	go func() {
		e.wg.Wait()
		close(doneChan)
	}()
	return doneChan
}
