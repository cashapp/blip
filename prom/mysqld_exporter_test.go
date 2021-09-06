package prom

import (
	"github.com/square/blip"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTransformToPromTxtFmt(t *testing.T) {

	// Creates a new exporter with some domain mapped Blip Metrics and
	// invokes `TransformToPromTxtFmt` to verify that Blip Metrics
	// get converted to prometheus exposition text format successfully.

	var blipMetrics = map[string][]blip.MetricValue{}
	blipMetrics["global_variables"] = append(blipMetrics["global_variables"], blip.MetricValue{
		Name:  "max_connections",
		Value: 512,
		Type:  blip.GAUGE,
	})
	blipMetrics["global_status"] = append(blipMetrics["global_status"], blip.MetricValue{
		Name:  "performance_schema_lost_total",
		Value: 5,
		Type:  blip.COUNTER,
	})

	expectedMetricOutput := `# HELP mysql_global_status_performance_schema_lost_total Generic counter metric
# TYPE mysql_global_status_performance_schema_lost_total counter
mysql_global_status_performance_schema_lost_total 5
# HELP mysql_global_variables_max_connections Generic gauge metric
# TYPE mysql_global_variables_max_connections gauge
mysql_global_variables_max_connections 512
`
	var exporter = NewSink(blipMetrics)
	buf, err := exporter.TransformToPromTxtFmt()
	assert.Nil(t, err)
	assert.Equal(t, expectedMetricOutput, buf.String())
}
