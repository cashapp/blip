package sink

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/signalfx/golib/v3/datapoint"
	"github.com/signalfx/golib/v3/sfxclient"

	"github.com/cashapp/blip"
	"github.com/cashapp/blip/event"
	"github.com/cashapp/blip/status"
)

// SignalFx sends metrics to SignalFx.
type SignalFx struct {
	sfxSink     *sfxclient.HTTPSink
	sendTimeout time.Duration
	monitorId   string
	dim         map[string]string
	event       event.MonitorReceiver
}

func NewSignalFx(monitorId string, opts, tags map[string]string, httpClient *http.Client) (*SignalFx, error) {
	sfxSink := sfxclient.NewHTTPSink()
	sfxSink.Client = httpClient // made by blip.Factory.HTTPClient

	s := &SignalFx{
		sfxSink:   sfxSink,
		event:     event.MonitorReceiver{MonitorId: monitorId},
		monitorId: monitorId,
		dim:       tags,
	}

	for k, v := range opts {
		switch k {
		case "auth-token-file":
			bytes, err := ioutil.ReadFile(v)
			if err != nil {
				return nil, err
			} else {
				sfxSink.AuthToken = string(bytes)
			}
		case "auth-token":
			sfxSink.AuthToken = v
		default:
			if blip.Strict {
				return nil, fmt.Errorf("invalid option: %s", k)
			}
		}
	}

	return s, nil
}

func (s *SignalFx) Send(ctx context.Context, m *blip.Metrics) error {
	status.Monitor(s.monitorId, "signalfx", "sending metrics")

	n := 0
	defer func() {
		status.Monitor(s.monitorId, "signalfx", "last sent %d metrics at %s", n, time.Now())
	}()

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
	return s.sfxSink.AddDatapoints(ctx, dp)
}

func (s *SignalFx) Name() string {
	return "signalfx"
}
