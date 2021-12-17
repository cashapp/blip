package monitor_test

import (
	"context"
	"sync"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"

	"github.com/cashapp/blip"
	"github.com/cashapp/blip/dbconn"
	"github.com/cashapp/blip/metrics"
	"github.com/cashapp/blip/monitor"
	"github.com/cashapp/blip/plan"
	"github.com/cashapp/blip/test/mock"
)

// --------------------------------------------------------------------------

func TestLevelCollector(t *testing.T) {
	// Verify the most basic functionality of the LPC: that it collects each
	// level at the correct frequency. The test plan,
	//   planName := "../test/plans/lpc_1_5_10.yaml"
	// (below) has 3 levels at 1s, 5s, and 10s. But we're not going to wait
	// 10s for a test, so tickerDuration = 10 * time.Millisecond (below) changes
	// the ticks to 10ms, which produces the same effect: collect level 1
	// every tick (1s normally), collect level 2 every 5th tick (5s normally),
	// and collect level 3 every 10th tick (10s normally).

	//blip.Debugging = true

	// Create and register a mock blip.Collector that saves the level name
	// every time it's called. This is quite deep within the call stack,
	// which is what we want: LPC->engine->collector. By using a fake collector
	// but real LPC and enginer, we testing the real, unmodified logic--
	// the LPC and engine don't know or care that this collector is a mock.
	mux := &sync.Mutex{}
	gotLevels := []string{}
	mc := mock.MetricsCollector{
		CollectFunc: func(ctx context.Context, levelName string) ([]blip.MetricValue, error) {
			mux.Lock()
			gotLevels = append(gotLevels, levelName)
			mux.Unlock()
			return nil, nil
		},
	}
	mf := mock.MetricFactory{
		MakeFunc: func(domain string, args blip.CollectorFactoryArgs) (blip.Collector, error) {
			return mc, nil
		},
	}
	metrics.Register(mc.Domain(), mf) // MUST CALL FIRST, before the rest...

	// Make a mini, fake config that uses the test plan and load it realistically
	// because the plan loader combines and sorts levels, etc. This is a lot of
	// boilerplate, but it ensure we test a realistic LPC and monitor--only the
	// collector is fake.
	planName := "../test/plans/lpc_1_5_10.yaml"
	moncfg := blip.ConfigMonitor{
		MonitorId: monitorId1,
		Username:  "root",
		Password:  "test",
		Hostname:  "127.0.0.1:33560", // 5.6
	}
	cfg := blip.Config{
		Plans:    blip.ConfigPlans{Files: []string{planName}},
		Monitors: []blip.ConfigMonitor{moncfg},
	}
	moncfg.ApplyDefaults(cfg)

	dbMaker := dbconn.NewConnFactory(nil, nil)
	pl := plan.NewLoader(nil)

	if err := pl.LoadShared(cfg.Plans, dbMaker); err != nil {
		t.Fatal(err)
	}
	if err := pl.LoadMonitor(moncfg, dbMaker); err != nil {
		t.Fatal(err)
	}

	// Before running the LPC, change its internal ticker interval so we can test
	// this quickly rather than waiting a real 10s
	monitor.TickerDuration(10 * time.Millisecond)
	defer monitor.TickerDuration(1 * time.Second)

	// Create LPC and and run it, but it starts paused until ChangePlan is called
	// starts working once a plan is set.
	lpc := monitor.NewLevelCollector(monitor.LevelCollectorArgs{
		Config:     moncfg,
		Engine:     monitor.NewEngine(monitorId1, db),
		PlanLoader: pl,
		Sinks:      []blip.Sink{mock.Sink{}},
	})
	stopChan := make(chan struct{})
	doneChan := make(chan struct{})
	go lpc.Run(stopChan, doneChan)

	// Wait a few ticks and check LPC status to verify that is, in fact, paused
	time.Sleep(100 * time.Millisecond)
	status := lpc.Status()
	if status.Paused != true {
		t.Errorf("LPC not paused, expected it to be paused until ChangePlan is called")
	}

	// ChangePlan sets the plan and un-pauses (starts collecting the new plan).
	// So call that and wait 15 ticks (150ms / 10s), then close the stopChan
	// to stop the LPC (don't leak goroutines)
	lpc.ChangePlan(blip.STATE_ACTIVE, planName) // set plan, start collecting metrics
	time.Sleep(150 * time.Millisecond)
	close(stopChan)
	select {
	case <-doneChan:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for LPC to stop")
	}

	// LPC should have experienced 15s of running in 150ms because we set the
	// TickerDuration to 10ms (instead of its default 1s). That means the mock
	// collector should have been called 15 times, but CI systems can be really
	// slow, so we'll allow 15 or 16 calls.
	mux.Lock()
	if len(gotLevels) > 16 {
		t.Errorf("got %d levels, expected 15 (exactxly) or 16 (at most)", len(gotLevels))
	}

	// If leveled collection is working properly, the first 12 levels collected--
	// as called by the LPC to engine.Collect--should be this sequence:
	if len(gotLevels) < 12 {
		t.Fatalf("got %d levels, expected at least 12", len(gotLevels))
	}
	expectLevels := []string{
		//            TICK  LEVEL
		"level_3", // 0s    1+2+3 (all levels)
		"level_1", // 1     1..
		"level_1", // 2     1..
		"level_1", // 3     1..
		"level_1", // 4     1..
		"level_2", // 5     .2.
		"level_1", // 6     1..
		"level_1", // 7     1..
		"level_1", // 8     1..
		"level_1", // 9     1..
		"level_3", // 10    ..3
		"level_1", // 11    1..
	}
	assert.ElementsMatch(t, gotLevels[:12], expectLevels)
	mux.Unlock()
}