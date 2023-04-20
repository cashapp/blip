// Copyright 2022 Block, Inc.

package sink

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/DataDog/datadog-go/v5/statsd"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"

	"github.com/cashapp/blip"
	"github.com/cashapp/blip/sink/tr"
	"github.com/cashapp/blip/status"
)

var portRe = regexp.MustCompile(`:\d+$`)

const (
	MAX_PAYLOAD_SIZE int = 512000
)

// Datadog sends metrics to Datadog.
type Datadog struct {
	monitorId string
	tags      []string            // monitor.tags (dimensions)
	tr        tr.DomainTranslator // datadog.metric-translator
	prefix    string              // datadog.metric-prefix

	// -- Api
	metricsApi *datadogV2.MetricsApi
	apiKeyAuth string
	appKeyAuth string
	resources  []datadogV2.MetricResource
	compress   bool

	maxMetricsPerRequest     int // Limit the number of metrics we send per request. Only used with the API
	maxMetricsPerRequestLock sync.Mutex
	maxPayloadSize           int

	// -- DogStatsD
	dogstatsd       bool
	dogstatsdClient *statsd.Client
	dogstatsdHost   string
}

func NewDatadog(monitorId string, opts, tags map[string]string, httpClient *http.Client) (*Datadog, error) {
	tagList := make([]string, 0, len(tags))
	var resources []datadogV2.MetricResource = nil

	for k, v := range tags {
		tagList = append(tagList, fmt.Sprintf("%s:%s", k, v))

		// If we have a "host" tag, we should include a resource definition for host
		// so that metrics are properly associated with infrastructure in Datadog
		if k == "host" {
			resources = []datadogV2.MetricResource{
				{
					Name: datadog.PtrString(v),
					Type: datadog.PtrString("host"),
				},
			}
		}
	}

	d := &Datadog{
		monitorId:            monitorId,
		tags:                 tagList,
		resources:            resources,
		maxMetricsPerRequest: math.MaxInt32, // By default don't limit the number of metrics per request.
		compress:             true,
		maxPayloadSize:       MAX_PAYLOAD_SIZE,
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

		case "api-compress":
			d.compress = blip.Bool(v)

		case "dogstatsd-host":
			d.dogstatsdHost = v

		default:
			return nil, fmt.Errorf("invalid option: %s", k)
		}
	}

	if d.dogstatsdHost != "" {
		d.dogstatsd = true
	}

	// if DogStatsD and api are both setup, return error as it will otherwise result in duplicate metrics
	if d.dogstatsd && (d.apiKeyAuth != "" && d.appKeyAuth != "") {
		return nil, fmt.Errorf("datadog sink requires either dogstatsd host or (api-key-auth and app-key-auth), not both at the same time")
	}

	if d.dogstatsd {
		if !portRe.MatchString(d.dogstatsdHost) {
			d.dogstatsdHost += ":8125"
		}
		client, err := statsd.New(d.dogstatsdHost)
		if err != nil {
			return nil, err
		}
		d.dogstatsdClient = client
	} else {
		if d.apiKeyAuth == "" {
			return nil, fmt.Errorf("datadog sink requires either api-key-auth or api-key-auth-file")
		}

		if d.appKeyAuth == "" {
			return nil, fmt.Errorf("datadog sink requires either app-key-auth or app-key-auth-file")
		}

		c := datadog.NewConfiguration()
		c.HTTPClient = httpClient
		c.Compress = d.compress
		metricsApi := datadogV2.NewMetricsApi(datadog.NewAPIClient(c))
		d.metricsApi = metricsApi
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
	var dp []datadogV2.MetricSeries

	if !s.dogstatsd {
		dp = make([]datadogV2.MetricSeries, n)
	}
	blip.Debug("[%s]: datadog: preparing to send %d metrics", s.monitorId, n)
	n = 0

	// Make a copy of maxMetricsPerRequest in case it gets updated by other threads
	localMaxMetricsPerRequest := s.maxMetricsPerRequest
	rangeStart := 0
	var apiErrors []string

	// Setup our context for API calls
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
				if s.dogstatsd {
					err := s.dogstatsdClient.Count(name, int64(metrics[i].Value), tags, 1)
					if err != nil {
						blip.Debug("error sending data points to Datadog: %s", err)
					}
				} else {
					dp[n] = datadogV2.MetricSeries{
						Metric: name,
						Type:   datadogV2.METRICINTAKETYPE_COUNT.Ptr(),
						Points: []datadogV2.MetricPoint{
							{
								Value:     datadog.PtrFloat64(metrics[i].Value),
								Timestamp: datadog.PtrInt64(timestamp),
							},
						},
						Tags:      tags,
						Resources: s.resources,
					}
				}

			case blip.GAUGE, blip.BOOL:
				if s.dogstatsd {
					err := s.dogstatsdClient.Gauge(name, metrics[i].Value, tags, 1)
					if err != nil {
						blip.Debug("error sending data points to Datadog: %s", err)
					}
				} else {
					dp[n] = datadogV2.MetricSeries{
						Metric: name,
						Type:   datadogV2.METRICINTAKETYPE_GAUGE.Ptr(),
						Points: []datadogV2.MetricPoint{
							{
								Value:     datadog.PtrFloat64(metrics[i].Value),
								Timestamp: datadog.PtrInt64(timestamp),
							},
						},
						Tags:      tags,
						Resources: s.resources,
					}
				}

			default:
				// datadog doesn't support this Blip metric type, so skip it
				continue METRICS // @todo error?
			}

			n++

			// Check if we have reached the maximum number of metrics per request
			if !s.dogstatsd && n%localMaxMetricsPerRequest == 0 {
				if err := s.sendApi(ddCtx, dp[rangeStart:n]); err != nil {
					blip.Debug("[%s]: datadog: sending failed with err: %+v", s.monitorId, err)
					apiErrors = append(apiErrors, err.Error())
				}
				rangeStart = n
			}
		} // metric
	} // domain

	// This shouldn't happen: >0 Blip metrics in but =0 Datadog data points out
	if n == 0 {
		return fmt.Errorf("no Datadog data points after processing %d Blip metrics", len(m.Values))
	}

	// dogstatsd metrics are sent to the datadog agent inside the loop, there's nothing else to do
	if s.dogstatsd {
		return nil
	}

	if n-rangeStart > 0 {
		if err := s.sendApi(ddCtx, dp[rangeStart:n]); err != nil {
			blip.Debug("[%s]: datadog: sending failed with err: %+v", s.monitorId, err)
			apiErrors = append(apiErrors, err.Error())
		}
	}

	if len(apiErrors) > 0 {
		return fmt.Errorf("%s", strings.Join(apiErrors, "\n"))
	}

	return nil
}

// Send metrics to the API taking into consideration the number of metrics sent per request.
func (s *Datadog) sendApi(ddCtx context.Context, dp []datadogV2.MetricSeries) error {
	localMaxMetricsPerRequest := s.maxMetricsPerRequest

	for rangeStart := 0; rangeStart < len(dp); {
		// Determine the subset of metrics to send based on our
		// max per request
		rangeEnd := rangeStart + localMaxMetricsPerRequest
		if rangeEnd > len(dp) {
			rangeEnd = len(dp)
		}

		optParams := *datadogV2.NewSubmitMetricsOptionalParameters()
		if s.compress {
			optParams.ContentEncoding = datadogV2.METRICCONTENTENCODING_GZIP.Ptr()
		}
		_, r, err := s.metricsApi.SubmitMetrics(ddCtx, *datadogV2.NewMetricPayload(dp[rangeStart:rangeEnd]), optParams)
		blip.Debug("[%s]: datadog api request status code: %d, response: %s", s.monitorId, r.StatusCode, r)
		var metricNames []string
		for i := rangeStart; rangeStart < rangeEnd; i++ {
			metricNames = append(metricNames, dp[i].Metric)
		}
		blip.Debug("[%s]: datadog api request, sending metrics: %s", strings.Join(metricNames, ","))
		if err != nil {
			if r != nil && r.StatusCode == http.StatusRequestEntityTooLarge {
				// Is the number of metrics sent already the smallest possible?
				if localMaxMetricsPerRequest == 1 {
					return fmt.Errorf("Unable to send metrics: %v", err)
				}

				// The payload was too large, so we need to recalculate it and try with a smaller size
				if localMaxMetricsPerRequest, err = s.estimateMaxMetricsPerRequest(dp[rangeStart:rangeEnd], localMaxMetricsPerRequest); err != nil {
					return fmt.Errorf("unable to determine proper number of metrics per request: %v", err)
				}

				// Retry the metrics with the new payload size
				continue
			}

			blip.Debug("error sending data points to Datadog: %s, Http Response: %s", err, r)
			return err
		}

		rangeStart = rangeEnd
	}

	// Update the maxMetricsPerRequest for the sink
	if localMaxMetricsPerRequest < s.maxMetricsPerRequest {
		s.maxMetricsPerRequestLock.Lock()
		defer s.maxMetricsPerRequestLock.Unlock()
		// Check the value again in case it changed after getting the lock
		if localMaxMetricsPerRequest < s.maxMetricsPerRequest {
			s.maxMetricsPerRequest = localMaxMetricsPerRequest
		}
	}

	return nil
}

// Estimate the number of metrics we can send in a payload based on a sample metric
func (s *Datadog) estimateMaxMetricsPerRequest(metrics []datadogV2.MetricSeries, currentMaxMetricsPerRequest int) (int, error) {
	// Estimate the size of a single metric
	estMetricSize, err := s.estimateSize(metrics)
	if err != nil {
		return 0, err
	}

	// Using our estimated metric size determine out how many metrics can fit inside the max payload, but pad it slightly to control for headers, etc.
	estMaxMetricsPerRequest := (s.maxPayloadSize - 300) / estMetricSize

	if estMaxMetricsPerRequest >= currentMaxMetricsPerRequest {
		// If the estimated maximum is greater than what we currently have set as the maximum then
		// reduce the current maximum by 10% as a guess for finding a maximnum number of metrics
		// to send that will not be rejected by the API.
		estMaxMetricsPerRequest = int(float32(currentMaxMetricsPerRequest) * .9)
	}

	// Ensure we send at least one metric per request
	if estMaxMetricsPerRequest <= 0 {
		estMaxMetricsPerRequest = 1
	}

	return estMaxMetricsPerRequest, nil
}

// Estimate the size of a metric payload for use in determining the maximum number of
// metrics per request. We take the total size of the payload and divide by the number
// of metrics.
func (s *Datadog) estimateSize(metrics []datadogV2.MetricSeries) (int, error) {
	data, err := json.Marshal(metrics)
	if err != nil {
		return 0, err
	}

	size := len(data)

	if s.compress {
		var b bytes.Buffer
		w := zlib.NewWriter(&b)
		w.Write(data)
		w.Close()
		size = len(b.Bytes())
	}

	return size / len(metrics), nil
}

func (s *Datadog) Name() string {
	return "datadog"
}
