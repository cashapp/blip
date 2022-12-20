// Copyright 2022 Block, Inc.

package plan_test

import (
	"testing"

	"github.com/go-test/deep"

	"github.com/stretchr/testify/assert"

	"github.com/cashapp/blip"
	"github.com/cashapp/blip/plan"
)

// --------------------------------------------------------------------------

func TestSort(t *testing.T) {
	// The smallest possible plan: 1 level, 1 domain, 1 metric
	p := blip.Plan{
		Name: "test1",
		Levels: map[string]blip.Level{
			"L1": blip.Level{
				Name: "L1",
				Freq: "1s",
				Collect: map[string]blip.Domain{
					"D1": blip.Domain{
						Name: "D1",
						Metrics: []string{
							"M1",
						},
					},
				},
			},
		},
	}
	expectPlan := p // copy
	expectLevels := []plan.SortedLevel{
		{Freq: 1, Name: "L1"},
	}
	gotLevels := plan.Sort(&p)
	assert.Equal(t, expectLevels, gotLevels)
	assert.Equal(t, expectPlan, p)

	// A more realistic plan: 3 levels with increasing freq, same domain
	p = blip.Plan{
		Name: "test1",
		Levels: map[string]blip.Level{
			"L1": blip.Level{
				Name: "L1",
				Freq: "5s",
				Collect: map[string]blip.Domain{
					"D1": blip.Domain{
						Name: "D1",
						Metrics: []string{
							"M1",
						},
					},
				},
			},
			"L2": blip.Level{
				Name: "L2",
				Freq: "10s",
				Collect: map[string]blip.Domain{
					"D1": blip.Domain{
						Name: "D1",
						Metrics: []string{
							"M2",
						},
					},
				},
			},
			"L3": blip.Level{
				Name: "L3",
				Freq: "20s",
				Collect: map[string]blip.Domain{
					"D1": blip.Domain{
						Name: "D1",
						Metrics: []string{
							"M3",
						},
					},
				},
			},
		},
	}
	expectPlan = blip.Plan{
		Name: "test1",
		Levels: map[string]blip.Level{
			"L1": blip.Level{
				Name: "L1",
				Freq: "5s",
				Collect: map[string]blip.Domain{
					"D1": blip.Domain{
						Name: "D1",
						Metrics: []string{
							"M1",
						},
					},
				},
			},
			"L2": blip.Level{
				Name: "L2",
				Freq: "10s",
				Collect: map[string]blip.Domain{
					"D1": blip.Domain{
						Name: "D1",
						Metrics: []string{
							"M2",
							"M1",
						},
					},
				},
			},
			"L3": blip.Level{
				Name: "L3",
				Freq: "20s",
				Collect: map[string]blip.Domain{
					"D1": blip.Domain{
						Name: "D1",
						Metrics: []string{
							"M3",
							"M2",
							"M1",
						},
					},
				},
			},
		},
	}
	expectLevels = []plan.SortedLevel{
		{Freq: 5, Name: "L1"},
		{Freq: 10, Name: "L2"},
		{Freq: 20, Name: "L3"},
	}
	gotLevels = plan.Sort(&p)
	assert.Equal(t, expectLevels, gotLevels)
	assert.Equal(t, expectPlan, p)

	// A plan for options and errors: 2 levels, same domain
	p = blip.Plan{
		Name: "test1",
		Levels: map[string]blip.Level{
			"L1": blip.Level{
				Name: "L1",
				Freq: "5s",
				Collect: map[string]blip.Domain{
					"D1": blip.Domain{
						Name: "D1",
						Metrics: []string{
							"M1",
						},
						Options: map[string]string{
							"option1": "1",
							"option2": "2",
						},
						Errors: map[string]string{
							"error1": "1",
							"error2": "2",
						},
					},
				},
			},
			"L2": blip.Level{
				Name: "L2",
				Freq: "10s",
				Collect: map[string]blip.Domain{
					"D1": blip.Domain{
						Name: "D1",
						Metrics: []string{
							"M2",
						},
						Options: map[string]string{
							"option1": "1-1",
						},
						Errors: map[string]string{
							"error1": "1-1",
						},
					},
				},
			},
		},
	}
	expectPlan = blip.Plan{
		Name: "test1",
		Levels: map[string]blip.Level{
			"L1": blip.Level{
				Name: "L1",
				Freq: "5s",
				Collect: map[string]blip.Domain{
					"D1": blip.Domain{
						Name: "D1",
						Metrics: []string{
							"M1",
						},
						Options: map[string]string{
							"option1": "1",
							"option2": "2",
						},
						Errors: map[string]string{
							"error1": "1",
							"error2": "2",
						},
					},
				},
			},
			"L2": blip.Level{
				Name: "L2",
				Freq: "10s",
				Collect: map[string]blip.Domain{
					"D1": blip.Domain{
						Name: "D1",
						Metrics: []string{
							"M2",
							"M1",
						},
						Options: map[string]string{
							"option1": "1-1", // Use option in L2
							"option2": "2",
						},
						Errors: map[string]string{
							"error1": "1-1", // Use error in L2
							"error2": "2",
						},
					},
				},
			},
		},
	}
	expectLevels = []plan.SortedLevel{
		{Freq: 5, Name: "L1"},
		{Freq: 10, Name: "L2"},
	}
	gotLevels = plan.Sort(&p)
	assert.Equal(t, expectLevels, gotLevels)
	assert.Equal(t, expectPlan, p)
}

func TestSortComplex(t *testing.T) {
	p := blip.Plan{
		Name: "test1",
		Levels: map[string]blip.Level{
			"L1": {
				Name: "L1",
				Freq: "5s",
				Collect: map[string]blip.Domain{
					"D1": {
						Name: "D1",
						Metrics: []string{
							"D1_M1",
						},
						Options: map[string]string{},
						Errors:  map[string]string{},
					},
				},
			},
			"L2": {
				Name: "L2",
				Freq: "20s",
				Collect: map[string]blip.Domain{
					"D1": {
						Name: "D1",
						Metrics: []string{
							"D1_M2",
						},
						Options: map[string]string{},
						Errors:  map[string]string{},
					},
				},
			},
			"L3": {
				Name: "L3",
				Freq: "30s",
				Collect: map[string]blip.Domain{
					"D2": {
						Name: "D2",
						Metrics: []string{
							"D2_M1",
						},
						Options: map[string]string{},
						Errors:  map[string]string{},
					},
				},
			},
			"L4": {
				Name: "L4",
				Freq: "60s",
				Collect: map[string]blip.Domain{
					"D3": {
						Name: "D3",
						Metrics: []string{
							"D3_M1",
						},
						Options: map[string]string{
							"option1": "1",
						},
						Errors: map[string]string{
							"error1": "1",
						},
					},
				},
			},
			"L5": {
				Name: "L5",
				Freq: "300s",
				Collect: map[string]blip.Domain{
					"D4": {
						Name: "D4",
						Metrics: []string{
							"D4_M1",
						},
						Options: map[string]string{},
						Errors:  map[string]string{},
					},
				},
			},
		},
	}
	expectPlan := blip.Plan{
		Name: "test1",
		Levels: map[string]blip.Level{
			"L1": {
				Name: "L1",
				Freq: "5s",
				Collect: map[string]blip.Domain{
					"D1": {
						Name: "D1",
						Metrics: []string{
							"D1_M1",
						},
						Options: map[string]string{},
						Errors:  map[string]string{},
					},
				},
			},
			"L2": {
				Name: "L2",
				Freq: "20s",
				Collect: map[string]blip.Domain{
					"D1": {
						Name: "D1",
						Metrics: []string{
							"D1_M2", // This level
							"D1_M1", // L1
						},
						Options: map[string]string{},
						Errors:  map[string]string{},
					},
				},
			},
			"L3": {
				Name: "L3",
				Freq: "30s",
				Collect: map[string]blip.Domain{
					"D2": { // This level
						Name: "D2",
						Metrics: []string{
							"D2_M1",
						},
						Options: map[string]string{},
						Errors:  map[string]string{},
					},
					"D1": { // L1, not L2 because this level 30s mod L2 20s != 0
						Name: "D1",
						Metrics: []string{
							"D1_M1",
						},
						Options: map[string]string{},
						Errors:  map[string]string{},
					},
				},
			},
			"L4": {
				Name: "L4",
				Freq: "60s",
				Collect: map[string]blip.Domain{
					"D3": { // This level
						Name: "D3",
						Metrics: []string{
							"D3_M1",
						},
						Options: map[string]string{
							"option1": "1",
						},
						Errors: map[string]string{
							"error1": "1",
						},
					},
					"D1": { // L1 + L2
						Name: "D1",
						Metrics: []string{
							"D1_M2",
							"D1_M1",
						},
						Options: map[string]string{},
						Errors:  map[string]string{},
					},
					"D2": { // L3 (30s)
						Name: "D2",
						Metrics: []string{
							"D2_M1",
						},
						Options: map[string]string{},
						Errors:  map[string]string{},
					},
				},
			},
			"L5": {
				Name: "L5",
				Freq: "300s",
				Collect: map[string]blip.Domain{
					"D4": { // This level
						Name: "D4",
						Metrics: []string{
							"D4_M1",
						},
						Options: map[string]string{},
						Errors:  map[string]string{},
					},
					"D1": { // L1 + L2
						Name: "D1",
						Metrics: []string{
							"D1_M2",
							"D1_M1",
						},
						Options: map[string]string{},
						Errors:  map[string]string{},
					},
					"D2": { // L3 (30s)
						Name: "D2",
						Metrics: []string{
							"D2_M1",
						},
						Options: map[string]string{},
						Errors:  map[string]string{},
					},
					"D3": { // L4 (60s)
						Name: "D3",
						Metrics: []string{
							"D3_M1",
						},
						Options: map[string]string{
							"option1": "1",
						},
						Errors: map[string]string{
							"error1": "1",
						},
					},
				},
			},
		},
	}
	expectLevels := []plan.SortedLevel{
		{Freq: 5, Name: "L1"},
		{Freq: 20, Name: "L2"},
		{Freq: 30, Name: "L3"},
		{Freq: 60, Name: "L4"},
		{Freq: 300, Name: "L5"},
	}
	gotLevels := plan.Sort(&p)
	assert.Equal(t, expectLevels, gotLevels)
	if diff := deep.Equal(p, expectPlan); diff != nil {
		t.Error(diff)
	}
}
