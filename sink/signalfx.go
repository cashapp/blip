package sink

import (
	"context"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/signalfx/golib/v3/datapoint"
	"github.com/signalfx/golib/v3/sfxclient"

	"github.com/cashapp/blip"
	"github.com/cashapp/blip/event"
)

// Sink sends metrics to SignalFx.
type sfxSink struct {
	sink        *sfxclient.HTTPSink
	sendTimeout time.Duration
	monitorId   string
	dim         map[string]string
	event       event.MonitorSink
}

func NewSignalFxSink(monitorId string, opts, tags map[string]string) (*sfxSink, error) {
	sink := sfxclient.NewHTTPSink()
	s := &sfxSink{
		sink:      sink,
		event:     event.MonitorSink{MonitorId: monitorId},
		monitorId: monitorId,
		dim:       tags,
	}

	for k, v := range opts {
		switch k {
		case "auth-token-file":
			bytes, err := ioutil.ReadFile(v)
			if err != nil {
				if blip.Strict {
					return nil, err
				}
				// @todo
			} else {
				sink.AuthToken = string(bytes)
			}
		case "auth-token":
			sink.AuthToken = v
		default:
			if blip.Strict {
				return nil, fmt.Errorf("invalid option: %s", k)
			}
		}
	}

	return s, nil
}

func (s *sfxSink) Send(ctx context.Context, m *blip.Metrics) error {
	n := 0
	for _, metrics := range m.Values {
		n += len(metrics)
	}
	dp := make([]*datapoint.Datapoint, n)
	n = 0
	for domain := range m.Values {
		metrics := m.Values[domain]
		for i := range metrics {
			name := domain + "." + metrics[i].Name
			switch metrics[i].Type {
			case blip.COUNTER:
				dp[n] = sfxclient.CumulativeF(name, s.dim, metrics[i].Value)
			case blip.GAUGE:
				dp[n] = sfxclient.GaugeF(name, s.dim, metrics[i].Value)
			}
			n++
		}
	}
	blip.Debug("%s: sent %d metrics", s.monitorId, n)
	return s.sink.AddDatapoints(ctx, dp)
}

func (s *sfxSink) Status() string {
	return ""
}
