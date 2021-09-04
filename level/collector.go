package level

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/square/blip"
	"github.com/square/blip/collect"
	"github.com/square/blip/event"
	"github.com/square/blip/monitor"
	"github.com/square/blip/sink"
	"github.com/square/blip/status"
)

// Collector calls a monitor to collect metrics according to a plan.
type Collector interface {
	// Run runs the collector to collect metrics; it's a blocking call.
	Run(stopChan, doneChan chan struct{}) error

	// ChangePlan changes the plan; it's called by an Adjuster.
	ChangePlan(newState, newPlanName string) error

	// Pause pauses metrics collection until ChangePlan is called.
	Pause() error
}

var _ Collector = &collector{}

type level struct {
	freq float64
	name string
}

type byFreq []level

func (a byFreq) Len() int           { return len(a) }
func (a byFreq) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byFreq) Less(i, j int) bool { return a[i].freq < a[j].freq }

// collector is the implementation of Collector.
type collector struct {
	monitorId        string
	monitor          *monitor.Monitor
	metronome        *sync.Cond
	planLoader       *collect.PlanLoader
	sinks            []sink.Sink
	transformMetrics func(*blip.Metrics) error
	// --
	state                string
	plan                 collect.Plan
	changing             bool
	changePlanCancelFunc context.CancelFunc
	changePlanDoneChan   chan struct{}
	changeMux            *sync.Mutex
	stateMux             *sync.Mutex
	event                event.MonitorSink
	levels               []level
	paused               bool
}

type CollectorArgs struct {
	Monitor          *monitor.Monitor
	Metronome        *sync.Cond
	PlanLoader       *collect.PlanLoader
	Sinks            []sink.Sink
	TransformMetrics func(*blip.Metrics) error
}

func NewCollector(args CollectorArgs) *collector {
	return &collector{
		monitorId:  args.Monitor.MonitorId(),
		monitor:    args.Monitor,
		metronome:  args.Metronome,
		planLoader: args.PlanLoader,
		sinks:      args.Sinks,
		// --
		changeMux: &sync.Mutex{},
		stateMux:  &sync.Mutex{},
		event:     event.MonitorSink{MonitorId: args.Monitor.MonitorId()},
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

	n := 1          // 1=whole second tick, -1=half second (500ms) tick
	s := float64(0) // number of whole second ticks
	level := -1
	c.metronome.L.Lock()
	for {
		c.metronome.Wait() // for tick every 500ms

		// Was Stop called?
		select {
		case <-stopChan: // yes, return immediately
			return nil
		default: // no
		}

		// Multiple n by -1 to flip-flop between 1 and -1 to determine
		// if this is a half- or whole-second tick
		n = n * -1
		if n == -1 {
			continue // ignore half-second ticks (500ms, 1.5s, etc.)
		}
		s = s + 1

		// Determine lowest level to collect
		c.stateMux.Lock()
		if c.paused {
			c.stateMux.Unlock()
			continue
		}
		for i := range c.levels {
			if math.Mod(s, c.levels[i].freq) == 0 {
				level = i
			}
		}
		if level == -1 {
			c.stateMux.Unlock()
			continue // no metrics to collect at this frequency
		}

		// Collect metrics at this level, unlock, and reset
		select {
		case collectLevelChan <- c.levels[level].name:
		default:
			// all collectors blocked
			blip.Debug("all mon chan blocked")
		}
		c.stateMux.Unlock()
		level = -1
	}
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
		metrics, err := c.monitor.Collect(context.Background(), level)
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

	newPlan, err := c.planLoader.Plan(c.monitor.MonitorId(), newPlanName, c.monitor.DB())
	if err != nil {
		return err
	}

	newLevels := LevelUp(&newPlan)

	if err := c.monitor.Prepare(ctx, newPlan); err != nil {
		return err
	}

	c.stateMux.Lock() // -- X lock
	oldState := c.state
	oldPlanName := c.plan.Name
	c.state = newState
	c.plan = newPlan
	c.levels = newLevels
	c.stateMux.Unlock() // -- X unlock

	c.event.Sendf(event.CHANGE_PLAN, "state:%s plan:%s -> state:%s plan:%s", oldState, oldPlanName, newState, newPlan.Name)

	return nil
}

func LevelUp(plan *collect.Plan) []level {
	levels := make([]level, len(plan.Levels))
	i := 0
	for _, l := range plan.Levels {
		d, _ := time.ParseDuration(l.Freq)
		levels[i] = level{
			name: l.Name,
			freq: d.Seconds(),
		}
		i++
	}
	sort.Sort(byFreq(levels))
	blip.Debug("levels: %v", levels)

	// Level N applies to N+(N+1)
	for i := 0; i < len(levels); i++ {
		rootLevel := levels[i].name
		root := plan.Levels[rootLevel]

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

func (c *collector) Pause() error {
	c.stateMux.Lock()
	c.paused = true
	c.stateMux.Unlock()
	return nil
}
