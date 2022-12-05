// Copyright 2022 Block, Inc.

package sink

import (
	"context"
	"fmt"
	"time"

	"github.com/DataDog/datadog-go/v5/statsd"

	"github.com/cashapp/blip"
	"github.com/cashapp/blip/sink/tr"
	"github.com/cashapp/blip/status"
)

// DogStatsD sends metrics to Datadog agent.
type DogStatsD struct {
	monitorId string
	tags      []string            // monitor.tags (dimensions)
	tr        tr.DomainTranslator // datadog.metric-translator
	prefix    string              // datadog.metric-prefix
	// --
	client *statsd.Client
	host   string
	port   string
}

func NewDogStatsD(monitorId string, opts, tags map[string]string) (*DogStatsD, error) {
	tagList := make([]string, 0, len(tags))

	for k, v := range tags {
		tagList = append(tagList, fmt.Sprintf("%s:%s", k, v))
	}

	d := &DogStatsD{
		monitorId: monitorId,
		tags:      tagList,
	}

	for k, v := range opts {
		switch k {
		case "metric-translator":
			tr, err := tr.Make(v)
			if err != nil {
				return nil, err
			}
			d.tr = tr
		case "metric-prefix":
			if v == "" {
				return nil, fmt.Errorf("dogStatsD sink metric-prefix is empty string; value required when option is specified")
			}
			d.prefix = v
		case "host":
			if v == "" {
				return nil, fmt.Errorf("dogStatsD sink host is empty string; host is required")
			}
			d.host = v
		case "port":
			if v == "" {
				return nil, fmt.Errorf("dogStatsD sink port is empty string; port is required")
			}

		default:
			return nil, fmt.Errorf("invalid option: %s", k)
		}
	}

	client, err := statsd.New(fmt.Sprintf("localhost:8125")) //, d.host, d.port))
	fmt.Println(d.host, d.port)
	if err != nil {
		return nil, err
	}
	d.client = client

	return d, nil
}

func (d *DogStatsD) Send(ctx context.Context, m *blip.Metrics) error {
	status.Monitor(d.monitorId, d.Name(), "sending metrics")

	// On return, set monitor status for this sink
	n := 0
	defer func() {
		status.Monitor(d.monitorId, d.Name(), "last sent %d metrics at %d", n, time.Now())
	}()

	// Pre-alloc DogStatsD data points
	for _, metrics := range m.Values {
		n += len(metrics)
	}
	if n == 0 {
		return fmt.Errorf("no Blip metrics were collected")
	}
	n = 0

	// Convert each Blip metric value to an DogStatsD data point
	for domain := range m.Values { // each domain
		metrics := m.Values[domain]
		var name string

	METRICS:
		for i := range metrics { // each metric in this domain

			// Set full metric name: translator (if any) else Blip standard,
			// then prefix (if any)
			if d.tr == nil {
				name = domain + "." + metrics[i].Name
			} else {
				name = d.tr.Translate(domain, metrics[i].Name)
			}
			if d.prefix != "" {
				name = d.prefix + name
			}

			// Copy metric meta and groups into tags (dimensions), if any
			var tags []string
			if len(metrics[i].Meta) == 0 && len(metrics[i].Group) == 0 {
				// Optimization: if no meta or group, then reuse pointer to
				// d.tags which points to the tags--never modify d.tags!
				tags = d.tags
			} else {
				// There are meta or groups (or both), so we MUST COPY tags
				// from d.tags and the rest into a new map
				tags = make([]string, 0, len(d.tags)+len(metrics[i].Meta)+len(metrics[i].Group))
				for _, v := range d.tags { // copy tags (from config)
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

			// Convert Blip metric type to DogStatsD metric type
			switch metrics[i].Type {
			case blip.COUNTER:
				err := d.client.Count(name, int64(metrics[i].Value), tags, 1)
				if err != nil {
					blip.Debug("error sending data points to Datadog: %s", err)
				}
			case blip.GAUGE, blip.BOOL:
				err := d.client.Gauge(name, metrics[i].Value, tags, 1)
				if err != nil {
					blip.Debug("error sending data points to Datadog: %s", err)
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

	return nil
}

func (d *DogStatsD) Name() string {
	return "dogstatsd"
}
