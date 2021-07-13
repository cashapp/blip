package level

import (
	"context"
	"log"
	"sync"

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
}

func NewCollector(monitor *monitor.Monitor, metronome *sync.Cond, planLoader *collect.PlanLoader) *collector {
	return &collector{
		monitor:    monitor,
		metronome:  metronome,
		planLoader: planLoader,
		// --
		changeMux: &sync.Mutex{},
		stateMux:  &sync.Mutex{},
		event:     event.MonitorSink{MonitorId: monitor.Id},
	}
}

func (c *collector) Run(stopChan, doneChan chan struct{}) error {
	defer close(doneChan)

	n := 1 // 1=whole second tick, -1=half second (500ms) tick
	s := 0 // number of whole second ticks
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
		if n == 1 { // whole second tick
			s = s + 1
			log.Println("woke up - whole", s)
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

	newPlan, err := c.planLoader.Plan(c.monitor.Id, newPlanName, c.monitor.DB)
	if err != nil {
		return err
	}

	if err := c.monitor.Prepare(ctx, newPlan); err != nil {
		return err
	}

	c.stateMux.Lock() // -- X lock
	oldState := c.state
	oldPlanName := c.plan.Name
	c.state = newState
	c.plan = newPlan
	c.stateMux.Unlock() // -- X unlock

	c.event.Sendf(event.CHANGE_PLAN, "state:%s plan:%s -> state:%s plan:%s", oldState, oldPlanName, newState, newPlan.Name)

	return nil
}
