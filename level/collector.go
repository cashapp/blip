package level

import (
	"context"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/square/blip"
	"github.com/square/blip/collect"
	"github.com/square/blip/event"
	"github.com/square/blip/monitor"
)

// Collector calls a monitor to collect metrics according to a plan.
type Collector interface {
	// Run runs the collector to collect metrics; it's a blocking call.
	Run(stopChan, doneChan chan struct{}) error

	// ChangePlan changes the plan; it's called by an Adjuster.
	ChangePlan(newState, newPlanName string) error
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
	monitor    *monitor.Monitor
	metronome  *sync.Cond
	planLoader *collect.PlanLoader
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
}

func NewCollector(monitor *monitor.Monitor, metronome *sync.Cond, planLoader *collect.PlanLoader) *collector {
	return &collector{
		monitor:    monitor,
		metronome:  metronome,
		planLoader: planLoader,
		// --
		changeMux: &sync.Mutex{},
		stateMux:  &sync.Mutex{},
		event:     event.MonitorSink{MonitorId: monitor.MonitorId()},
	}
}

func (c *collector) Run(stopChan, doneChan chan struct{}) error {
	defer close(doneChan)

	n := 1          // 1=whole second tick, -1=half second (500ms) tick
	s := float64(0) // number of whole second ticks
	level := 0

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
			continue
		}
		s = s + 1
		c.stateMux.Lock()

		// Determine highest level of metrics to collect. For exmaple, given
		// levels at 1, 5, and 30s, when s=5 we up to
		for i := range c.levels {
			if math.Mod(s, c.levels[i].freq) == 0 {
				level = i
			}
		}
		if level == -1 {
			continue // no metrics to collect at this frequency
		}

		// Collect metrics at this level, unlock, and reset
		c.monitor.Collect(context.Background(), c.levels[level].name)
		c.stateMux.Unlock()
		level = -1

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
