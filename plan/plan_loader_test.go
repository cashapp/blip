package plan_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/square/blip"
	//"github.com/square/blip/dbconn"
	"github.com/square/blip/plan"
	"github.com/square/blip/proto"
	//	"github.com/square/blip/test"
)

// --------------------------------------------------------------------------

const (
	monitorId1 = "testmon1"
)

func TestLoadDefault(t *testing.T) {
	cfg := blip.DefaultConfig(false)

	pl := plan.NewLoader(nil)
	if err := pl.LoadShared(cfg.Plans, nil); err != nil {
		t.Fatal(err)
	}

	gotPlans := pl.PlansLoaded("")
	expectPlans := []proto.PlanLoaded{
		{
			Name:   blip.INTERNAL_PLAN_NAME,
			Source: "blip",
		},
	}
	assert.Equal(t, gotPlans, expectPlans)
}

func TestLoadOneFile(t *testing.T) {
	file := "../test/plans/version.yaml"
	fileabs, err := filepath.Abs(file)
	if err != nil {
		t.Fatal(err)
	}

	cfg := blip.Config{
		Plans:    blip.ConfigPlans{Files: []string{file}},
		Monitors: []blip.ConfigMonitor{},
	}

	pl := plan.NewLoader(nil)
	if err := pl.LoadShared(cfg.Plans, nil); err != nil {
		t.Fatal(err)
	}

	gotPlans := pl.PlansLoaded("")
	expectPlans := []proto.PlanLoaded{
		{
			Name:   file,
			Source: fileabs,
		},
	}
	assert.Equal(t, gotPlans, expectPlans)
}
