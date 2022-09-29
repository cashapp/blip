package sink

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"

	"github.com/cashapp/blip"
	"github.com/cashapp/blip/sink/tr"
	"github.com/cashapp/blip/status"
)

// SignalFx sends metrics to SignalFx.
type Datadog struct {
	monitorId string
	tags      []string            // monitor.tags (dimensions)
	tr        tr.DomainTranslator // datadog.metric-translator
	prefix    string              // datadog.metric-prefix
	// --
	metricsApi *datadogV2.MetricsApi
	apiKeyAuth string
	appKeyAuth string
}

func NewDatadog(monitorId string, opts, tags map[string]string, httpClient *http.Client) (*Datadog, error) {
	c := datadog.NewConfiguration()
	c.HTTPClient = httpClient
	metricsApi := datadogV2.NewMetricsApi(datadog.NewAPIClient(c))

	tagList := make([]string, 0, len(tags))

	for k, v := range tags {
		tagList = append(tagList, fmt.Sprintf("%s:%s", k, v))
	}

	d := &Datadog{
		monitorId:  monitorId,
		tags:       tagList,
		metricsApi: metricsApi,
	}

	for k, v := range opts {
		switch k {
		case "api-key-auth":
			d.apiKeyAuth = v

		case "api-key-auth-file":
			bytes, err := ioutil.ReadFile(v)
			if err != nil {
				return nil, err
			} else {
				d.apiKeyAuth = string(bytes)
			}

		case "app-key-auth":
			d.appKeyAuth = v

		case "app-key-auth-file":
			bytes, err := ioutil.ReadFile(v)
			if err != nil {
				return nil, err
			} else {
				d.appKeyAuth = string(bytes)
			}

		case "metric-translator":
			tr, err := tr.Make(v)
			if err != nil {
				return nil, err
			}
			d.tr = tr
		case "metric-prefix":
			if v == "" {
				return nil, fmt.Errorf("datadog sink metric-prefix is empty string; value required when option is specified")
			}
			d.prefix = v

		default:
			return nil, fmt.Errorf("invalid option: %s", k)
		}
	}

	if d.apiKeyAuth == "" {
		return nil, fmt.Errorf("datadog sink required either api-key-auth or api-key-auth-file")
	}

	if d.appKeyAuth == "" {
		return nil, fmt.Errorf("datadog sink required either app-key-auth or app-key-auth-file")
	}

	return d, nil
}

func (s *Datadog) Send(ctx context.Context, m *blip.Metrics) error {
	status.Monitor(s.monitorId, s.Name(), "sending metrics")

	// On return, set monitor status for this sink
	n := 0
	defer func() {
		status.Monitor(s.monitorId, s.Name(), "last sent %d metrics at %s", n, time.Now())
	}()

	// Pre-alloc SFX data points
	for _, metrics := range m.Values {
		n += len(metrics)
	}
	if n == 0 {
		return fmt.Errorf("no Blip metrics were collected")
	}
	dp := make([]datadogV2.MetricSeries, n)
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

			// Copy metric meta and groups into tags (dimensions), if any
			var tags []string
			if len(metrics[i].Meta) == 0 && len(metrics[i].Group) == 0 {
				// Optimization: if no meta or group, then reuse pointer to
				// s.tags which points to the tags--never modify s.tags!
				tags = s.tags
			} else {
				// There are meta or groups (or both), so we MUST COPY tags
				// from s.tags and the rest into a new map
				tags = make([]string, 0, len(s.tags)+len(metrics[i].Meta)+len(metrics[i].Group))
				for _, v := range s.tags { // copy tags (from config)
					tags = append(tags, v)
				}

				for k, v := range metrics[i].Meta { // metric meta
					if k == "ts" { // avoid time series explosion: ts is high cardinality
						continue
					}
					tags = append(tags, fmt.Sprintf("%s:%s", k, v))
				}

				for k, v := range metrics[i].Group { // metric groups
					tags = append(tags, fmt.Sprintf("%s:%s", k, v))
				}
			}

			var timestamp int64

			// Datadog requires a timestamp when creating a data point
			if tsStr, ok := metrics[i].Meta["ts"]; !ok {
				timestamp = m.Begin.Unix()
			} else {
				var err error
				timestamp, err = strconv.ParseInt(tsStr, 10, 64) // ts in milliseconds, string -> int64
				if err != nil {
					blip.Debug("invalid timestamp for %s %s: %s: %s", domain, metrics[i].Name, tsStr, err)
					continue METRICS
				}
			}

			// Convert Blip metric type to Datadog metric type
			switch metrics[i].Type {
			case blip.COUNTER:
				dp[n] = datadogV2.MetricSeries{
					Metric: name,
					Type:   datadogV2.METRICINTAKETYPE_COUNT.Ptr(),
					Points: []datadogV2.MetricPoint{
						{
							Value:     datadog.PtrFloat64(metrics[i].Value),
							Timestamp: datadog.PtrInt64(timestamp),
						},
					},
					Tags: tags,
				}
			case blip.GAUGE:
				dp[n] = datadogV2.MetricSeries{
					Metric: name,
					Type:   datadogV2.METRICINTAKETYPE_GAUGE.Ptr(),
					Points: []datadogV2.MetricPoint{
						{
							Value:     datadog.PtrFloat64(metrics[i].Value),
							Timestamp: datadog.PtrInt64(timestamp),
						},
					},
					Tags: tags,
				}
			default:
				// datadog doesn't support this Blip metric type, so skip it
				continue METRICS // @todo error?
			}

			n++
		} // metric
	} // domain

	// This shouldn't happen: >0 Blip metrics in but =0 Datadog data points out
	if n == 0 {
		return fmt.Errorf("no Datadog data points after processing %d Blip metrics", len(m.Values))
	}

	ddCtx := context.WithValue(
		ctx,
		datadog.ContextAPIKeys,
		map[string]datadog.APIKey{
			"apiKeyAuth": {
				Key: s.apiKeyAuth,
			},
			"appKeyAuth": {
				Key: s.apiKeyAuth,
			},
		},
	)

	payload := datadogV2.MetricPayload{
		Series: dp,
	}

	// Send metrics to Datadog. The Datadog client handles everything; we just pass
	// it data points.
	_, r, err := s.metricsApi.SubmitMetrics(ddCtx, payload, *datadogV2.NewSubmitMetricsOptionalParameters())
	if err != nil {
		blip.Debug("error sending data points to Datadog: %s", err)
		blip.Debug("error sending data points to Datadog: Http Response - %s", r)
	}

	return err
}

func (s *Datadog) Name() string {
	return "datadog"
}
