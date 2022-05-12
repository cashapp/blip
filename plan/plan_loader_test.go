// Copyright 2022 Block, Inc.

package plan

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cashapp/blip"
	//"github.com/cashapp/blip/dbconn"
	"github.com/cashapp/blip/proto"
	//	"github.com/cashapp/blip/test"
)

// --------------------------------------------------------------------------

const (
	monitorId1 = "testmon1"
)

func TestLoadDefault(t *testing.T) {
	cfg := blip.DefaultConfig(false)

	pl := NewLoader(nil)
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

	pl := NewLoader(nil)
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

func TestPlan_ShouldReturnDeepCopyOfPlan(t *testing.T) {
	pl := NewLoader(
		func(blip.ConfigPlans) ([]blip.Plan, error) { return []blip.Plan{}, nil },
	)
	planName := "foobar"
	pl.sharedPlans = []planMeta{
		{
			name: planName,
			plan: blip.Plan{
				Levels: map[string]blip.Level{
					"l1": {
						Name: "l1",
						Freq: "1s",
						Collect: map[string]blip.Domain{
							"d1": {
								Name:    "d1",
								Metrics: []string{"m1"},
							},
						},
					},
					"l2": {
						Name: "l2",
						Freq: "5s",
						Collect: map[string]blip.Domain{
							"d1": {
								Name:    "d1",
								Metrics: []string{"m2"},
							},
						},
					},
				},
			},
		},
	}

	got, err := pl.Plan("", planName, nil)
	require.NoError(t, err)

	expected := pl.sharedPlans[0].plan
	assert.Equal(t, expected, got)
	verifyDeepCopy(t, expected, got)
}

// verifyDeepCopy verifies is a is a deep copy of b by comparing address of the slices and maps in blip.Plan.
// This solution is not ideal, but there really isn't a good way to compare reference object addresses in go.
// The converse is to attempt to mutate the slices and maps embedded within blip.Plan, but even more verbose.
func verifyDeepCopy(t *testing.T, a, b blip.Plan) {
	// Verify top level map.
	assert.NotEqual(t, fmt.Sprintf("%p", a.Levels), fmt.Sprintf("%p", b.Levels))
	for levelKey, aLevel := range a.Levels {
		// Verify domain maps.
		aCollect := aLevel.Collect
		bCollect := b.Levels[levelKey].Collect
		assert.NotEqual(t, fmt.Sprintf("%p", aCollect), fmt.Sprintf("%p", bCollect))
		for aDomainKey, aDomain := range aCollect {
			// Verify slice of metrics to collect.
			bDomain := bCollect[aDomainKey]
			assert.NotEqual(t, fmt.Sprintf("%p", aDomain.Metrics), fmt.Sprintf("%p", bDomain.Metrics))
		}
	}
}
