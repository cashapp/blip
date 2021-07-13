package monitor

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"github.com/square/blip/collect"
	"github.com/square/blip/event"
	"github.com/square/blip/metrics"
)

// Monitor monitors a single MySQL instances.
type Monitor struct {
	Id      string
	DB      *sql.DB
	mcMaker metrics.CollectorFactory
	// --
	atLevel map[string][]metrics.Collector // keyed on level
	colls   map[string]metrics.Collector
	*sync.Mutex
	connected bool
	ready     bool
	plan      collect.Plan
	event     event.MonitorSink
}

func NewMonitor(monitorId string, db *sql.DB, mcMaker metrics.CollectorFactory) *Monitor {
	return &Monitor{
		Id:      monitorId,
		DB:      db,
		mcMaker: mcMaker,
		// --
		atLevel: map[string][]metrics.Collector{},
		colls:   map[string]metrics.Collector{},
		Mutex:   &sync.Mutex{},
		event:   event.MonitorSink{MonitorId: monitorId},
	}
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

	// Try forever to make a successful connection
	if !m.connected {
		m.event.Send(event.MONITOR_CONNECTING)
		for {
			dbctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err := m.DB.PingContext(dbctx)
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
		for domName, _ := range level.Collect {

			// Make collector if needed
			mc, ok := m.colls[domName]
			if !ok {
				var err error
				mc, err = m.mcMaker.Make(domName)
				if err != nil {
					return err // @todo
				}
				m.colls[domName] = mc
			}

			atLevel[levelName] = append(atLevel[levelName], mc)

			mc.Prepare(plan) // @todo pass ctx

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

func (m *Monitor) Collect(ctx context.Context, levelName string) (collect.Metrics, error) {
	m.Lock()
	defer m.Unlock()
	if !m.ready {
		return collect.Metrics{}, nil
	}
	return collect.Metrics{}, nil
}
