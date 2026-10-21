// Copyright 2026 Block, Inc.

package monitor

import (
	"context"
	"database/sql"
	"sync"
	"testing"
	"time"

	"github.com/cashapp/blip/v2"
	"github.com/cashapp/blip/v2/plan"
	"github.com/cashapp/blip/v2/status"
	"github.com/cashapp/blip/v2/test/mock"
)

// faultTestEngine returns an engine for monitorId with one level that collects
// the given collectors in order, without preparing a plan or using a database.
func faultTestEngine(monitorId, level string, collectors ...blip.Collector) *Engine {
	engine := newEngineWithDBProvider(blip.ConfigMonitor{MonitorId: monitorId}, &sql.DB{}, nil)
	engine.plan = blip.Plan{Name: "faults", Levels: map[string]blip.Level{level: {Name: level, Freq: "1s"}}}
	for _, c := range collectors {
		cl := &clutch{
			c:              c,
			domain:         c.Domain(),
			cmr:            time.Second,
			collectionChan: engine.collectionChan,
			event:          engine.event,
			counters:       engine.counters,
			Mutex:          &sync.Mutex{},
		}
		engine.collectors[c.Domain()] = cl
		engine.collectAt[level] = append(engine.collectAt[level], cl)
	}
	engine.checkAt[level] = nil
	return engine
}

func counters(t *testing.T, monitorId string) map[string]uint64 {
	t.Helper()
	return status.ReportCounters(monitorId)[monitorId]
}

func TestEngineCountsEMRTimeouts(t *testing.T) {
	const (
		monitorId = "faults-emr"
		level     = "fast"
	)
	defer status.RemoveMonitor(monitorId)

	// With one collector at a time, slow holds the only slot past the EMR, so
	// slow is started but doesn't return, and fast is never started.
	defer func(n int) { CollectParallel = n }(CollectParallel)
	CollectParallel = 1

	release := make(chan struct{})
	slow := mock.MetricsCollector{
		DomainFunc: func() string { return "test.slow" },
		CollectFunc: func(ctx context.Context, _ string) ([]blip.MetricValue, error) {
			<-release
			return nil, nil
		},
	}
	fast := mock.MetricsCollector{DomainFunc: func() string { return "test.fast" }}
	engine := faultTestEngine(monitorId, level, slow, fast)

	got := counters(t, monitorId)
	for _, name := range []string{status.ENGINE_EMR_TIMEOUTS, status.ENGINE_DOMAINS_NOT_STARTED, status.ENGINE_DOMAINS_NOT_RETURNED, status.COLLECTOR_FAULTS, status.COLLECTOR_FLUSH_DROPS} {
		if v, ok := got[name]; !ok || v != 0 {
			t.Errorf("%s = %d (reported %t) before Collect, expected 0 and reported", name, v, ok)
		}
	}

	emrCtx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	engine.Collect(emrCtx, 1, level, time.Now())

	got = counters(t, monitorId)
	if got[status.ENGINE_EMR_TIMEOUTS] != 1 {
		t.Errorf("%s = %d, expected 1", status.ENGINE_EMR_TIMEOUTS, got[status.ENGINE_EMR_TIMEOUTS])
	}
	if got[status.ENGINE_DOMAINS_NOT_STARTED] != 1 {
		t.Errorf("%s = %d, expected 1", status.ENGINE_DOMAINS_NOT_STARTED, got[status.ENGINE_DOMAINS_NOT_STARTED])
	}
	if got[status.ENGINE_DOMAINS_NOT_RETURNED] != 1 {
		t.Errorf("%s = %d, expected 1", status.ENGINE_DOMAINS_NOT_RETURNED, got[status.ENGINE_DOMAINS_NOT_RETURNED])
	}

	close(release)
	engine.Stop()

	// A collection that finishes within the EMR doesn't count
	engine = faultTestEngine(monitorId+"-ok", level, fast)
	defer status.RemoveMonitor(monitorId + "-ok")
	emrCtx2, cancel2 := context.WithTimeout(context.Background(), time.Second)
	defer cancel2()
	if _, err := engine.Collect(emrCtx2, 1, level, time.Now()); err != nil {
		t.Fatal(err)
	}
	if v := counters(t, monitorId+"-ok")[status.ENGINE_EMR_TIMEOUTS]; v != 0 {
		t.Errorf("%s = %d after a collection within the EMR, expected 0", status.ENGINE_EMR_TIMEOUTS, v)
	}
	engine.Stop()
}

func TestEngineCountsCollectorFaults(t *testing.T) {
	const (
		monitorId = "faults-collector"
		level     = "fast"
	)
	defer status.RemoveMonitor(monitorId)

	// stuck ignores its CMR and keeps running into the next interval
	release := make(chan struct{})
	stuck := mock.MetricsCollector{
		DomainFunc: func() string { return "test.stuck" },
		CollectFunc: func(ctx context.Context, _ string) ([]blip.MetricValue, error) {
			<-release
			return nil, nil
		},
	}
	engine := faultTestEngine(monitorId, level, stuck)

	for interval := uint(1); interval <= 2; interval++ {
		emrCtx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		engine.Collect(emrCtx, interval, level, time.Now())
		cancel()
	}

	if v := counters(t, monitorId)[status.COLLECTOR_FAULTS]; v != 1 {
		t.Errorf("%s = %d, expected 1", status.COLLECTOR_FAULTS, v)
	}

	close(release)
	engine.Stop()
}

func TestClutchCountsFlushDrops(t *testing.T) {
	const monitorId = "faults-flush"
	defer status.RemoveMonitor(monitorId)

	// Unbuffered and no receiver: the flush can't be sent, so it's dropped
	cl := &clutch{
		domain:         "test.flush",
		collectionChan: make(chan collection),
		counters:       newEngineCounters(monitorId),
		Mutex:          &sync.Mutex{},
	}
	cl.Lock()
	cl.flush(true)
	cl.Unlock()

	if v := counters(t, monitorId)[status.COLLECTOR_FLUSH_DROPS]; v != 1 {
		t.Errorf("%s = %d, expected 1", status.COLLECTOR_FLUSH_DROPS, v)
	}
}

func TestDroppedTicks(t *testing.T) {
	d := 100 * time.Millisecond
	t0 := time.Now()
	tests := []struct {
		name string
		last time.Time
		now  time.Time
		want uint64
	}{
		{"first tick", time.Time{}, t0, 0},
		{"on time", t0, t0.Add(d), 0},
		{"jitter", t0, t0.Add(d + 30*time.Millisecond), 0},
		{"one dropped", t0, t0.Add(2 * d), 1},
		{"two dropped with jitter", t0, t0.Add(3*d - 20*time.Millisecond), 2},
	}
	for _, tt := range tests {
		if got := droppedTicks(tt.last, tt.now, d); got != tt.want {
			t.Errorf("%s: droppedTicks = %d, expected %d", tt.name, got, tt.want)
		}
	}
}

func TestLevelCollectorCountsDroppedTicks(t *testing.T) {
	const (
		monitorId = "faults-ticks"
		level     = "every"
	)
	defer status.RemoveMonitor(monitorId)

	// The first collection blocks the LCO loop for 3.5 ticks, so the ticker
	// drops ticks. Later collections return immediately.
	var mux sync.Mutex
	calls := 0
	slowOnce := mock.MetricsCollector{
		DomainFunc: func() string { return "test.slow-once" },
		CollectFunc: func(ctx context.Context, _ string) ([]blip.MetricValue, error) {
			mux.Lock()
			calls++
			n := calls
			mux.Unlock()
			if n == 1 {
				time.Sleep(350 * time.Millisecond)
			}
			return nil, nil
		},
	}

	TickerDuration(100*time.Millisecond, 100*time.Millisecond)
	defer TickerDuration(time.Second, time.Second)

	c := newLevelCollectorWithDBProvider(LevelCollectorArgs{Config: blip.ConfigMonitor{MonitorId: monitorId}}, nil)
	c.engine = faultTestEngine(monitorId, level, slowOnce)
	c.plan = c.engine.plan
	c.levels = []plan.SortedLevel{{Freq: 100 * time.Millisecond, Name: level}}
	c.emr = time.Second // longer than the slow collection, so the EMR doesn't expire
	c.paused = false

	// Run returns on the 7th tick: each tick reads one value, then stopChan is closed
	stopChan := make(chan struct{}, 6)
	for i := 0; i < 6; i++ {
		stopChan <- struct{}{}
	}
	close(stopChan)
	doneChan := make(chan struct{})
	go c.Run(stopChan, doneChan)
	select {
	case <-doneChan:
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for LCO to stop")
	}

	// Ticks at about 300ms and 400ms are dropped while the first collection
	// (tick 100ms) blocks until 450ms. Allow for scheduling delay under -race.
	got := counters(t, monitorId)
	if v := got[status.LEVEL_TICKS_DROPPED]; v < 1 || v > 3 {
		t.Errorf("%s = %d, expected 2 (1 to 3)", status.LEVEL_TICKS_DROPPED, v)
	}
	if v := got[status.ENGINE_EMR_TIMEOUTS]; v != 0 {
		t.Errorf("%s = %d, expected 0", status.ENGINE_EMR_TIMEOUTS, v)
	}
}

func TestLevelCollectorCountsDroppedMetrics(t *testing.T) {
	const (
		monitorId = "faults-metrics"
		level     = "every"
	)
	defer status.RemoveMonitor(monitorId)

	c := newLevelCollectorWithDBProvider(LevelCollectorArgs{Config: blip.ConfigMonitor{MonitorId: monitorId}}, nil)
	c.engine = faultTestEngine(monitorId, level, mock.MetricsCollector{DomainFunc: func() string { return "test.ok" }})
	c.plan = c.engine.plan
	c.emr = time.Second

	// No sinks are receiving, so fill the metrics channel to make it block
	for i := 0; i < cap(c.metricsChan); i++ {
		c.metricsChan <- nil
	}
	c.collect(1, level, time.Now())

	if v := counters(t, monitorId)[status.LEVEL_METRICS_DROPPED]; v != 1 {
		t.Errorf("%s = %d, expected 1", status.LEVEL_METRICS_DROPPED, v)
	}
	if v := counters(t, monitorId)[status.LEVEL_TICKS_DROPPED]; v != 0 {
		t.Errorf("%s = %d, expected 0", status.LEVEL_TICKS_DROPPED, v)
	}
	c.engine.Stop()
}
