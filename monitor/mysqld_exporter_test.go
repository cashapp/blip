package monitor_test

import (
	"strings"
	"testing"

	"github.com/cashapp/blip"
	"github.com/cashapp/blip/monitor"
)

func TestProm(t *testing.T) {
	// Test that the Prometheus-emulating Exporter scrapes from MySQL
	exp := monitor.NewExporter(
		blip.ConfigExporter{},
		monitor.NewEngine(blip.ConfigMonitor{MonitorId: monitorId1}, db),
	)

	got, err := exp.Scrape()
	if err != nil {
		t.Error(err)
	}

	// The real output is very long, but we can be certain the basic mechanics
	// are working if it continas this exact snippet, which means it must have
	// successfully queried MySQL and printed the output (the metrics in Exposition
	// format native to Prometheus, not Blip).
	expect := `# HELP mysql_global_variables_port Generic gauge metric.
# TYPE mysql_global_variables_port gauge
mysql_global_variables_port 3306`

	if !strings.Contains(got, expect) {
		t.Errorf("output does not contain:\n%s\n\n%s\n", expect, got)
	}
}
