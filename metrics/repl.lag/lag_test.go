// Copyright 2022, Block, Inc.

package repllag

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/cashapp/blip/test"
)

func TestPrepareForSingleLevelAndNoSourceOnMySQL57(t *testing.T) {
	// default source on MySQL 5.7 should be `blip`
	_, db, err := test.Connection("mysql57")
	if err != nil {
		if test.Build {
			t.Skip("mysql57 not running")
		} else {
			t.Fatal(err)
		}
	}
	defer db.Close()

	c := NewLag(db)

	defaultPlan := test.ReadPlan(t, "")
	_, err = c.Prepare(context.Background(), defaultPlan)
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, "blip", c.lagSourceIn["kpi"])
}

func TestPrepareForSingleLevelAndNoSourceOnMySQL80(t *testing.T) {
	// default source on MySQL 8.0 should be `pfs`
	_, db, err := test.Connection("mysql80")
	if err != nil {
		if test.Build {
			t.Skip("mysql57 not running")
		} else {
			t.Fatal(err)
		}
	}
	defer db.Close()

	c := NewLag(db)

	defaultPlan := test.ReadPlan(t, "")
	_, err = c.Prepare(context.Background(), defaultPlan)
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, "pfs", c.lagSourceIn["kpi"])
}

func TestPrepareWithInvalidSource(t *testing.T) {
	_, db, err := test.Connection("mysql80")
	if err != nil {
		if test.Build {
			t.Skip("mysql57 not running")
		} else {
			t.Fatal(err)
		}
	}
	defer db.Close()

	c := NewLag(db)

	plan := test.ReadPlan(t, "")
	plan.Levels["kpi"].Collect[DOMAIN].Options[OPT_LAG_SOURCE] = "invalid-lag-source"
	_, err = c.Prepare(context.Background(), plan)
	assert.Error(t, err)
}

func TestCollectWithNoSource(t *testing.T) {
	// defaults to pfs
	_, db, err := test.Connection("mysql80")
	if err != nil {
		if test.Build {
			t.Skip("mysql57 not running")
		} else {
			t.Fatal(err)
		}
	}
	defer db.Close()

	c := NewLag(db)

	plan := test.ReadPlan(t, "")
	plan.Levels["kpi"].Collect[DOMAIN].Options[OPT_REPORT_NOT_A_REPLICA] = "yes"
	plan.Levels["kpi"].Collect[DOMAIN].Options[OPT_REPL_CHECK] = "read_only"
	_, err = c.Prepare(context.Background(), plan)
	if err != nil {
		t.Error(err)
	}
	metrics, _ := c.Collect(context.TODO(), "kpi")
	assert.Equal(t, 1, len(metrics))

	assert.Equal(t, metrics[0].Name, "current")
	assert.Equal(t, metrics[0].Value, float64(-1))
}

func TestCollectWithAllSources(t *testing.T) {
	// defaults to pfs
	_, db, err := test.Connection("mysql80")
	if err != nil {
		if test.Build {
			t.Skip("mysql57 not running")
		} else {
			t.Fatal(err)
		}
	}
	defer db.Close()

	srcs := [3]string{"auto", "pfs", "blip"}

	for _, src := range srcs {
		c := NewLag(db)
		plan := test.ReadPlan(t, "")
		plan.Levels["kpi"].Collect[DOMAIN].Options[OPT_REPORT_NOT_A_REPLICA] = "yes"
		if src == "blip" {
			plan.Levels["kpi"].Collect[DOMAIN].Options[OPT_REPORT_NO_HEARTBEAT] = "yes"
		}
		_, err = c.Prepare(context.Background(), plan)
		if err != nil {
			t.Error(err)
		}
		metrics, _ := c.Collect(context.TODO(), "kpi")
		assert.Equal(t, 1, len(metrics))
		assert.Equal(t, metrics[0].Name, "current")
		assert.Equal(t, metrics[0].Value, float64(-1))
	}
}
