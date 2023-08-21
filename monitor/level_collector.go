// Copyright 2023 Block, Inc.

package monitor

import (
	"context"
	"database/sql"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"

	"github.com/cashapp/blip"
	"github.com/cashapp/blip/event"
	"github.com/cashapp/blip/plan"
	"github.com/cashapp/blip/status"
)

// LevelCollector (LCO) executes the current plan to collect metrics.
// It's also responsible for changing the plan when called by the PlanChanger.
//
// The term "collector" is a little misleading because the LCO doesn't collect
// metrics, but it is the first step in the metrics collection process, which
// looks roughly like: LCO -> Engine -> metric collectors -> MySQL.
// In Run, the LCO checks every 1s for the highest level in the plan to collect.
// For example, after 5s it'll collect levels with a frequency divisible by 5s.
// See https://cashapp.github.io/blip/v1.0/intro/plans.
//
// Metrics from MySQL flow back to the LCO as blip.Metrics, which the LCO
// passes to blip.Plugin.TransformMetrics if specified, then to all sinks
// specified for the monitor.
type LevelCollector interface {
	// Run runs the collector to collect metrics; it's a blocking call.
	Run(stopChan, doneChan chan struct{}) error

	// ChangePlan changes the plan; it's called by the PlanChanger.
	ChangePlan(newState, newPlanName string) error

	// Pause pauses metrics collection until ChangePlan is called.
	Pause()
}

var _ LevelCollector = &lco{}

// lco is the implementation of LevelCollector.
type lco struct {
	cfg              blip.ConfigMonitor
	planLoader       *plan.Loader
	sinks            []blip.Sink
	transformMetrics func(*blip.Metrics) error
	// --
	monitorId string
	engine    *Engine

	stateMux *sync.Mutex
	state    string
	plan     blip.Plan
	levels   []plan.SortedLevel
	paused   bool

	changeMux            *sync.Mutex
	changePlanCancelFunc context.CancelFunc
	changePlanDoneChan   chan struct{}
	stopped              bool

	event event.MonitorReceiver
}

type LevelCollectorArgs struct {
	Config           blip.ConfigMonitor
	DB               *sql.DB
	PlanLoader       *plan.Loader
	Sinks            []blip.Sink
	TransformMetrics func(*blip.Metrics) error
}

func NewLevelCollector(args LevelCollectorArgs) *lco {
	return &lco{
		cfg:              args.Config,
		planLoader:       args.PlanLoader,
		sinks:            args.Sinks,
		transformMetrics: args.TransformMetrics,
		// --
		monitorId: args.Config.MonitorId,
		engine:    NewEngine(args.Config, args.DB),
		stateMux:  &sync.Mutex{},
		paused:    true,
		changeMux: &sync.Mutex{},
		event:     event.MonitorReceiver{MonitorId: args.Config.MonitorId},
	}
}

// TickerDuration sets the internal ticker duration for testing. This is only
// called for testing; do not called outside testing.
func TickerDuration(d time.Duration) {
	tickerDuration = d
}

var tickerDuration = 1 * time.Second // used for testing

// maxCollectors is the maximum number of parallel collect() goroutines.
// Code comment block just below for variable sem.
const maxCollectors = 2

func (c *lco) Run(stopChan, doneChan chan struct{}) error {
	defer close(doneChan)

	// Metrics are collected async so that this main loop does not block.
	// Normally, collecting metrics should be synchronous: every 1s, take
	// about 100-300 milliseconds get metrics and done--plenty of time
	// before the next whole second tick. But in the real world, there are
	// always blips (yes, that's partly where Blip gets its name from):
	// MySQL takes 1 or 2 seconds--or longer--to return metrics, especially
	// for "big" domains like size.table that might need to iterator over
	// hundreds or thousands of tables. Consequently, we collect metrics
	// asynchronously in multiple goroutines. By default, 2 goroutines
	// (maxCollectors) should be more than sufficient. If not, there's probably
	// an underlyiny problem that needs to be fixed.
	sem := make(chan bool, maxCollectors)
	for i := 0; i < maxCollectors; i++ {
		sem <- true
	}

	// -----------------------------------------------------------------------
	// LCO main loop: collect metrics on whole second ticks

	status.Monitor(c.monitorId, status.LEVEL_COLLECTOR, "started at %s (paused until plan change)", blip.FormatTime(time.Now()))

	s := -1 // number of whole second ticks
	ticker := time.NewTicker(tickerDuration)
	defer ticker.Stop()
	for range ticker.C {
		s = s + 1 // count seconds

		// Was Stop called?
		select {
		case <-stopChan: // yes, return immediately
			// Stop changePlan goroutine (if any) and prevent new ones in the
			// pathological case that the LCH calls ChangePlan while the LCO
			// is terminating
			c.changeMux.Lock()
			defer c.changeMux.Unlock()
			c.stopped = true // make ChangePlan do nothing
			select {
			case <-c.changePlanDoneChan:
				c.changePlanCancelFunc() // stop --> changePlan goroutine
				<-c.changePlanDoneChan   // wait for changePlan goroutine
			default:
			}

			// Stop the engine and clean up metrics
			c.engine.Stop()
			return nil
		default: // no
		}

		c.stateMux.Lock() // -- LOCK --
		if c.paused {
			s = -1              // reset count on pause
			c.stateMux.Unlock() // -- Unlock
			continue
		}

		// Determine lowest level to collect
		level := -1
		for i := range c.levels {
			if s%c.levels[i].Freq == 0 {
				level = i
			}
		}
		if level == -1 {
			c.stateMux.Unlock() // -- Unlock
			continue            // no metrics to collect at this frequency
		}

		// Collect metrics at this level
		select {
		case <-sem:
			go func(levelName string) {
				defer func() {
					sem <- true
					if err := recover(); err != nil { // catch panic in collectors, TransformMetrics, and sinks
						b := make([]byte, 4096)
						n := runtime.Stack(b, false)
						c.event.Sendf(event.LPC_PANIC, "PANIC: %s: %s\n%s", c.monitorId, err, string(b[0:n]))
					}
				}()
				c.collect(levelName)
			}(c.levels[level].Name)
			status.Monitor(c.monitorId, status.LEVEL_COLLECTOR, "idle; started collecting %s/%s at %s", c.plan.Name, c.levels[level].Name, blip.FormatTime(time.Now()))
		default:
			// all collectors blocked
			errMsg := fmt.Errorf("cannot callect %s/%s: %d of %d collectors still running",
				c.plan.Name, c.levels[level].Name, maxCollectors, maxCollectors)
			c.event.Sendf(event.LPC_BLOCKED, errMsg.Error())
			status.Monitor(c.monitorId, status.LEVEL_COLLECTOR, "blocked: %s", errMsg)
		}

		c.stateMux.Unlock() // -- UNLOCK --
	}
	return nil
}

func (c *lco) collect(levelName string) {
	collectNo := status.MonitorMulti(c.monitorId, status.LEVEL_COLLECT, "%s/%s: collecting", c.plan.Name, levelName)
	defer status.RemoveComponent(c.monitorId, collectNo)

	t0 := time.Now()

	// **************************************************************
	// COLLECT METRICS
	//
	// Collect all metrics at this level. This is where metrics
	// collection begins. Then Engine.Collect does the real work.
	metrics, err := c.engine.Collect(context.Background(), levelName)
	// **************************************************************
	if err != nil {
		status.Monitor(c.monitorId, "error:"+collectNo, err.Error())
		c.event.Errorf(event.ENGINE_COLLECT_ERROR, err.Error())
	} else {
		status.RemoveComponent(c.monitorId, "error:"+collectNo)
	}

	// Return early unless there are metrics
	if metrics == nil {
		status.Monitor(c.monitorId, status.LEVEL_COLLECT, "last collected %s/%s at %s in %s, but engine returned no metrics", c.plan.Name, levelName, blip.FormatTime(t0), time.Since(t0))
		blip.Debug("%s: level %s: nil metrics", c.monitorId, levelName)
		return
	}

	// Call user-defined TransformMetrics plugin, if set
	if c.transformMetrics != nil {
		blip.Debug("%s: level %s: transform metrics", c.monitorId, levelName)
		status.Monitor(c.monitorId, collectNo, "%s/%s: TransformMetrics", c.plan.Name, levelName)
		c.transformMetrics(metrics)
	}

	// Send metrics to all sinks configured for this monitor. This is done
	// sync because sinks are supposed to be fast or async _and_ have their
	// timeout, which is why we pass context.Background() here. Also, this
	// func runs in parallel (up to maxCollectors), so if a sink is slow,
	// that might be ok.
	blip.Debug("%s: level %s: sending metrics", c.monitorId, levelName)
	for i := range c.sinks {
		sinkName := c.sinks[i].Name()
		status.Monitor(c.monitorId, collectNo, "%s/%s: sending to %s", c.plan.Name, levelName, sinkName)
		err := c.sinks[i].Send(context.Background(), metrics)
		if err != nil {
			c.event.Errorf(event.SINK_SEND_ERROR, "%s :%s", sinkName, err) // log by default
			status.Monitor(c.monitorId, "error:"+sinkName, err.Error())
		} else {
			status.RemoveComponent(c.monitorId, "error:"+sinkName)
		}
	}

	status.Monitor(c.monitorId, status.LEVEL_COLLECT, "last collected and sent metrics for %s/%s at %s in %s", c.plan.Name, levelName, blip.FormatTime(t0), time.Since(t0))
	blip.Debug("%s: level %s: done in %s", c.monitorId, levelName, time.Since(t0))
}

// ChangePlan changes the metrics collect plan based on database state.
// It loads the plan from the plan.Loader, then it calls Engine.Prepare.
// This is the only time and place that Engine.Prepare is called.
//
// The caller is either LevelAdjuster.CheckState or Monitor.Start. The former
// is the case when config.monitors.plans.adjust is set. In this case,
// the LevelAdjuster (LPA) periodically checks database state and calls this
// function when the database state changes. It trusts that this function
// changes the state, so the LPA does not retry the call. The latter case,
// called from Monitor.Start, happen when the LPA is not enabled, so the
// monitor sets state=active, plan=<default>; then it trusts this function
// to keep retrying.
//
// ChangePlan is safe to call by multiple goroutines because it serializes
// plan changes, and the last plan wins. For example, if plan change 1 is in
// progress, plan change 2 cancels it and is applied. If plan change 3 happens
// while plan change 2 is in progress, then 3 cancels 2 and 3 is applied.
// Since the LPA is the only periodic caller and it has delays (so plans don't
// change too quickly), this shouldn't happen.
//
// Currently, the only way this function fails is if the plan cannot be loaded.
// That shouldn't happen because plans are loaded on startup, but it might
// happen in the future if Blip adds support for reloading plans via the API.
// Then, plans and config.monitors.*.plans.adjust might become out of sync.
// In this hypothetical error case, the plan change fails but the current plan
// continues to work.
func (c *lco) ChangePlan(newState, newPlanName string) error {
	// Serialize access to this func
	c.changeMux.Lock()
	defer c.changeMux.Unlock()

	if c.stopped { // Run stopped?
		return nil
	}

	// Check if changePlan goroutine from previous call is running
	select {
	case <-c.changePlanDoneChan:
	default:
		if c.changePlanCancelFunc != nil {
			blip.Debug("cancel previous changePlan")
			c.changePlanCancelFunc() // stop --> changePlan goroutine
			<-c.changePlanDoneChan   // wait for changePlan goroutine
		}
	}

	blip.Debug("start new changePlan: %s %s", newState, newPlanName)
	ctx, cancel := context.WithCancel(context.Background())
	c.changePlanCancelFunc = cancel
	c.changePlanDoneChan = make(chan struct{})

	// Don't block caller. If state changes again, LPA will call this
	// func again, in which case the code above will cancel the current
	// changePlan goroutine (if it's still running) and re-change/re-prepare
	// the plan for the latest state.
	go c.changePlan(ctx, c.changePlanDoneChan, newState, newPlanName)

	return nil
}

// changePlan is a gorountine run by ChangePlan It's potentially long-running
// because it waits for Engine.Prepare. If that function returns an error
// (e.g. MySQL is offline), then this function retires forever, or until canceled
// by either another call to ChangePlan or Run is stopped (LCO is terminated).
//
// Never all this function directly; it's only called via ChangePlan, which
// serializes access and guarantees only one changePlan goroutine at a time.
func (c *lco) changePlan(ctx context.Context, doneChan chan struct{}, newState, newPlanName string) {
	defer close(doneChan)

	c.stateMux.Lock()
	oldState := c.state
	oldPlanName := c.plan.Name
	c.stateMux.Unlock()
	change := fmt.Sprintf("state:%s plan:%s -> state:%s plan:%s", oldState, oldPlanName, newState, newPlanName)
	c.event.Sendf(event.CHANGE_PLAN, change)

	// Load new plan from plan loader, which contains all plans. Try forever because
	// that's what this func/gouroutine does: try forever (caller's expect that).
	// This shouldn't fail given that plans were already loaded and validated on startup,
	// but maybe plans reloaded after startup and something broke. User can fix by
	// reloading plans again.
	var newPlan blip.Plan
	var err error
	for {
		status.Monitor(c.monitorId, status.LEVEL_CHANGE_PLAN, "loading new plan %s (state %s)", newPlanName, newState)
		newPlan, err = c.planLoader.Plan(c.engine.MonitorId(), newPlanName, c.engine.DB())
		if err == nil {
			break // success
		}

		errMsg := fmt.Sprintf("%s: error loading new plan %s: %s (retrying)", change, newPlanName, err)
		status.Monitor(c.monitorId, status.LEVEL_CHANGE_PLAN, errMsg)
		c.event.Sendf(event.CHANGE_PLAN_ERROR, errMsg)
		time.Sleep(2 * time.Second)
	}

	change = fmt.Sprintf("state:%s plan:%s -> state:%s plan:%s", oldState, oldPlanName, newState, newPlan.Name)

	newPlan.MonitorId = c.monitorId
	newPlan.InterpolateEnvVars()
	newPlan.InterpolateMonitor(&c.cfg)

	// Convert plan levels to sorted levels for efficient level calculation in Run;
	// see code comments on sortedLevels.
	levels := plan.Sort(&newPlan)

	// ----------------------------------------------------------------------
	// Prepare the (new) plan
	//
	// This is two-phase commit:
	//   0. LCO: pause Run loop
	//   1. Engine: commit new plan
	//   2. LCO: commit new plan
	//   3. LCO: resume Run loop
	// Below in call c.engine.Prepare(ctx, newPlan, c.Pause, after), Prepare
	// does its work and, if successful, calls c.Pause, which is step 0;
	// then Prepare does step 1, which won't be collected yet because it
	// just paused LCO.Run which drives metrics collection; then Prepare calls
	// the after func/calleck defined below, which is step 2 and signals to
	// this func that we commit the new plan and resume Run (step 3) to begin
	// collecting that plan.

	after := func() {
		c.stateMux.Lock() // -- X lock --
		c.state = newState
		c.plan = newPlan
		c.levels = levels

		// Changing state/plan always resumes (if paused); in fact, it's the
		// only way to resume after Pause is called
		c.paused = false
		status.Monitor(c.monitorId, status.LEVEL_STATE, newState)
		status.Monitor(c.monitorId, status.LEVEL_PLAN, newPlan.Name)
		blip.Debug("%s: resume", c.monitorId)

		c.stateMux.Unlock() // -- X unlock --
	}

	// Try forever, or until context is cancelled, because it could be that MySQL is
	// temporarily offline. In the real world, this is not uncommon: Blip might be
	// started before MySQL, for example. We're running in a goroutine from ChangePlan
	// that already returned to its caller, so we're not blocking anything here.
	// More importantly, as documented in several place: this is _the code_ that
	// all other code relies on to try "forever" because a plan must be prepared
	// before anything can be collected.
	status.Monitor(c.monitorId, status.LEVEL_CHANGE_PLAN, "preparing new plan %s (state %s)", newPlan.Name, newState)
	retry := backoff.NewExponentialBackOff()
	retry.MaxElapsedTime = 0
	for {
		// ctx controls the goroutine, which might run "forever" if plans don't
		// change. ctxPrep is a timeout for Prepare to ensure that it does not
		// run try "forever". If preparing takes too long, there's probably some
		// issue, so we need to sleep and retry.
		ctxPrep, cancelPrep := context.WithTimeout(ctx, 10*time.Second)
		err := c.engine.Prepare(ctxPrep, newPlan, c.Pause, after)
		cancelPrep()
		if err == nil {
			break // success
		}
		if ctx.Err() != nil {
			blip.Debug("changePlan canceled")
			return // changePlan goroutine has been cancelled
		}
		status.Monitor(c.monitorId, status.LEVEL_CHANGE_PLAN, "%s: error preparing new plan %s: %s (retrying)", change, newPlan.Name, err)
		time.Sleep(retry.NextBackOff())
	}

	status.RemoveComponent(c.monitorId, status.LEVEL_CHANGE_PLAN)
	c.event.Sendf(event.CHANGE_PLAN_SUCCESS, change)
}

// Pause pauses metrics collection until ChangePlan is called. Run still runs,
// but it doesn't collect when paused. The only way to resume after pausing is
// to call ChangePlan again.
func (c *lco) Pause() {
	c.stateMux.Lock()
	c.paused = true
	status.Monitor(c.monitorId, status.LEVEL_COLLECTOR, "paused at %s", blip.FormatTime(time.Now()))
	c.event.Send(event.LPC_PAUSED)
	c.stateMux.Unlock()
}
