package monitor

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"runtime"
	"sync"
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

// mc represents one active metric collector, including its cleanup func (if any)
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
	monitorId string
	db        *sql.DB
	// --
	event event.MonitorSink
	sem   chan bool // semaphore for CollectParallel

	planMux *sync.RWMutex
	plan    blip.Plan
	atLevel map[string][]blip.Collector // keyed on level

	mcMux  *sync.Mutex
	mcList map[string]*amc // keyed on domain

	statusMux *sync.Mutex
	status    proto.MonitorEngineStatus
}

func NewEngine(monitorId string, db *sql.DB) *Engine {
	sem := make(chan bool, CollectParallel)
	for i := 0; i < CollectParallel; i++ {
		sem <- true
	}

	return &Engine{
		monitorId: monitorId,
		db:        db,
		// --
		event: event.MonitorSink{MonitorId: monitorId},
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
	blip.Debug("%s: prepare: %+v", e.monitorId, plan)
	e.event.Sendf(event.MONITOR_PREPARE_PLAN, plan.Name)
	status.Monitor(e.monitorId, "engine-prepare", "preparing plan %s", plan.Name)

	// Report last error, if any
	var lerr error
	defer func() {
		e.statusMux.Lock()
		if lerr == nil {
			status.RemoveComponent(e.monitorId, "engine-prepare")
			e.status.Error = ""
		} else {
			e.status.Error = lerr.Error()
		}
		e.statusMux.Unlock()
	}()

	e.statusMux.Lock()
	e.status.Connected = false
	e.statusMux.Unlock()

	status.Monitor(e.monitorId, "engine-prepare", "connecting to MySQL")
	dbctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	lerr = e.db.PingContext(dbctx)
	cancel()
	if ctx.Err() != nil {
		lerr = fmt.Errorf("cancelled while connecting to MySQL")
		return nil // cancellation is not an error
	}
	if lerr != nil {
		status.Monitor(e.monitorId, "engine-prepare", "error connecting to MySQL: %s", lerr)
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

			status.Monitor(e.monitorId, "engine-prepare", "preparing plan %s level %s collector %s",
				plan.Name, levelName, domain)

			// Make collector if needed
			mc, ok := mcNew[domain]
			if !ok {
				// Make and prepare collector once because collectors prepare
				// themselves for all levels in the plan
				c, err := metrics.Make(
					domain,
					blip.CollectorFactoryArgs{
						MonitorId: e.monitorId,
						DB:        e.db,
					},
				)
				if err != nil {
					blip.Debug(err.Error())
					return err
				}

				cleanup, err := c.Prepare(ctx, plan)
				if err != nil {
					return err
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
	//
	// *** This func can run concurrently! ***
	//

	// READ lock while collecting so Prepare cannot change plan while using it.
	// Must be read lock to allow concurrent calls.
	e.planMux.RLock()
	defer e.planMux.RUnlock()

	mc := e.atLevel[levelName]
	if mc == nil {
		blip.Debug("%s no mc at level '%s'", e.monitorId, levelName)
		return nil, nil
	}

	status.Monitor(e.monitorId, "engine", "collecting plan %s level %s", e.plan.Name, levelName)
	blip.Debug("collecting plan %s level %s", e.plan.Name, levelName)

	bm := &blip.Metrics{
		Plan:      e.plan.Name,
		Level:     levelName,
		MonitorId: e.monitorId,
		Values:    make(map[string][]blip.MetricValue, len(mc)),
	}
	mux := &sync.Mutex{} // serialize writes to Values ^

	var wg sync.WaitGroup
	bm.Begin = time.Now()
	for i := range mc {
		<-e.sem
		wg.Add(1)
		go func(mc blip.Collector) {
			defer func() {
				e.sem <- true
				wg.Done()

				// Print panic everywhere: log, event, and collector status
				if r := recover(); r != nil {
					b := make([]byte, 4096)
					n := runtime.Stack(b, false)
					perr := fmt.Errorf("PANIC: monitor ID %s: %v\n%s", e.monitorId, r, string(b[0:n]))
					log.Println(perr)
					e.event.Sendf(event.COLLECTOR_PANIC, perr.Error())
					e.mcMux.Lock()
					e.mcList[mc.Domain()].err = perr
					e.mcMux.Unlock()
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

			// Update collector status: set new error or clear old error
			e.mcMux.Lock()
			e.mcList[mc.Domain()].err = err
			e.mcMux.Unlock()

			// Return early if context canceled or timeout; discard metrics
			// because it's unlikely they're valid on context cancel/timeout
			if err != nil && ctx.Err() != nil {
				return
			}

			// Save collector metric values
			mux.Lock()
			bm.Values[mc.Domain()] = vals
			mux.Unlock()
		}(mc[i])
	}
	wg.Wait()
	bm.End = time.Now()

	status.Monitor(e.monitorId, "engine", "idle") // @todo don't overwrite concurrent calls

	e.statusMux.Lock()
	e.status.CollectOK += 1
	e.statusMux.Unlock()

	return bm, nil
}
