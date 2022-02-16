// Copyright 2022 Block, Inc.

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
	"github.com/cashapp/blip/sink/tr"
	"github.com/cashapp/blip/status"
)

// SignalFx sends metrics to SignalFx.
type SignalFx struct {
	monitorId string
	dim       map[string]string   // monitor.tags (dimensions)
	tr        tr.DomainTranslator // signalfx.metric-translator
	prefix    string              // signalfx.metric-prefix
	// --
	sfxSink *sfxclient.HTTPSink
}

func NewSignalFx(monitorId string, opts, tags map[string]string, httpClient *http.Client) (*SignalFx, error) {
	sfxSink := sfxclient.NewHTTPSink()
	sfxSink.Client = httpClient // made by blip.Factory.HTTPClient

	s := &SignalFx{
		sfxSink:   sfxSink,
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
		case "metric-translator":
			tr, err := tr.Make(v)
			if err != nil {
				return nil, err
			}
			s.tr = tr
		case "metric-prefix":
			if v == "" {
				return nil, fmt.Errorf("signalfx sink metric-prefix is empty string; value required when option is specified")
			}
			s.prefix = v
		default:
			if blip.Strict {
				return nil, fmt.Errorf("invalid option: %s", k)
			}
		}
	}

	if sfxSink.AuthToken == "" {
		return nil, fmt.Errorf("signalfx sink requires either auth-token or auth-token-file")
	}

	return s, nil
}

func (s *SignalFx) Send(ctx context.Context, m *blip.Metrics) error {
	status.Monitor(s.monitorId, "signalfx", "sending metrics")

	// On return, set monitor status for this sink
	n := 0
	defer func() {
		status.Monitor(s.monitorId, "signalfx", "last sent %d metrics at %s", n, time.Now())
	}()

	// Pre-alloc SFX data points
	for _, metrics := range m.Values {
		n += len(metrics)
	}
	if n == 0 {
		return fmt.Errorf("no Blip metrics were collected")
	}
	dp := make([]*datapoint.Datapoint, n)
	n = 0

	// Convert each Blip metric value to an SFX data point
	for domain := range m.Values { // each domain
		metrics := m.Values[domain]
		var name string

	METRICS:
		for i := range metrics { // each metric in this domain

			// Set full metric name: translator (if any) else Blip standard,
			// then prefix (if any)
			if s.tr == nil {
				name = domain + "." + metrics[i].Name
			} else {
				name = s.tr.Translate(domain, metrics[i].Name)
			}
			if s.prefix != "" {
				name = s.prefix + name
			}

			// Convert Blip metric type to SFX metric type
			switch metrics[i].Type {
			case blip.COUNTER:
				dp[n] = sfxclient.CumulativeF(name, s.dim, metrics[i].Value)
			case blip.GAUGE:
				dp[n] = sfxclient.GaugeF(name, s.dim, metrics[i].Value)
			default:
				// SFX doesn't support this Blip metric type, so skip it
				continue METRICS // @todo error?
			}

			n++
		} // metric
	} // domain

	// This shouldn't happen: >0 Blip metrics in but =0 SFX data points out
	if n == 0 {
		return fmt.Errorf("no SignalFx data points after processing %d Blip metrics", len(m.Values))
	}

	// Send metrics to SFX. The SFX client handles everything; we just pass
	// it data points.
	return s.sfxSink.AddDatapoints(ctx, dp[0:n])
}

func (s *SignalFx) Name() string {
	return "signalfx"
}
