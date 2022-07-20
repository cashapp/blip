// Copyright 2022 Block, Inc.

package monitor_test

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/cashapp/blip"
	"github.com/cashapp/blip/dbconn"
	"github.com/cashapp/blip/monitor"
	"github.com/cashapp/blip/plan"
	"github.com/cashapp/blip/test/mock"
)

const (
	monitorId1 = "testmon1"
)

var (
	db *sql.DB
)

// First Method that gets run before all tests.
func TestMain(m *testing.M) {
	var err error

	// Read plans from files

	// Connect to MySQL
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/?parseTime=true",
		"root",
		"test",
		"localhost",
		"33570",
	)
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	code := m.Run() // run tests
	os.Exit(code)
}

// --------------------------------------------------------------------------

func TestMonitor(t *testing.T) {
	//blip.Debugging = true

	moncfg := blip.ConfigMonitor{
		MonitorId: monitorId1,
		Username:  "root",
		Password:  "test",
		Hostname:  "127.0.0.1:33560", // 5.6
	}
	cfg := blip.Config{
		Plans:    blip.ConfigPlans{Files: []string{"../test/plans/version.yaml"}},
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

	mux := &sync.Mutex{}
	var gotMetrics *blip.Metrics
	sink := mock.Sink{
		SendFunc: func(ctx context.Context, m *blip.Metrics) error {
			mux.Lock()
			m = gotMetrics
			mux.Unlock()
			return nil
		},
	}

	mon := monitor.NewMonitor(monitor.MonitorArgs{
		Config:     moncfg,
		PlanLoader: pl,
		DbMaker:    dbMaker,
		Sinks:      []blip.Sink{sink},
		TransformMetric: func(metrics *blip.Metrics) error {
			return nil
		},
	})

	if err := mon.Start(); err != nil {
		t.Fatal(err)
	}

	time.Sleep(1300 * time.Millisecond)

	status := mon.Status()
	t.Logf("%+v", status)

	if status.MonitorId != monitorId1 {
		t.Errorf("MonitorStatus.MonitorId = '%s', expected '%s'", status.MonitorId, monitorId1)
	}
	if status.Collector.Engine.CollectAll != 1 && status.Collector.Engine.CollectAll != 2 {
		t.Errorf("MonitorStatus.Collector.Engine.CollectAll = %d, expected 1 or 2", status.Collector.Engine.CollectAll)
	}
	if status.Collector.Engine.CollectSome != 0 {
		t.Errorf("MonitorStatus.Collector.Engine.CollectSome= %d, expected 0", status.Collector.Engine.CollectSome)
	}
	if status.Collector.Engine.CollectFail != 0 {
		t.Errorf("MonitorStatus.Collector.Engine.CollectFail = %d, expected 0", status.Collector.Engine.CollectFail)
	}

	err := mon.Stop()
	if err != nil {
		t.Error(err)
	}
}
