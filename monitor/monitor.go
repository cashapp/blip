package monitor

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"github.com/square/blip"
	"github.com/square/blip/collect"
	"github.com/square/blip/event"
	"github.com/square/blip/metrics"
)

// Monitor monitors a single MySQL instances. It implments blip.Monitor.
type Monitor struct {
	monitorId string
	db        *sql.DB
	mcMaker   metrics.CollectorFactory
	// --
	mcList  map[string]metrics.Collector   // keyed on domain
	atLevel map[string][]metrics.Collector // keyed on level
	*sync.RWMutex
	connected bool
	ready     bool
	plan      collect.Plan
	event     event.MonitorSink
	sem       chan bool
	semSize   int
}

func NewMonitor(monitorId string, db *sql.DB, mcMaker metrics.CollectorFactory) *Monitor {
	sem := make(chan bool, 2)
	semSize := 2
	for i := 0; i < semSize; i++ {
		sem <- true
	}

	return &Monitor{
		monitorId: monitorId,
		db:        db,
		mcMaker:   mcMaker,
		// --
		atLevel: map[string][]metrics.Collector{},
		mcList:  map[string]metrics.Collector{},
		RWMutex: &sync.RWMutex{},
		event:   event.MonitorSink{MonitorId: monitorId},
		sem:     sem,
		semSize: semSize,
	}
}

func (m *Monitor) MonitorId() string {
	return m.monitorId
}

func (m *Monitor) DB() *sql.DB {
	return m.db
}

func (m *Monitor) Config() blip.ConfigMonitor {
	// Get config from DbMon
	return blip.ConfigMonitor{}
}

// Prepare prepares the monitor to collect metrics for the plan. The monitor
// must be successfully prepared for Collect() to work because Prepare()
// initializes metrics collectors for every level of the plan. Prepare() can
// be called again when, for example, the LPA (level.Adjuster) detects a state
// change and calls the LPC (level.Collector) to change plans, which than calls
// this func with the new state plan. (Each monitor has its own LPA and LPC.)
//
// Do not call this func concurrently! It does not guard against concurrent
// calls. Instead, serialization is handled by the only caller: ChangePlan()
// from the monitor's LPC.
func (m *Monitor) Prepare(ctx context.Context, plan collect.Plan) error {
	m.event.Sendf(event.MONITOR_PREPARE_PLAN, plan.Name)

	// Try forever to make a successful connection
	if !m.connected {
		m.event.Send(event.MONITOR_CONNECTING)
		for {
			dbctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err := m.db.PingContext(dbctx)
			cancel()
			if err == nil {
				m.event.Send(event.MONITOR_CONNECTED)
				break
			}

			select {
			case <-ctx.Done():
				return nil
			default:
			}

			time.Sleep(2 * time.Second)
		}
	}

	// Create and prepare metric collectors for every level
	atLevel := map[string][]metrics.Collector{}
	for levelName, level := range plan.Levels {
		for domain, _ := range level.Collect {

			// Make collector if needed
			mc, ok := m.mcList[domain]
			if !ok {
				var err error
				mc, err = m.mcMaker.Make(
					domain,
					metrics.FactoryArgs{
						MonitorId: m.monitorId,
						DB:        m.db,
					},
				)
				if err != nil {
					return err // @todo
				}
				m.mcList[domain] = mc
			}

			// @todo pass ctx

			if err := mc.Prepare(plan); err != nil {
				// @todo
			}

			// At this level, collect from this domain
			atLevel[levelName] = append(atLevel[levelName], mc)

			// OK to keep working?
			select {
			case <-ctx.Done():
				return nil
			default:
			}
		}
	}

	m.Lock()
	m.atLevel = atLevel
	m.plan = plan
	m.ready = true
	m.Unlock()

	return nil
}

func (m *Monitor) Collect(ctx context.Context, levelName string) (*blip.Metrics, error) {
	// Lock while collecting so Preapre cannot change plan while using it.
	// This func shouldn't take a lot less than 1s to exec.
	m.RLock()
	defer func() {
	RECHARGE_SEMAPHORE:
		for i := 0; i < m.semSize; i++ {
			select {
			case m.sem <- true:
			default:
				break RECHARGE_SEMAPHORE
			}
		}
		m.RUnlock()
	}()

	if !m.ready {
		blip.Debug("%s not ready", m.monitorId)
		return nil, nil
	}

	mc := m.atLevel[levelName]
	if mc == nil {
		blip.Debug("%s no", m.monitorId)
		return nil, nil
	}

	bm := &blip.Metrics{
		Plan:      m.plan.Name,
		Level:     levelName,
		MonitorId: m.monitorId,
		Values:    make(map[string]map[string]float64, len(mc)),
	}
	mux := &sync.Mutex{} // serialize writes to Values ^

	var wg sync.WaitGroup
	bm.Begin = time.Now()
	for i := range mc {
		<-m.sem
		wg.Add(1)
		go func(mc metrics.Collector) {
			defer wg.Done()
			defer func() { m.sem <- true }()
			res, err := mc.Collect(ctx, levelName)
			if err != nil {
				// @todo
			}
			mux.Lock()
			bm.Values[mc.Domain()] = res.Values
			mux.Unlock()
		}(mc[i])
	}
	wg.Wait()
	bm.End = time.Now()

	return bm, nil
}
