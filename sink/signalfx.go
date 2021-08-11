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
	buff        *RetryBuffer
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
	err := s.sink.AddDatapoints(ctx, []*datapoint.Datapoint{
		sfxclient.GaugeF("a.gauge", nil, 1.2),
		sfxclient.Cumulative("a.counter", map[string]string{"type": "dev"}, 100),
	})
	return err
}

func (s *sfxSink) Status() error {
	return nil
}

func (s *sfxSink) Name() string {
	return "signalfx"
}
