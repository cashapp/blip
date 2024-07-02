// Copyright 2024 Block, Inc.

package sink

import (
	"context"
	"sort"
	"strings"

	"github.com/cashapp/blip"
)

// The Delta sink calculates DELTER_COUNTER metrics from CUMULATIVE_COUNTER metrics. It acts as a transform,
// removing CUMULATIVE_COUNTER metrics and replacing them with DELTER_COUNTER values. This can be used
// to wrap sinks that expect counters to be submitted as the number/count of observations in the
// sampling interval rather than a cumulative total.
//
// The Delta sink is perferable to peforming delta calculations in the wrapped sink
// as the presence of a Retry sink can cause metrics to be sent to wrapped sink out of order, which
// can cause incorrect metric values to be submitted then delta calculations are performed.
// The Delta sink should never be wrapped inside of a Retry sink to prevent this.
type Delta struct {
	sink     blip.Sink
	counters map[string]float64 // holds last value of the counter so deltas can be calculated
}

var _ blip.Sink = &Delta{}

func NewDelta(sink blip.Sink) *Delta {
	if sink == nil {
		panic("sink is nil; value required")
	}
	if _, ok := sink.(*Delta); ok {
		panic("sink cannot be a Delta sink.")
	}

	return &Delta{
		sink:     sink,
		counters: make(map[string]float64),
	}
}

func (d *Delta) Name() string {
	return "delta"
}

// Calculates DELTA_COUNTER values from any CUMULATIVE_COUNTER values in
// the passed metircs, and then replacees the CUMULATIVE_COUNTER values
// with the new DELTA_COUNTER values. The updated metrics are forwarded
// to the next sink.
//
// This is safe to call from multiple goroutines.
func (d *Delta) Send(ctx context.Context, metrics *blip.Metrics) error {
	newValues := make(map[string][]blip.MetricValue)
	hasNewValues := false

	for key, collection := range metrics.Values {
		valueList := make([]blip.MetricValue, 0, len(collection))
		hasDelta := false

		for _, value := range collection {
			// Calculate a DELTA for any cumulative counters
			switch value.Type {
			case blip.CUMULATIVE_COUNTER:
				hasDelta = true
				metricValue := value.Value
				metricId := d.metricID(value.Name, value.Group)

				val, ok := d.counters[metricId]
				if !ok {
					// If we don't have a prior data point then we should
					// not calculate a delta and just remove the point.
					d.counters[metricId] = value.Value
					continue
				}

				delta := value.Value - val
				d.counters[metricId] = value.Value
				if delta >= 0 {
					metricValue = delta
				} else {
					blip.Debug("found negative delta for: %s (can happen due to restart), sending the potentially partial metric value", value.Name)
				}

				value.Value = metricValue
				value.Type = blip.DELTA_COUNTER
				break

			default:
				break
			}

			valueList = append(valueList, value)
		}

		if hasDelta {
			newValues[key] = valueList
			hasNewValues = true
		} else {
			// If we didn't have to calculate any deltas we can just reuse the existing array
			newValues[key] = collection
		}
	}

	if !hasNewValues {
		// If we didn't have to calculate any deltas then we should
		// just submit the original metrics
		return d.sink.Send(ctx, metrics)
	}

	return d.sink.Send(ctx, &blip.Metrics{
		Begin:     metrics.Begin,
		End:       metrics.End,
		MonitorId: metrics.MonitorId,
		Plan:      metrics.Plan,
		Level:     metrics.Level,
		State:     metrics.State,
		Values:    newValues,
	})
}

// metricID returns the metric name concatenated with sorted group keys.
// For example, if the metric name is "foo" and the group keys are "a" and "b",
// it returns "fooab". This is used to calculate delta counter values in Send.
func (s *Delta) metricID(name string, groups map[string]string) string {
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}

	// sort by keys
	sort.Strings(keys)

	var values []string
	// collect values by sorted keys
	for _, k := range keys {
		values = append(values, groups[k])
	}

	var key string
	key += name
	key += strings.Join(values, "")

	return key
}
