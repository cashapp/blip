package monitor

import (
	"context"
	"database/sql"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/square/blip"
	"github.com/square/blip/event"
	"github.com/square/blip/metrics"
	"github.com/square/blip/proto"
	"github.com/square/blip/status"
)

// Engine does the real work: collect metrics.
type Engine struct {
	monitorId string
	db        *sql.DB
	// --
	*sync.RWMutex
	mcList    map[string]blip.Collector // keyed on domain
	mcError   map[string]error
	atLevel   map[string][]blip.Collector // keyed on level
	plan      blip.Plan
	event     event.MonitorSink
	sem       chan bool
	semSize   int
	status    proto.MonitorEngineStatus
	errMux    *sync.Mutex
	statusMux *sync.Mutex
}

func NewEngine(monitorId string, db *sql.DB) *Engine {
	sem := make(chan bool, 2)
	semSize := 2
	for i := 0; i < semSize; i++ {
		sem <- true
	}

	return &Engine{
		monitorId: monitorId,
		db:        db,
		// --
		atLevel:   map[string][]blip.Collector{},
		mcList:    map[string]blip.Collector{},
		mcError:   map[string]error{},
		RWMutex:   &sync.RWMutex{},
		event:     event.MonitorSink{MonitorId: monitorId},
		sem:       sem,
		semSize:   semSize,
		status:    proto.MonitorEngineStatus{},
		errMux:    &sync.Mutex{},
		statusMux: &sync.Mutex{},
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

	e.errMux.Lock()
	errs := map[string]string{}
	for k, v := range e.mcError {
		if v == nil {
			continue
		}
		errs[k] = v.Error()
	}
	if len(errs) > 0 {
		cp.CollectorErrors = errs
	}
	e.errMux.Unlock()

	return cp
}

func (e *Engine) setErr(err error) {
	e.errMux.Lock()
	if err == nil {
		e.status.Error = ""
	} else {
		e.status.Error = err.Error()
	}
	e.errMux.Unlock()
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
	e.event.Sendf(event.MONITOR_PREPARE_PLAN, plan.Name)

	status.Monitor(e.monitorId, "engine-prepare", "preparing plan %s", plan.Name)
	var lerr error
	defer func() {
		if lerr == nil {
			status.RemoveComponent(e.monitorId, "engine-prepare")
		}
		e.setErr(lerr) // report last error, if any
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
	mcList := map[string]blip.Collector{} // keyed on domain
	atLevel := map[string][]blip.Collector{}
	for levelName, level := range plan.Levels {
		for domain, _ := range level.Collect {

			status.Monitor(e.monitorId, "engine-prepare", "preparing plan %s level %s collector %s",
				plan.Name, levelName, domain)

			// Make collector if needed
			mc, ok := mcList[domain]
			if !ok {
				mc, lerr = metrics.Make(
					domain,
					blip.CollectorFactoryArgs{
						MonitorId: e.monitorId,
						DB:        e.db,
					},
				)
				if lerr != nil {
					blip.Debug(lerr.Error())
					return lerr
				}
				mcList[domain] = mc
			}

			if err := mc.Prepare(ctx, plan); err != nil {
				if ctx.Err() != nil {
					lerr = fmt.Errorf("cancelled while preparing plan")
					return nil // cancellation is not an error
				}
				lerr = fmt.Errorf("prepare collector %s: %s", domain, err)
				return lerr
			}

			// At this level, collect from this domain
			atLevel[levelName] = append(atLevel[levelName], mc)

			// OK to keep working?
			if ctx.Err() != nil {
				lerr = fmt.Errorf("cancelled while preparing plan")
				return nil // cancellation is not an error
			}
		}
	}

	status.Monitor(e.monitorId, "engine-prepare", "finalizing plan %s", plan.Name)

	before() // notify caller (LPC.changePlan) that we're about to swap the plan

	e.Lock() // block Collect
	e.mcList = mcList
	e.atLevel = atLevel
	e.mcError = map[string]error{}
	e.plan = plan
	e.Unlock() // allow Collect (new plan)

	e.statusMux.Lock()
	e.status.Plan = plan.Name
	e.statusMux.Unlock()

	blip.Debug("changed plan to %s", plan.Name)

	after() // notify caller (LPC.changePlan) that we have swapped the plan

	return nil
}

func (e *Engine) Collect(ctx context.Context, levelName string) (*blip.Metrics, error) {
	//
	// *** This func can run concurrently! ***
	//

	// Lock while collecting so Prepare cannot change plan while using it
	e.RLock()
	defer e.RUnlock()

	mc := e.atLevel[levelName]
	if mc == nil {
		blip.Debug("%s no mc at level '%s'", e.monitorId, levelName)
		return nil, nil
	}

	status.Monitor(e.monitorId, "engine", "collecting plan %s level %s", e.plan.Name, levelName)

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
			// Panic will bubble up to Monitor.Run, which is runs the LPC
			// that called Engine.Collect
			defer func() {
				e.sem <- true
				if err := recover(); err != nil {
					b := make([]byte, 4096)
					n := runtime.Stack(b, false)
					e.errMux.Lock()
					e.mcError[mc.Domain()] = fmt.Errorf("PANIC: %s\n%s", err, string(b[0:n]))
					e.errMux.Unlock()
					status.Monitor(e.monitorId, "collector-panic", mc.Domain())
					// @todo log or even the panic err?
				}
				wg.Done()
			}()

			// **************************************************************
			// COLLECT METRICS
			//
			// Collect metrics in this domain. This is where metrics collection
			// happens: this domain-specific blip.Collector queries MySQL and
			// returns blip.Metrics at this level.
			vals, err := mc.Collect(ctx, levelName)
			// **************************************************************

			// @todo ignore err when ctx cancelled
			e.errMux.Lock()
			e.mcError[mc.Domain()] = err
			e.errMux.Unlock()

			mux.Lock()
			bm.Values[mc.Domain()] = vals
			mux.Unlock()
		}(mc[i])
	}
	wg.Wait()
	bm.End = time.Now()

	status.Monitor(e.monitorId, "engine", "idle")

	e.statusMux.Lock()
	e.status.CollectOK += 1 // @todo not thread-safe
	e.statusMux.Unlock()

	return bm, nil
}
