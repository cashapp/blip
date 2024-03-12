// Copyright 2023 Block, Inc.

package monitor_test

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/go-test/deep"
	"github.com/stretchr/testify/assert"

	"github.com/cashapp/blip"
	"github.com/cashapp/blip/dbconn"
	"github.com/cashapp/blip/metrics"
	"github.com/cashapp/blip/monitor"
	"github.com/cashapp/blip/plan"
	"github.com/cashapp/blip/status"
	"github.com/cashapp/blip/test"
	"github.com/cashapp/blip/test/mock"
)

// --------------------------------------------------------------------------

func TestLevelCollector(t *testing.T) {
	// Verify the most basic functionality of the LCO: that it collects each
	// level at the correct frequency. The test plan,
	//   planName := "../test/plans/lpc_1_5_10.yaml"
	// (below) has 3 levels at 1s, 5s, and 10s. But we're not going to wait
	// 10s for a test, so tickerDuration = 10 * time.Millisecond (below) changes
	// the ticks to 10ms, which produces the same effect: collect level 1
	// every tick (1s normally), collect level 2 every 5th tick (5s normally),
	// and collect level 3 every 10th tick (10s normally).

	// Create and register a mock blip.Collector that saves the level name
	// every time it's called. This is quite deep within the call stack,
	// which is what we want: LCO->engine->collector. By using a fake collector
	// but real LCO and engine , we testing the real, unmodified logic--
	// the LCO and engine don't know or care that this collector is a mock.
	_, db, err := test.Connection("mysql57")
	if err != nil {
		if test.Build {
			t.Skip("mysql57 not running")
		} else {
			t.Fatal(err)
		}
	}
	defer db.Close()

	monitorId := "m1"
	defer status.RemoveMonitor(monitorId)

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
	defer metrics.Remove(mc.Domain())

	// Make a mini, fake config that uses the test plan and load it realistically
	// because the plan loader combines and sorts levels, etc. This is a lot of
	// boilerplate, but it ensure we test a realistic LCO and monitor--only the
	// collector is fake.
	planName := "../test/plans/lpc_1_5_10.yaml"
	moncfg := blip.ConfigMonitor{
		MonitorId: monitorId,
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

	// Before running the LCO, change its internal ticker interval so we can test
	// this quickly rather than waiting a real 10s
	monitor.TickerDuration(10 * time.Millisecond)
	defer monitor.TickerDuration(1 * time.Second)

	mockSink := mock.Sink{
		SendFunc: func(ctx context.Context, m *blip.Metrics) error {
			t.Logf("Got metrics: %s", m.MonitorId)
			return nil
		},
	}

	// Create LCO and and run it, but it starts paused until ChangePlan is called
	// starts working once a plan is set.
	lpc := monitor.NewLevelCollector(monitor.LevelCollectorArgs{
		Config:     moncfg,
		DB:         db,
		PlanLoader: pl,
		Sinks:      []blip.Sink{mockSink},
	})
	stopChan := make(chan struct{})
	doneChan := make(chan struct{})
	go lpc.Run(stopChan, doneChan)

	// Wait a few ticks and check LCO status to verify that is, in fact, paused
	time.Sleep(100 * time.Millisecond)
	s := status.ReportMonitors(monitorId)
	if !strings.Contains(s[monitorId][status.LEVEL_COLLECTOR], "paused") {
		t.Errorf("LCO not paused, expected it to be paused until ChangePlan is called (status=%+v)", s)
	}

	// ChangePlan sets the plan and un-pauses (starts collecting the new plan).
	// So call that and wait 15 ticks (150ms / 10s), then close the stopChan
	// to stop the LCO (don't leak goroutines)
	lpc.ChangePlan(blip.STATE_ACTIVE, planName) // set plan, start collecting metrics
	time.Sleep(150 * time.Millisecond)
	close(stopChan)
	select {
	case <-doneChan:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for LCO to stop")
	}

	// LCO should have experienced 15s of running in 150ms because we set the
	// TickerDuration to 10ms (instead of its default 1s). That means the mock
	// collector should have been called 15 times, but CI systems can be really
	// slow, so we'll allow 15 or 16 calls.
	mux.Lock()
	if len(gotLevels) > 16 {
		t.Errorf("got %d levels, expected 15 or 16 ", len(gotLevels))
	}

	// If leveled collection is working properly, the first 12 levels collected--
	// as called by the LCO to engine.Collect--should be this sequence:
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

func TestLevelCollector_SinkProcessing(t *testing.T) {
	// Verify that values collected are sent to the sink
	// in the order they were collected. None of the values
	// should be dropped or missed

	// Create and register a mock blip.Collector that saves the level name
	// every time it's called. This is quite deep within the call stack,
	// which is what we want: LCO->engine->collector. By using a fake collector
	// but real LCO and enginer, we testing the real, unmodified logic--
	// the LCO and engine don't know or care that this collector is a mock.
	_, db, err := test.Connection("mysql57")
	if err != nil {
		if test.Build {
			t.Skip("mysql57 not running")
		} else {
			t.Fatal(err)
		}
	}
	defer db.Close()

	monitorId := "m1"
	defer status.RemoveMonitor(monitorId)

	mux := &sync.Mutex{}
	indexMux := &sync.Mutex{}
	index := 0
	gotMetrics := [][]blip.MetricValue{}

	metricValues := [][]blip.MetricValue{}
	for i := 0; i < 20; i++ {
		metricValues = append(metricValues, []blip.MetricValue{
			{
				Name:  "test-metric",
				Value: float64(i),
				Type:  blip.GAUGE,
			}})
	}

	mc := mock.MetricsCollector{
		CollectFunc: func(ctx context.Context, levelName string) ([]blip.MetricValue, error) {
			indexMux.Lock()
			defer indexMux.Unlock()
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}
			value := metricValues[index]
			index = index + 1
			if index >= len(metricValues) {
				index = 0
			}
			return value, nil
		},
	}
	mf := mock.MetricFactory{
		MakeFunc: func(domain string, args blip.CollectorFactoryArgs) (blip.Collector, error) {
			return mc, nil
		},
	}
	mockSink := mock.Sink{
		SendFunc: func(ctx context.Context, m *blip.Metrics) error {
			mux.Lock()
			gotMetrics = append(gotMetrics, m.Values["test"])
			mux.Unlock()
			return nil
		},
	}
	metrics.Register(mc.Domain(), mf) // MUST CALL FIRST, before the rest...
	defer metrics.Remove(mc.Domain())

	// Make a mini, fake config that uses the test plan and load it realistically
	// because the plan loader combines and sorts levels, etc. This is a lot of
	// boilerplate, but it ensure we test a realistic LCO and monitor--only the
	// collector is fake.
	planName := "../test/plans/lpc_1_5_10.yaml"
	moncfg := blip.ConfigMonitor{
		MonitorId: monitorId,
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

	// Before running the LCO, change its internal ticker interval so we can test
	// this quickly rather than waiting a real 10s
	monitor.TickerDuration(10 * time.Millisecond)
	defer monitor.TickerDuration(1 * time.Second)

	// Create LCO and and run it, but it starts paused until ChangePlan is called
	// starts working once a plan is set.
	lpc := monitor.NewLevelCollector(monitor.LevelCollectorArgs{
		Config:     moncfg,
		DB:         db,
		PlanLoader: pl,
		Sinks:      []blip.Sink{mockSink},
	})
	stopChan := make(chan struct{})
	doneChan := make(chan struct{})
	go lpc.Run(stopChan, doneChan)

	// Wait a few ticks and check LCO status to verify that is, in fact, paused
	time.Sleep(100 * time.Millisecond)
	s := status.ReportMonitors(monitorId)
	if !strings.Contains(s[monitorId][status.LEVEL_COLLECTOR], "paused") {
		t.Errorf("LCO not paused, expected it to be paused until ChangePlan is called (status=%+v)", s)
	}

	// ChangePlan sets the plan and un-pauses (starts collecting the new plan).
	// So call that and wait 15 ticks (150ms / 10s), then close the stopChan
	// to stop the LCO (don't leak goroutines)
	lpc.ChangePlan(blip.STATE_ACTIVE, planName) // set plan, start collecting metrics
	time.Sleep(150 * time.Millisecond)
	close(stopChan)
	select {
	case <-doneChan:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for LCO to stop")
	}

	// LCO should have experienced 15s of running in 150ms because we set the
	// TickerDuration to 10ms (instead of its default 1s). That means the mock
	// collector should have been called 15 times, but CI systems can be really
	// slow, so we'll allow 15 or 16 calls.
	mux.Lock()
	if len(gotMetrics) > 16 {
		t.Errorf("got %d metric values, expected 15 (exactxly) or 16 (at most)", len(gotMetrics))
	}

	// If leveled collection is working properly, the first 12 metric values collected--
	// as called by the LCO to engine.Collect--should be this sequence:
	if len(gotMetrics) < 12 {
		t.Fatalf("got %d metric values, expected at least 12", len(gotMetrics))
	}

	if diff := deep.Equal(gotMetrics[:12], metricValues[:12]); diff != nil {
		t.Error(diff)
	}
	mux.Unlock()
}

func TestLevelCollector_CancelCollector(t *testing.T) {
	// Verify that a long-running collector is properly
	// halted when the next instance of that collector
	// needs to start.

	// Create and register a mock blip.Collector that saves the level name
	// every time it's called. This is quite deep within the call stack,
	// which is what we want: LCO->engine->collector. By using a fake collector
	// but real LCO and enginer, we testing the real, unmodified logic--
	// the LCO and engine don't know or care that this collector is a mock.
	_, db, err := test.Connection("mysql57")
	if err != nil {
		if test.Build {
			t.Skip("mysql57 not running")
		} else {
			t.Fatal(err)
		}
	}
	defer db.Close()

	monitorId := "m1"
	defer status.RemoveMonitor(monitorId)

	mux := &sync.Mutex{}
	indexMux := &sync.Mutex{}
	index := 0
	gotMetrics := [][]blip.MetricValue{}

	metricValues := [][]blip.MetricValue{}
	for i := 0; i < 20; i++ {
		metricValues = append(metricValues, []blip.MetricValue{
			{
				Name:  "test-metric",
				Value: float64(i),
				Type:  blip.GAUGE,
			}})
	}

	mc := mock.MetricsCollector{
		CollectFunc: func(ctx context.Context, levelName string) ([]blip.MetricValue, error) {
			indexMux.Lock()
			defer indexMux.Unlock()
			value := metricValues[index]
			index = index + 1
			if index >= len(metricValues) {
				index = 0
			}

			// Simulate a long-running collector
			if index == 5 {
				<-ctx.Done()
				return nil, ctx.Err()
			}

			return value, nil
		},
	}
	mf := mock.MetricFactory{
		MakeFunc: func(domain string, args blip.CollectorFactoryArgs) (blip.Collector, error) {
			return mc, nil
		},
	}
	mockSink := mock.Sink{
		SendFunc: func(ctx context.Context, m *blip.Metrics) error {
			mux.Lock()
			if values, ok := m.Values["test"]; ok {
				gotMetrics = append(gotMetrics, values)
			}
			mux.Unlock()
			return nil
		},
	}
	metrics.Register(mc.Domain(), mf) // MUST CALL FIRST, before the rest...
	defer metrics.Remove(mc.Domain())

	// Make a mini, fake config that uses the test plan and load it realistically
	// because the plan loader combines and sorts levels, etc. This is a lot of
	// boilerplate, but it ensure we test a realistic LCO and monitor--only the
	// collector is fake.
	planName := "../test/plans/lpc_1_5_10.yaml"
	moncfg := blip.ConfigMonitor{
		MonitorId: monitorId,
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

	// Before running the LCO, change its internal ticker interval so we can test
	// this quickly rather than waiting a real 10s
	monitor.TickerDuration(10 * time.Millisecond)
	defer monitor.TickerDuration(1 * time.Second)

	// Create LCO and and run it, but it starts paused until ChangePlan is called
	// starts working once a plan is set.
	lpc := monitor.NewLevelCollector(monitor.LevelCollectorArgs{
		Config:     moncfg,
		DB:         db,
		PlanLoader: pl,
		Sinks:      []blip.Sink{mockSink},
	})
	stopChan := make(chan struct{})
	doneChan := make(chan struct{})
	go lpc.Run(stopChan, doneChan)

	// Wait a few ticks and check LCO status to verify that is, in fact, paused
	time.Sleep(100 * time.Millisecond)
	s := status.ReportMonitors(monitorId)
	if !strings.Contains(s[monitorId][status.LEVEL_COLLECTOR], "paused") {
		t.Errorf("LCO not paused, expected it to be paused until ChangePlan is called (status=%+v)", s)
	}

	// ChangePlan sets the plan and un-pauses (starts collecting the new plan).
	// So call that and wait 15 ticks (150ms / 10s), then close the stopChan
	// to stop the LCO (don't leak goroutines)
	lpc.ChangePlan(blip.STATE_ACTIVE, planName) // set plan, start collecting metrics
	time.Sleep(150 * time.Millisecond)
	close(stopChan)
	select {
	case <-doneChan:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for LCO to stop")
	}

	// LCO should have experienced 15s of running in 150ms because we set the
	// TickerDuration to 10ms (instead of its default 1s). That means the mock
	// collector should have been called 15 times, but CI systems can be really
	// slow, so we'll allow 15 or 16 calls.
	mux.Lock()
	if len(gotMetrics) > 16 {
		t.Errorf("got %d metric values, expected 15 (exactly) or 16 (at most)", len(gotMetrics))
	}

	// If leveled collection is working properly, the first 12 metric values collected--
	// as called by the LCO to engine.Collect--should be this sequence:
	if len(gotMetrics) < 12 {
		t.Fatalf("got %d metric values, expected at least 12", len(gotMetrics))
	}

	if diff := deep.Equal(gotMetrics[:12], metricValues[:12]); diff != nil {
		t.Error(diff)
	}
	mux.Unlock()
}

type metricPayload struct {
	metrics []blip.MetricValue
	err     error
}

func makeProgressiveCollector(t *testing.T) *mock.MetricsCollector {
	var cleanupCtx context.Context

	index := 0
	metricValues := [][]blip.MetricValue{}
	for i := 0; i < 20; i++ {
		metricValues = append(metricValues, []blip.MetricValue{
			{
				Name:  "test-metric",
				Value: float64(i),
				Type:  blip.GAUGE,
			}})
	}

	requestChan := make(chan context.Context)
	flushChan := make(chan metricPayload)
	running := false
	var mx sync.Mutex

	timerDuration := []time.Duration{5 * time.Millisecond, 60 * time.Millisecond}

	run := func() {
		defer func() {
			mx.Lock()
			running = false
			mx.Unlock()
		}()

		for {
			localCtx := context.Background()

			t.Log("Starting collection loop")
			tc := time.NewTicker(timerDuration[index%len(timerDuration)])
			value := metricValues[index]
			index = index + 1
			if index >= len(metricValues) {
				index = 0
			}

		RETRY_OUTPUT:
			select {
			case <-localCtx.Done():
				t.Log("Collector got a request for metrics")
				// Caller requires an update
				// e.g. Domain timeout
				flushChan <- metricPayload{
					metrics: value,
				}
				// Switch back to a default context now that we have given the caller what they need
				localCtx = context.Background()
			case newCtx := <-requestChan:
				// collector started a new pass
				// so we need the context for that pass
				// Once we get it immediately check to see if we need to return metrics
				localCtx = newCtx
				goto RETRY_OUTPUT
			case <-tc.C:
				t.Log("Collector finished query, sending metrics")
				// Simulate that our results are "done"
				flushChan <- metricPayload{
					metrics: value,
				}
				tc.Stop()
				// Switch back to a default context now that we have given the caller what they need
				localCtx = context.Background()
				return

			case <-cleanupCtx.Done():
				t.Log("Collector is stopping")
				// The cleanup has been called on the collector
				close(flushChan)
				close(requestChan)
				tc.Stop()
				return
			}

			tc.Stop()
		}

	}

	longMc := mock.MetricsCollector{
		PrepareFunc: func(ctx context.Context, plan blip.Plan) (func(), error) {
			ctx, cleanupFn := context.WithCancel(context.Background())
			cleanupCtx = ctx
			return func() {
				cleanupFn()
			}, nil
		},
		DomainFunc: func() string {
			return "long"
		},
		CollectFunc: func(ctx context.Context, levelName string) ([]blip.MetricValue, error) {
			t.Logf("Entering collector: %s", levelName)
			mx.Lock()
			if !running {
				t.Log("Starting new collector")
				running = true
				go run()
			}
			mx.Unlock()

			t.Logf("Waiting for metrics: %s", levelName)
			// Pass the internal collector the context for this pass
			requestChan <- ctx
			// We will now get a result from the internal collector,
			// either due to it finishing or because our context ended
			payload, ok := <-flushChan
			t.Log("Got payload from collector")
			if !ok {
				return nil, fmt.Errorf("Collector closed")
			}
			return payload.metrics, payload.err
		},
	}

	return &longMc
}

func TestLevelCollector_ProgressiveCollector(t *testing.T) {
	//blip.Debugging = true
	// Verifies that a collect progressive returns
	// results when it's next run comes up.

	// Create and register a mock blip.Collector that saves the level name
	// every time it's called. This is quite deep within the call stack,
	// which is what we want: LCO->engine->collector. By using a fake collector
	// but real LCO and enginer, we testing the real, unmodified logic--
	// the LCO and engine don't know or care that this collector is a mock.
	_, db, err := test.Connection("mysql57")
	if err != nil {
		if test.Build {
			t.Skip("mysql57 not running")
		} else {
			t.Fatal(err)
		}
	}
	defer db.Close()

	monitorId := "m1"
	defer status.RemoveMonitor(monitorId)

	mux := &sync.Mutex{}
	indexMux := &sync.Mutex{}
	index := 0
	gotMetrics := [][]blip.MetricValue{}
	longMetrics := [][]blip.MetricValue{}

	metricValues := [][]blip.MetricValue{}
	for i := 0; i < 20; i++ {
		metricValues = append(metricValues, []blip.MetricValue{
			{
				Name:  "test-metric",
				Value: float64(i),
				Type:  blip.GAUGE,
			}})
	}

	mc := mock.MetricsCollector{
		CollectFunc: func(ctx context.Context, levelName string) ([]blip.MetricValue, error) {
			indexMux.Lock()
			defer indexMux.Unlock()
			value := metricValues[index]
			index = index + 1
			if index >= len(metricValues) {
				index = 0
			}

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}

			return value, nil
		},
	}

	longMc := makeProgressiveCollector(t)

	mf := mock.MetricFactory{
		MakeFunc: func(domain string, args blip.CollectorFactoryArgs) (blip.Collector, error) {
			if domain == "long" {
				return longMc, nil
			}
			return mc, nil
		},
	}
	mockSink := mock.Sink{
		SendFunc: func(ctx context.Context, m *blip.Metrics) error {
			mux.Lock()
			if values, ok := m.Values["test"]; ok {
				gotMetrics = append(gotMetrics, values)
			}
			if values, ok := m.Values["long"]; ok {
				longMetrics = append(longMetrics, values)
			}
			mux.Unlock()
			return nil
		},
	}
	metrics.Register(mc.Domain(), mf) // MUST CALL FIRST, before the rest...
	metrics.Register(longMc.Domain(), mf)
	defer metrics.Remove(mc.Domain())
	defer metrics.Remove(longMc.Domain())

	// Make a mini, fake config that uses the test plan and load it realistically
	// because the plan loader combines and sorts levels, etc. This is a lot of
	// boilerplate, but it ensure we test a realistic LCO and monitor--only the
	// collector is fake.
	planName := "../test/plans/lpc_2_domains.yaml"
	moncfg := blip.ConfigMonitor{
		MonitorId: monitorId,
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

	// Before running the LCO, change its internal ticker interval so we can test
	// this quickly rather than waiting a real 10s
	monitor.TickerDuration(10 * time.Millisecond)
	defer monitor.TickerDuration(1 * time.Second)

	// Create LCO and and run it, but it starts paused until ChangePlan is called
	// starts working once a plan is set.
	lpc := monitor.NewLevelCollector(monitor.LevelCollectorArgs{
		Config:     moncfg,
		DB:         db,
		PlanLoader: pl,
		Sinks:      []blip.Sink{mockSink},
	})
	stopChan := make(chan struct{})
	doneChan := make(chan struct{})
	go lpc.Run(stopChan, doneChan)

	// Wait a few ticks and check LCO status to verify that is, in fact, paused
	time.Sleep(100 * time.Millisecond)
	s := status.ReportMonitors(monitorId)
	if !strings.Contains(s[monitorId][status.LEVEL_COLLECTOR], "paused") {
		t.Errorf("LCO not paused, expected it to be paused until ChangePlan is called (status=%+v)", s)
	}

	// ChangePlan sets the plan and un-pauses (starts collecting the new plan).
	// So call that and wait 15 ticks (150ms / 10s), then close the stopChan
	// to stop the LCO (don't leak goroutines)
	lpc.ChangePlan(blip.STATE_ACTIVE, planName) // set plan, start collecting metrics
	time.Sleep(150 * time.Millisecond)
	close(stopChan)
	t.Log("Starting wait for LCO to stop")
	select {
	case <-doneChan:
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for LCO to stop")
	}

	// LCO should have experienced 15s of running in 150ms because we set the
	// TickerDuration to 10ms (instead of its default 1s). That means the mock
	// collector should have been called 15 times, but CI systems can be really
	// slow, so we'll allow 15 or 16 calls.
	mux.Lock()
	defer mux.Unlock()
	if len(gotMetrics) > 16 {
		t.Errorf("got %d metric values, expected 15 (exactly) or 16 (at most)", len(gotMetrics))
	}

	// If leveled collection is working properly, the first 12 metric values collected--
	// as called by the LCO to engine.Collect--should be this sequence:
	if len(gotMetrics) < 12 {
		t.Fatalf("got %d metric values, expected at least 12", len(gotMetrics))
	}

	if len(longMetrics) < 2 {
		t.Fatalf("got %d metric values from 'long' domain, expected at least 2", len(longMetrics))
	}

	if diff := deep.Equal(gotMetrics[:12], metricValues[:12]); diff != nil {
		t.Error(diff)
	}
	if diff := deep.Equal(longMetrics[:2], metricValues[:2]); diff != nil {
		t.Error(diff)
	}
}

func TestLevelCollectorChangePlan(t *testing.T) {
	// ChangePlan is called async by LPA (if enabled), and ChangePlan runs a
	// goroutine (called changePlan) to handle it. When called again, it should
	// cancel the current changePlan goroutine, if any, then start a new one
	// for the new plan.
	//
	// To simulate, we need to make changePlan block, but it only does two things:
	// PlanLoader.Plan() to load the new plan, then Engine.Prepare() to prepare
	// the new plan. Neither have any direct callbacks or interfaces that we
	// can mock to make them slow (because these components are meant to be the
	// fastest and most efficient). However, Prepare() calls the same method on all
	// collectors, which we can mock. So we'll inject slowness in the callstack like:
	//
	//   4.   Collector.Prepare
	//   3.   Engine.Prepare
	//   2.   LCO.changePlan (goroutine)
	//   1. LCO.ChangePlan
	//   0. test

	// Create and register a mock blip.Collector that saves the level name
	// every time it's called. This is quite deep within the call stack,
	// which is what we want: LCO->engine->collector. By using a fake collector
	// but real LCO and engine, we testing the real, unmodified logic--
	// the LCO and engine don't know or care that this collector is a mock.
	_, db, err := test.Connection("mysql57")
	if err != nil {
		if test.Build {
			t.Skip("mysql57 not running")
		} else {
			t.Fatal(err)
		}
	}
	defer db.Close()

	monitorId := "m2"
	defer status.RemoveMonitor(monitorId)

	callChan := make(chan bool, 1)
	returnChan := make(chan error, 1)
	mc := mock.MetricsCollector{
		PrepareFunc: func(ctx context.Context, plan blip.Plan) (func(), error) {
			blip.Debug("collector called")
			callChan <- true // signal test
			blip.Debug("collector waiting")
			err := <-returnChan // wait for test
			blip.Debug("collector return")
			return nil, err
		},
	}
	mf := mock.MetricFactory{
		MakeFunc: func(domain string, args blip.CollectorFactoryArgs) (blip.Collector, error) {
			return mc, nil
		},
	}
	metrics.Register(mc.Domain(), mf) // MUST CALL FIRST, before the rest...
	defer metrics.Remove(mc.Domain())

	// Make a mini, fake config that uses the test plan and load it realistically
	planName := "../test/plans/test.yaml"
	moncfg := blip.ConfigMonitor{MonitorId: monitorId}
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

	// Create LCO and run it, but it starts paused until ChangePlan is called
	lco := monitor.NewLevelCollector(monitor.LevelCollectorArgs{
		Config:     moncfg,
		DB:         db,
		PlanLoader: pl,
		Sinks:      []blip.Sink{mock.Sink{}},
	})
	stopChan := make(chan struct{})
	doneChan := make(chan struct{})
	go lco.Run(stopChan, doneChan)
	defer close(stopChan)

	// Call stack:
	//   4.   Collector.Prepare
	//   3.   Engine.Prepare
	//   2.   LCO.changePlan (goroutine)
	//   1. LCO.ChangePlan
	//   0. test

	// CP1: first change plan: returns immediately but the mock collector (mc) blocks on callChan
	lco.ChangePlan(blip.STATE_ACTIVE, planName)
	select {
	case <-callChan:
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for CP1")
	}
	// CP1 is blocked in call stack 4

	// CP2: second change plan: cancels CP1, waits for CP1 to close its doneChan, then proceeds
	go lco.ChangePlan(blip.STATE_READ_ONLY, planName)

	// CP2 is blocked in call stack 2

	time.Sleep(100 * time.Millisecond)
	returnChan <- fmt.Errorf("fake context canceled error") // CP1 returns; CP2 advances to call stack 4

	select {
	case <-callChan:
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for CP2")
	}

	returnChan <- nil // CP2 returns, which sets the state to READ_ONLY
	time.Sleep(150 * time.Millisecond)
	s := status.ReportMonitors(monitorId)
	if s[monitorId][status.LEVEL_STATE] != blip.STATE_READ_ONLY {
		t.Errorf("got state %s, expected %s", s[monitorId][status.LEVEL_STATE], blip.STATE_READ_ONLY)
	}
}
