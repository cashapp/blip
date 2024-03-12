// Copyright 2024 Block, Inc.

package monitor

import (
	"context"
	"database/sql"
	"fmt"
	"runtime"
	"sort"
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

// collection is the return values from one Collector.Collect call.
type collection struct {
	Interval uint
	Level    string
	Domain   string
	Values   []blip.MetricValue
	Err      error
	Runtime  time.Duration
}

// Engine does the real work: collect metrics.
type Engine struct {
	cfg       blip.ConfigMonitor
	db        *sql.DB
	monitorId string
	// --
	event event.MonitorReceiver
	*sync.Mutex
	plan           blip.Plan
	collectors     map[string]*clutch   // keyed on domain
	collectAt      map[string][]*clutch // keyed on level, sorted ascending by CMR
	checkAt        map[string][]*clutch // keyed on level
	collectionChan chan collection
}

func NewEngine(cfg blip.ConfigMonitor, db *sql.DB) *Engine {
	return &Engine{
		cfg:       cfg,
		db:        db,
		monitorId: cfg.MonitorId,
		// --
		event:          event.MonitorReceiver{MonitorId: cfg.MonitorId},
		Mutex:          &sync.Mutex{},
		collectors:     map[string]*clutch{},
		collectAt:      map[string][]*clutch{},
		checkAt:        map[string][]*clutch{},
		collectionChan: make(chan collection, len(metrics.List())*2),
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

	// Find minimum intervals (freq) for plan and each domain.
	minFreq, domainFreq := plan.Freq()

	// Create and prepare metric collectors for every level. Return on error
	// because the error might be fatal, e.g. something misconfigured and the
	// plan cannot work.
	collectors := map[string]*clutch{}  // keyed on domain
	collectAt := map[string][]*clutch{} // keyed on level
	domainsAt := map[string][]string{}  // keyed on level
	allDomains := map[string]bool{}     // keyed on level
	for levelName, level := range plan.Levels {
		domains := make([]string, 0, len(level.Collect))
		domainsAt[levelName] = make([]string, 0, len(level.Collect))

		for domain := range level.Collect {
			// At this level, collect this domain (sorted by domain freq below)
			domains = append(domains, domain)
			allDomains[domain] = true

			// Make collector first time it's seen (they're unique in a plan)
			if _, ok := collectors[domain]; ok {
				continue // already seen
			}
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

			// Wrap collector in a clutch that provides the connection between
			// engine and collector: engaged during Engine.Collect; disengaged
			// if EMR (engine max runtime) expires but CMR (collector max runtime)
			// allows the collector to keep running.
			collectors[domain] = &clutch{ // new clutch
				c:              c,
				cleanup:        cleanup,
				cmr:            blip.TimeLimit(domainFreq[domain], 0.2, 2000), // collector interval minus 20% (max 2s)
				collectionChan: e.collectionChan,
				monitorId:      e.monitorId,
				event:          e.event,
				Mutex:          &sync.Mutex{},
			}
		}

		// Sort domains collected at this level by freq (asc)
		sort.Slice(domains, func(i, j int) bool { return domainFreq[domains[i]] < domainFreq[domains[j]] })
		blip.Debug("domain priority at %s: %v", levelName, domains)
		collectAt[levelName] = make([]*clutch, len(domains))
		for i := range domains {
			collectAt[levelName][i] = collectors[domains[i]]
		}

		domainsAt[levelName] = domains // used outside loop below
	}

	// Inverse: at each level, which domains are NOT run and instead checked
	// for past long-running metrics
	for levelName, domains := range domainsAt {
		// Find domains NOT collected at this level. During Collect at this level,
		// these domains will be check for pending metrics to flush. That's why we
		// exclude domains collected at the level: collecting will flush pending
		// metrics, too.
		check := []*clutch{}
		included := []string{}
		for domain := range allDomains {
			// Is domain excluded because it's collect at this level, or a min freq domain?
			atLevel := false
			for _, excludedDomain := range domains {
				if domain == excludedDomain || domainFreq[domain] == minFreq {
					atLevel = true
					break
				}
			}
			if !atLevel { // domain NOT collected at this level, so check/flush at this level
				check = append(check, collectors[domain])
				included = append(included, domain)
			}
		}
		e.checkAt[levelName] = check // all domains to check at this level (none collected at this level)
		blip.Debug("check pending flush at %s: %v", levelName, included)
	}

	// Successfully prepared the plan
	status.Monitor(e.monitorId, status.ENGINE_PREPARE, "%s: level-collector before callback", plan.Name)
	before() // notify caller (lco.changePlan) that we're about to swap the plan

	status.Monitor(e.monitorId, status.ENGINE_PREPARE, "%s: finalize", plan.Name)
	e.Lock() // LOCK plan -------------------------------------------

	// Stop current collectors and call their cleanup func, if any. For example,
	// the repl collector uses a cleanup func to stop its heartbeat.BlipReader goroutine.
	e.stopCollectors()

	e.collectors = collectors // new mcs
	e.plan = plan             // new plan
	e.collectAt = collectAt   // new levels

	e.Unlock() // UNLOCK plan ---------------------------------------

	status.Monitor(e.monitorId, status.ENGINE_PLAN, plan.Name)
	e.event.Sendf(event.ENGINE_PREPARE_SUCCESS, plan.Name)

	status.Monitor(e.monitorId, status.ENGINE_PREPARE, "%s: level-collector after callback", plan.Name)
	after() // notify caller (lco.changePlan) that we have swapped the plan

	return nil
}

func (e *Engine) Collect(emrCtx context.Context, interval uint, levelName string, startTime time.Time) ([]*blip.Metrics, error) {
	e.Lock()
	defer e.Unlock()

	// Collection ID for status and logging, paired with monitor ID like "db1.local: myPlan/kpi/5"
	coId := fmt.Sprintf("%s/%s/%d", e.plan.Name, levelName, interval)

	// All metric collectors at this level
	domains := e.collectAt[levelName]
	if domains == nil {
		blip.Debug("%s: %s: no domains, ignoring", e.monitorId, coId)
		return nil, nil
	}
	blip.Debug("%s: %s: collect", e.monitorId, coId)
	status.Monitor(e.monitorId, status.ENGINE_COLLECT, coId+": collecting")

	// Collect metrics for each domain in parallel (limit: CollectParallel)
	sem := make(chan bool, CollectParallel) // semaphore for CollectParallel
	for i := 0; i < CollectParallel; i++ {
		sem <- true
	}

	// Collect domains at this level
	running := map[string]bool{}
	begin := time.Now()
	for _, cl := range domains {
		select {
		case <-sem:
			go cl.collect(interval, levelName, startTime, sem)
			running[cl.c.Domain()] = true
		case <-emrCtx.Done():
			blip.Debug("EMR timeout starting collectors")
			break
		}
	}

	// Flush metrics from domains NOT started at this level
	for _, cl := range e.checkAt[levelName] {
		cl.Lock()
		if cl.pending {
			cl.flush(false)
		}
		cl.Unlock()
	}

	// Wait for all collectors to finish, then record end time
	metrics := []*blip.Metrics{
		{
			Plan:      e.plan.Name,
			Level:     levelName,
			Interval:  interval,
			MonitorId: e.monitorId,
			Values:    map[string][]blip.MetricValue{},
			Begin:     begin,
		},
	}
	errs := map[string]error{}
	nValues := 0
SWEEP:
	for len(running) > 0 {
		status.Monitor(e.monitorId, status.ENGINE_COLLECT, "%s: receiving metrics, %d collectors running", coId, len(running))
		select {
		case c := <-e.collectionChan:
			if c.Interval == interval {
				delete(running, c.Domain)
				if n := len(c.Values); n > 0 {
					metrics[0].Values[c.Domain] = c.Values
					nValues += n
				}
				if c.Err != nil {
					errs[c.Domain] = c.Err
				}
			} else {
				// Old metrics
			}
		case <-emrCtx.Done(): // engine runtime max
			blip.Debug("EMR timeout receiving collections")
			break SWEEP
		}
	}
	metrics[0].End = time.Now()

	// Process collector errors, if any
	status.Monitor(e.monitorId, status.ENGINE_COLLECT, coId+": logging errors")
	errCount := 0
	for domain, err := range errs {
		// Update MonitorEngineStatus: set new error or clear old error
		if err != nil {
			errCount += 1
			errMsg := fmt.Sprintf("error collecting %s/%s/%s: %s", e.plan.Name, levelName, domain, err)
			status.Monitor(e.monitorId, "error:"+domain, "at %s: %s: %s", metrics[0].Begin, coId, errMsg)
			e.event.Errorf(event.COLLECTOR_ERROR, errMsg) // log by default
		} else {
			status.RemoveComponent(e.monitorId, "error:"+domain)
		}
	}

	status.Monitor(e.monitorId, status.ENGINE_COLLECT, "%s: done: %d started, %d domains %d values collected, %s runtime, %d error",
		coId, len(domains), len(domains)-len(running), nValues, metrics[0].End.Sub(begin), errCount)

	// Total success? Yes if no errors.
	if errCount == 0 {
		return metrics, nil
	}

	// Partial success? Yes if there are some metrics values.
	if nValues > 0 {
		return metrics, fmt.Errorf("%s: partial success: %d metrics collected, %d errors", coId, nValues, errCount)
	}

	// Errors and zero metrics: all collectors failed
	return metrics, fmt.Errorf("%s: failed: zero metrics collected, %d errors", coId, errCount)
}

// Stop the engine and cleanup any metrics associated with it.
// TODO: There is a possible race condition when this is called. Since
// Engine.Collect is called as a go-routine, we could have an invocation
// of the function block waiting for Engine.Stop to unlock
// after which Collect would run after cleanup has been called.
// This could result in a panic, though that should be caught and logged.
// Since the monitor is stopping anyway this isn't a huge issue.
func (e *Engine) Stop() {
	blip.Debug("Stopping engine...")
	e.Lock()
	defer e.Unlock()
	e.stopCollectors()
	// Prevent Collect from running in case it's blocked on mutex
	for level := range e.collectAt {
		e.collectAt[level] = nil
	}
}

func (e *Engine) stopCollectors() {
	/* -- CALLER MUST LOCK Engine -- */
	for _, cl := range e.collectors {
		cl.Lock()
		if !cl.running {
			blip.Debug("%s: %s not running", e.monitorId, cl.c.Domain())
			continue
		}
		blip.Debug("%s: %s stopping", e.monitorId, cl.c.Domain())
		cl.cancel()
		if cl.cleanup != nil {
			blip.Debug("%s: %s cleanup", e.monitorId, cl.c.Domain())
			cl.cleanup()
		}
		cl.Unlock()
	}
}

// --------------------------------------------------------------------------

// amc represents one active metric collector and its cleanup func (if any).
// There's only one mc per domain, even if the domain is used at multiple levels
// (because the mc prepares itself for each level). When plans change, we start
// over (we don't reset/reuse mcs): discard all old mc (calling their cleanup funcs)
// then create a new list of mcs.
type clutch struct {
	c              blip.Collector
	cleanup        func()            // from c.Prepare (optional)
	cmr            time.Duration     // collector max runtime (CMR)
	collectionChan chan<- collection // flush vals/err to
	monitorId      string            // for logging
	event          event.MonitorReceiver
	*sync.Mutex

	// When running:
	ctx       context.Context // context.WithTimeout(cmr)
	cancel    context.CancelFunc
	interval  uint               // invoked for interval
	level     string             // invoked at level
	running   bool               // collect() running
	bg        bool               // true if c.Collect returns ErrMore
	pending   bool               // vals ready to flush
	vals      []blip.MetricValue // pending values from c
	err       error              // last error from c
	startTime time.Time          // when collect(true) called
	stopTime  time.Time          // when collect(true) called
}

func (cl *clutch) collect(interval uint, level string, startTime time.Time, sem chan bool) {
	blip.Debug("[[[ %+v ]]]\n", cl)
	cl.Lock() // ___LOCK___

	if cl.running {
		// Collector fault: it didn't terminate itself at CMR
		blip.Debug("~~~~~~~~~~~~~~~~~~~~~ collector fault ~~~~~~~~~~~~~~~~~~~~~~~~~~~~")
		// @todo
	}

	// Metrics from previous run, do first so cl.internal level don't overwrite
	if cl.pending {
		cl.flush(false) // false -> don't override bg stop time (see defer below)
	}

	// Start new collection. DO NOT defer and set cl.running = false before
	// here else the period flush block above ^ will set cl.running = false
	// when it returns, which is not the case: this func is only done running
	// when the code below returns.
	blip.Debug("<< CL %d/%s", cl.interval, cl.level)
	cl.ctx, cl.cancel = context.WithDeadline(context.Background(), startTime.Add(cl.cmr))
	cl.startTime = time.Now()
	cl.running = true
	cl.interval = interval
	cl.level = level
	cl.Unlock() // ___unlock___

	defer func() {
		cl.Lock()
		cl.running = false
		cl.cancel() // cl.ctx
		if cl.bg {
			cl.stopTime = time.Now() // bg stop time
		}
		cl.Unlock()

		// Handle collector panic
		if r := recover(); r != nil {
			if !cl.bg { // foreground
				select {
				case sem <- true:
				default:
				}
			}

			b := make([]byte, 4096)
			n := runtime.Stack(b, false)
			perr := fmt.Errorf("PANIC: monitor ID %s: %v\n%s", cl.monitorId, r, string(b[0:n]))
			cl.event.Errorf(event.COLLECTOR_PANIC, perr.Error())
		}

	}()

	// ----------------------------------------------------------------------
	// FAST PATH, NORMAL CASE: collect once quickly, flush metrics back to
	// engine for reporting, and done.
	vals, err := cl.c.Collect(cl.ctx, cl.level) // foreground collect
	cl.Lock()
	cl.vals = vals
	cl.err = err
	cl.flush(err != blip.ErrMore)
	cl.Unlock()
	sem <- true // always unblock next collector
	if err != blip.ErrMore {
		return
	}

	// ----------------------------------------------------------------------

	// Special case: blip.ErrMore == long-running collector, keep running and
	// saving metrics in cl.vals until collector stops returning blip.ErrMore.
	// The engine will call cl.flush every interval if cl.pending is true.
	cl.Lock()
	cl.bg = true
	cl.Unlock()
	for err == blip.ErrMore && err != context.Canceled && err != context.DeadlineExceeded {
		vals, err = cl.c.Collect(nil, "") // background collect
		cl.Lock()
		if len(vals) > 0 {
			cl.pending = true
			cl.vals = append(cl.vals, vals...)
		}
		cl.err = err
		cl.Unlock()
	}
}

func (cl *clutch) flush(done bool) {
	/* -- CALLER MUST LOCK clutch -- */
	c := collection{
		Interval: cl.interval,
		Level:    cl.level,
		Domain:   cl.c.Domain(),
		Values:   cl.vals,
		Err:      cl.err,
	}
	if done {
		cl.stopTime = time.Now()
	}
	if !cl.stopTime.IsZero() {
		c.Runtime = cl.stopTime.Sub(cl.startTime)
	}

	// Flush metrics back to engine BEFORE resetting them below
	select {
	case cl.collectionChan <- c:
	default:
		blip.Debug("flush blocked: %v", c)
		// @todo error? panic?
	}

	cl.pending = false
	cl.vals = []blip.MetricValue{}
	cl.err = nil
}
