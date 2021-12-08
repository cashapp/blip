package monitor_test

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/cashapp/blip"
	"github.com/cashapp/blip/dbconn"
	"github.com/cashapp/blip/monitor"
	"github.com/cashapp/blip/plan"
)

// --------------------------------------------------------------------------

func TestLoaderLoadOne(t *testing.T) {
	// Test most basic monitor loader function: loading one monitor from the
	// a Bilp config. The config details don't really matter here; we just
	// want to see that the Loader loads the monitor, which we detect using
	// its optiona RunMonitor callback to both inspect what it loaded and
	// prevent it from actually running the monitor.
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

	// Optional Loader callback to yay or nay running a Monitor. For this test,
	// it's nay: we don't need to run the monitor, just see that it's loaded.
	mux := &sync.Mutex{}
	gotMonitors := []*monitor.Monitor{}
	runMonitor := func(mon *monitor.Monitor) bool {
		mux.Lock()
		gotMonitors = append(gotMonitors, mon)
		mux.Unlock()
		return false // do not run Monitor
	}

	// Create a new Loader and call its main method: Load.
	args := monitor.LoaderArgs{
		Config:       cfg,
		DbMaker:      dbconn.NewConnFactory(nil, nil),
		PlanLoader:   plan.NewLoader(nil),
		LoadMonitors: nil,
		RunMonitor:   runMonitor,
	}
	loader := monitor.NewLoader(args)
	err := loader.Load(context.Background())
	if err != nil {
		t.Error(err)
	}

	// Verify that it loaded the one monitor
	gotIds := []string{}
	mux.Lock()
	for _, mon := range gotMonitors {
		gotIds = append(gotIds, mon.MonitorId())
	}
	mux.Unlock()
	expectIds := []string{moncfg.MonitorId}
	assert.ElementsMatch(t, gotIds, expectIds)
}
