package monitor

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/square/blip"
	"github.com/square/blip/event"
	"github.com/square/blip/plan"
	"github.com/square/blip/status"
)

// LevelCollector collect metrics according to a plan. It doesn't collect metrics
// directly, as part of a Monitor, it calls the Engine when it's time to collect
// metrics for a certain level--based on the frequency the users specifies for
// each level. After the Engine returns metrics, the collector (or "LPC" for short)
// calls the blip.Plugin.TransformMetrics (if specified), then sends metrics to
// all sinks specififed for the monitor. Then it waits until it's time to collect
// metrics for the next level. Consequently, the LPC drives metrics collection,
// but the Engine does the actual work of collecting metrics.
type LevelCollector interface {
	// Run runs the collector to collect metrics; it's a blocking call.
	Run(stopChan, doneChan chan struct{}) error

	// ChangePlan changes the plan; it's called by an Adjuster.
	ChangePlan(newState, newPlanName string) error

	// Pause pauses metrics collection until ChangePlan is called.
	Pause() error
}

var _ LevelCollector = &collector{}

// collector is the implementation of LevelCollector.
type collector struct {
	monitorId        string
	engine           *Engine
	planLoader       *plan.Loader
	sinks            []blip.Sink
	transformMetrics func(*blip.Metrics) error
	// --
	state                string
	plan                 blip.Plan
	changing             bool
	changePlanCancelFunc context.CancelFunc
	changePlanDoneChan   chan struct{}
	changeMux            *sync.Mutex
	stateMux             *sync.Mutex
	event                event.MonitorSink
	levels               []level
	paused               bool
}

type LevelCollectorArgs struct {
	MonitorId        string
	Engine           *Engine
	PlanLoader       *plan.Loader
	Sinks            []blip.Sink
	TransformMetrics func(*blip.Metrics) error
}

func NewLevelCollector(args LevelCollectorArgs) *collector {
	return &collector{
		monitorId:        args.MonitorId,
		engine:           args.Engine,
		planLoader:       args.PlanLoader,
		sinks:            args.Sinks,
		transformMetrics: args.TransformMetrics,
		// --
		changeMux: &sync.Mutex{},
		stateMux:  &sync.Mutex{},
		event:     event.MonitorSink{MonitorId: args.MonitorId},
		paused:    true,
	}
}

func (c *collector) Run(stopChan, doneChan chan struct{}) error {
	defer close(doneChan)

	// Metrics are collected async so that this main loop does not block.
	// Normally, collecting metrics should be synchronous: every 1s, take
	// about 100-300 milliseconds get metrics and done--plenty of time
	// before the next whole second tick. But in the real world, there are
	// always blips (yes, that's partly where Blip gets its name from):
	// MySQL takes 1 or 2 seconds--or longer--to return metrics, especially
	// for "big" domains like size.data that might need to iterator over
	// hundreds or thousands of tables. Consequently, we collect metrics
	// asynchronously in two collector goroutines: a primary (1) that should
	// be all that's needed, but also a secondary/backup (2) to handle
	// real world blips. This is hard-coded because the primary should
	// be sufficient 99% of the time, and the backup is just that: a backup
	// to handle rare blips. If both are slow/blocked, then that's a genuine
	// problem to report and let the user deal with--don't hide the problem
	// with more collector goroutines.
	//
	// Do not buffer collectLevelChan: a collector goroutine must be ready
	// on send, else it means both are slow/blocked and that's a problem.
	collectLevelChan := make(chan string)         // DO NOT BUFFER
	go c.collector(1, collectLevelChan, stopChan) // primary
	go c.collector(2, collectLevelChan, stopChan) // secondary/backup

	// -----------------------------------------------------------------------
	// LPC main loop: collect metrics on whole second ticks

	s := 1 // number of whole second ticks
	level := -1
	levelName := ""

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		s = s + 1 // count seconds

		// Was Stop called?
		select {
		case <-stopChan: // yes, return immediately
			return nil
		default: // no
		}

		c.stateMux.Lock() // -- LOCK --
		if c.paused {
			c.stateMux.Unlock() // -- Unlock
			continue
		}

		// Determine lowest level to collect
		for i := range c.levels {
			if i%c.levels[i].freq == 0 {
				level = i
			}
		}
		if level == -1 {
			c.stateMux.Unlock() // -- Unlock
			continue            // no metrics to collect at this frequency
		}

		// Collect metrics at this level, unlock, and reset
		levelName = c.levels[level].name
		level = -1
		c.stateMux.Unlock() // -- UNLOCK --

		select {
		case collectLevelChan <- levelName:
		default:
			// all collectors blocked
			blip.Debug("all mon chan blocked")
		}
	}
	return nil
}

func (c *collector) collector(n int, col chan string, stopChan chan struct{}) {
	lpc := fmt.Sprintf("lpc-%d", n)
	var level string
	for {
		status.Monitor(c.monitorId, lpc, "idle")

		select {
		case level = <-col: // signal
		case <-stopChan:
			return
		}

		status.Monitor(c.monitorId, lpc, "collecting plan %s level %s", c.plan.Name, level)
		metrics, err := c.engine.Collect(context.Background(), level)
		if err != nil {
			// @todo
		}

		if c.transformMetrics != nil {
			c.transformMetrics(metrics)
		}

		status.Monitor(c.monitorId, lpc, "sending metrics for plan %s level %s", c.plan.Name, level)
		for i := range c.sinks {
			c.sinks[i].Send(context.Background(), metrics)
			// @todo error
		}
	}
}

// ChangePlan changes the level plan in the monitor based on database state.
// This func is called only by the PlanAdjuster, which calls it sequentially.
// If the state changes while changing plans, this func cancels the previous
// plan change (changePlan goroutine) and starts changing to the new state plan.
// Monitor.Prepare does not guard against concurrent calls; it relies on this
// func to call it sequentially, never concurrently. In other words, this func
// serializes calls to Monitor.Prepare.
func (c *collector) ChangePlan(newState, newPlanName string) error {
	// Serialize access to this func
	c.changeMux.Lock()
	defer c.changeMux.Unlock()

	// Check if changePlan goroutine from previous call is running
	c.stateMux.Lock()
	changing := c.changing
	if changing {
		c.stateMux.Unlock() // let changePlan goroutine return
		c.changePlanCancelFunc()
		<-c.changePlanDoneChan
		c.stateMux.Lock()
	}
	c.changing = true
	ctx, cancel := context.WithCancel(context.Background())
	c.changePlanCancelFunc = cancel
	c.changePlanDoneChan = make(chan struct{})
	c.stateMux.Unlock()

	// Don't block caller (LPA). If state changes again, LPA will call this
	// func again, in which case the code above will cancel the current
	// changePlan goroutine (if it's still running) and re-change/re-prepare
	// the plan for the latest state.
	go c.changePlan(ctx, newState, newPlanName)

	return nil
}

func (c *collector) changePlan(ctx context.Context, newState, newPlanName string) error {
	defer func() {
		c.changePlanCancelFunc()
		c.stateMux.Lock()
		close(c.changePlanDoneChan)
		c.changing = false
		c.stateMux.Unlock()
	}()

	// Load new plan from plan loader, which contains all plans
	newPlan, err := c.planLoader.Plan(c.engine.MonitorId(), newPlanName, c.engine.DB())
	if err != nil {
		return err
	}

	// Convert plan levels to sorted levels for efficient level calculation in Run;
	// see code comments belows
	levels := sortedLevels(newPlan)

	// New plan is ready. Lock to prevent conflict with Run which is accessing
	// the (current) plan every 1s. Run only locks when calculating which level
	// (by name) to collect, then it unlocks and tells the Engine to collect that
	// level.
	c.stateMux.Lock()
	c.paused = true
	c.stateMux.Unlock()

	// Have engine prepare the plan, which really makes each metrics collector
	// prepare for the plan
	if err := c.engine.Prepare(ctx, newPlan); err != nil {
		c.stateMux.Lock()
		c.paused = false // plan did NOT change; resume current plan
		c.stateMux.Unlock()
		return err
	}

	// collect new plan
	c.stateMux.Lock() // -- X lock --
	oldState := c.state
	oldPlanName := c.plan.Name
	c.state = newState
	c.plan = newPlan
	c.levels = levels
	c.paused = false
	c.stateMux.Unlock() // -- X unlock --

	c.event.Sendf(event.CHANGE_PLAN, "state:%s plan:%s -> state:%s plan:%s",
		oldState, oldPlanName, newState, newPlan.Name)

	return nil
}

// Pause pauses metrics collection until ChangePlan is called.
func (c *collector) Pause() error {
	c.stateMux.Lock()
	c.paused = true
	c.stateMux.Unlock()
	return nil
}

// ---------------------------------------------------------------------------
// Plan vs. sorted level
// ---------------------------------------------------------------------------

// level represents a sorted level created by sortedLevels below.
type level struct {
	freq int
	name string
}

// Sort levels ascending by frequency.
type byFreq []level

func (a byFreq) Len() int           { return len(a) }
func (a byFreq) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byFreq) Less(i, j int) bool { return a[i].freq < a[j].freq }

// sortedLevels returns a list of levels sorted (asc) by frequency. Sorted levels
// are used in the main Run loop: for i := range c.levels. Sorted levels are
// required because plan levels are unorded because the plan is a map. We could
// check every level in the plan, but that's wasteful. With sorted levels, we
// can precisely check which levels to collect at every 1s tick.
//
// Also, plan levels are abbreviated whereas sorted levels are complete.
// For example, a plan says "collect X every 5s, and collect Y every 10s".
// But the complete version of that is "collect X every 5s, and collect X + Y
// every 10s." See "metric inheritance" in the docs.
//
// Also, we convert duration strings from the plan level to integers for sorted
// levels in order to do modulo (%) in the main Run loop.
func sortedLevels(plan blip.Plan) []level {
	// Make a sorted level for each plan level
	levels := make([]level, len(plan.Levels))
	i := 0
	for _, l := range plan.Levels {
		d, _ := time.ParseDuration(l.Freq) // "5s" -> 5 (for freq below)
		levels[i] = level{
			name: l.Name,
			freq: int(d.Seconds()),
		}
		i++
	}

	// Sort levels by ascending frequency
	sort.Sort(byFreq(levels))
	blip.Debug("levels: %v", levels)

	// Metric inheritence: level N applies to N+(N+1)
	for i := 0; i < len(levels); i++ {
		// At level N
		rootLevel := levels[i].name
		root := plan.Levels[rootLevel]

		// Add metrics from N to all N+1
		for j := i + 1; j < len(levels); j++ {
			leafLevel := levels[j].name
			leaf := plan.Levels[leafLevel]

			for domain := range root.Collect {
				dom, ok := leaf.Collect[domain]
				if !ok {
					leaf.Collect[domain] = root.Collect[domain]
				} else {
					dom.Metrics = append(dom.Metrics, root.Collect[domain].Metrics...)
					leaf.Collect[domain] = dom
				}
			}
		}
	}

	return levels
}
