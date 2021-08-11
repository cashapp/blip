package sink

import (
	"context"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/signalfx/golib/v3/datapoint"
	"github.com/signalfx/golib/v3/sfxclient"

	"github.com/square/blip"
)

// Sink sends metrics to SignalFx.
type sfxSink struct {
	sink        *sfxclient.HTTPSink
	sendTimeout time.Duration
	dim         map[string]string
}

func NewSignalFxSink(opts map[string]string) (*sfxSink, error) {
	sink := sfxclient.NewHTTPSink()
	s := &sfxSink{
		sink: sink,
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
		case "send-timeout":
			d, err := time.ParseDuration(v)
			if err != nil {
				return nil, err // @todo
			}
			s.sendTimeout = d
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
	for metrics := range m.Values {
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
	return s.sink.AddDatapoints(ctx, dp)
}

func (s *sfxSink) Status() error {
	return nil
}

func (s *sfxSink) Name() string {
	return "signalfx"
}
