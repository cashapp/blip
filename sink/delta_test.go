package sink

import (
	"context"
	"testing"
	"time"

	"github.com/cashapp/blip"
	"github.com/cashapp/blip/test/mock"
	"github.com/go-test/deep"
)

func getNoChangesValues() []blip.MetricValue {
	return []blip.MetricValue{
		{
			Name:  "gauge",
			Value: 1.0,
			Type:  blip.GAUGE,
		},
		{
			Name:  "delta",
			Value: 1.0,
			Type:  blip.DELTA_COUNTER,
		},
	}
}

func getChangesValues() []blip.MetricValue {
	return []blip.MetricValue{
		{
			Name:  "counter",
			Value: 1.0,
			Type:  blip.CUMULATIVE_COUNTER,
			Group: map[string]string{
				"a": "1",
			},
		},
		{
			Name:  "counter",
			Value: 1.0,
			Type:  blip.CUMULATIVE_COUNTER,
			Group: map[string]string{
				"a": "2",
			},
		},
		{
			Name:  "delta",
			Value: 1.0,
			Type:  blip.DELTA_COUNTER,
		},
	}
}

func TestDeltaSink(t *testing.T) {
	var returnedResults *blip.Metrics
	mockSink := &mock.Sink{
		SendFunc: func(ctx context.Context, m *blip.Metrics) error {
			returnedResults = m
			return nil
		},
	}

	noChangesValues := getNoChangesValues()
	metrics := &blip.Metrics{
		Begin:     time.Now().Add(-1 * time.Hour),
		End:       time.Now(),
		MonitorId: "testmonitor",
		Plan:      "testplan",
		Level:     "testlevel",
		State:     "teststate",
		Values: map[string][]blip.MetricValue{
			"nochanges": noChangesValues,
			"changes":   getChangesValues(),
		},
	}

	deltaSink := NewDelta(mockSink)
	err := deltaSink.Send(context.Background(), metrics)
	if err != nil {
		t.Error(err)
	}

	// If we had to perform delta calculations then we will get a pointer to a new
	// set of metrics.
	if metrics == returnedResults {
		t.Error("Expected returnedResults and metrics to be different pointers but they matched")
	}

	// The first time we submit cumulative metrics we should not expect to see them
	// in the output as we didn't have enough data points to calculate deltas. The
	// only deltas should be those submitted as delta values.
	expectedMetrics := &blip.Metrics{
		Begin:     metrics.Begin,
		End:       metrics.End,
		MonitorId: "testmonitor",
		Plan:      "testplan",
		Level:     "testlevel",
		State:     "teststate",
		Values: map[string][]blip.MetricValue{
			"nochanges": noChangesValues,
			"changes": {
				{
					Name:  "delta",
					Value: 1.0,
					Type:  blip.DELTA_COUNTER,
				},
			},
		},
	}

	if diff := deep.Equal(expectedMetrics, returnedResults); diff != nil {
		t.Error(diff)
	}

	// Update and send new metrics
	// The cumulative counters should have their values increased
	// so we get valid deltas.
	changesValues := getChangesValues()
	changesValues[0].Value = 2.0
	changesValues[1].Value = 3.0

	metrics = &blip.Metrics{
		Begin:     time.Now().Add(-1 * time.Hour),
		End:       time.Now(),
		MonitorId: "testmonitor",
		Plan:      "testplan",
		Level:     "testlevel",
		State:     "teststate",
		Values: map[string][]blip.MetricValue{
			"nochanges": noChangesValues,
			"changes":   changesValues,
		},
	}

	err = deltaSink.Send(context.Background(), metrics)
	if err != nil {
		t.Error(err)
	}

	if metrics == returnedResults {
		t.Error("Expected returnedResults and metrics to be different pointers but they matched")
	}

	// We should see the newly calculated delta values now that we had a prior run
	expectedMetrics = &blip.Metrics{
		Begin:     metrics.Begin,
		End:       metrics.End,
		MonitorId: "testmonitor",
		Plan:      "testplan",
		Level:     "testlevel",
		State:     "teststate",
		Values: map[string][]blip.MetricValue{
			"nochanges": noChangesValues,
			"changes": {
				{
					Name:  "counter",
					Value: 1.0,
					Type:  blip.DELTA_COUNTER,
					Group: map[string]string{
						"a": "1",
					},
				},
				{
					Name:  "counter",
					Value: 2.0,
					Type:  blip.DELTA_COUNTER,
					Group: map[string]string{
						"a": "2",
					},
				},
				{
					Name:  "delta",
					Value: 1.0,
					Type:  blip.DELTA_COUNTER,
				},
			},
		},
	}

	if diff := deep.Equal(expectedMetrics, returnedResults); diff != nil {
		t.Error(diff)
	}
}

func TestDeltaSink_Passthrough(t *testing.T) {
	var returnedResults *blip.Metrics
	mockSink := &mock.Sink{
		SendFunc: func(ctx context.Context, m *blip.Metrics) error {
			returnedResults = m
			return nil
		},
	}

	metrics := &blip.Metrics{
		Begin:     time.Now().Add(-1 * time.Hour),
		End:       time.Now(),
		MonitorId: "testmonitor",
		Plan:      "testplan",
		Level:     "testlevel",
		State:     "teststate",
		Values: map[string][]blip.MetricValue{
			"nochanges": getNoChangesValues(),
		},
	}

	deltaSink := NewDelta(mockSink)
	err := deltaSink.Send(context.Background(), metrics)
	if err != nil {
		t.Error(err)
	}

	// If the metrics don't contain any cumulative metrics then we should
	// just get the same metrics pointer returned as no transformations needed to happen.
	if metrics != returnedResults {
		t.Error("Expected returnedResults and metrics to be the same pointers but they are different")
	}
}

func TestDelta_NoDeltaSink(t *testing.T) {
	mockSink := mock.Sink{
		SendFunc: func(ctx context.Context, m *blip.Metrics) error {
			return nil
		},
	}

	deltaSink := NewDelta(mockSink)

	func() {
		defer func() {
			if err := recover(); err == nil {
				t.Error("Expected an error but didn't get one")
			}
		}()

		// Create the Retry sink with a Delta sink, which isn't allowed.
		NewDelta(deltaSink)
	}()
}

func TestDelta_NoNilSink(t *testing.T) {
	func() {
		defer func() {
			if err := recover(); err == nil {
				t.Error("Expected an error but didn't get one")
			}
		}()

		// Create the Retry sink with a Delta sink, which isn't allowed.
		NewDelta(nil)
	}()
}
