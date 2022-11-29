// Copyright 2022 Block, Inc.

package status_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/cashapp/blip/status"
)

func TestMonitorMulti(t *testing.T) {
	status.Reset()

	// Empty status when nothing has been reported
	got := status.ReportMonitors()
	expect := map[string]map[string]string{}
	assert.Equal(t, got, expect)

	// Report status for same monitor (1) and same component (a), but do it twice to test
	// that both are kept and reported separately
	c1 := status.MonitorMulti("1", "a", "foo")
	c2 := status.MonitorMulti("1", "a", "bar")

	// Each component should have a unique name, which is how the pkg keeps them separate
	if c1 == c2 {
		t.Errorf("monitor component 1 == 2, expected different values: %s == %s", c1, c2)
	}

	// Naming format is currently "component(N)" where N is monotonically increasing int
	if c1 != "a(1)" {
		t.Errorf("component 1 = %s, expected a(1)", c1)
	}
	if c2 != "a(2)" {
		t.Errorf("component 2 = %s, expected a(2)", c2)
	}

	// Now status should report each component separately
	got = status.ReportMonitors()
	expect = map[string]map[string]string{
		"1": {
			c1: "foo",
			c2: "bar",
		},
	}
	assert.Equal(t, expect, got)

	// Removing one component should not affect the other one
	status.RemoveComponent("1", c1)
	got = status.ReportMonitors()
	expect = map[string]map[string]string{
		"1": {
			//c1: "foo", // REMOVED
			c2: "bar",
		},
	}
	assert.Equal(t, expect, got)

	// Removing should be idempotent, so remove c1 again
	status.RemoveComponent("1", c1)
	got = status.ReportMonitors()
	expect = map[string]map[string]string{
		"1": {
			//c1: "foo", // REMOVED
			c2: "bar",
		},
	}
	assert.Equal(t, got, expect)

	// And for completeness, remove all and make sure status is empty again
	status.RemoveComponent("1", c2)
	got = status.ReportMonitors()
	expect = map[string]map[string]string{
		"1": {
			//c1: "foo", // REMOVED
			//c2: "bar", // REMOVED
		},
	}
	assert.Equal(t, expect, got)

	// The 3rd component should be a(3)
	c3 := status.MonitorMulti("1", "a", "three")
	if c3 != "a(3)" {
		t.Errorf("component 3 = %s, expected a(3)", c3)
	}
}
