package prom

import (
	"bytes"
	"fmt"
	"log"
	"regexp"
	"strings"

	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"

	"github.com/square/blip"
)

// Exporter is a pseudo-sink that emulates mysqld_exporter.
type Exporter struct {
	domainMappedBlipMetrics map[string][]blip.MetricValue
	registry                *prom.Registry
}

func NewSink(domainMappedBlipMetrics map[string][]blip.MetricValue) *Exporter {
	return &Exporter{
		domainMappedBlipMetrics: domainMappedBlipMetrics,
		registry:                prom.NewRegistry(),
	}
}

func (s *Exporter) Status() error {
	return nil
}

func newDesc(subsystem, name, help string) *prom.Desc {
	return prom.NewDesc(
		prom.BuildFQName("mysql", subsystem, name),
		help, nil, nil,
	)
}

func validPrometheusName(s string) string {
	nameRe := regexp.MustCompile("([^a-zA-Z0-9_])")
	s = nameRe.ReplaceAllString(s, "_")
	s = strings.ToLower(s)
	return s
}

func (e *Exporter) Describe(descs chan<- *prom.Desc) {
	// Left empty intentionally to make the collector unchecked.
}

func (e *Exporter) Collect(ch chan<- prom.Metric) {
	for domain, blipMetrics := range e.domainMappedBlipMetrics {
		for _, bm := range blipMetrics {
			switch bm.Type {
			case blip.COUNTER:
				promMetric, err := prom.NewConstMetric(
					newDesc(domain, validPrometheusName(bm.Name), "Generic counter metric"),
					prom.CounterValue,
					bm.Value,
				)
				if err != nil {
					log.Printf("Error converting blip metric to prom metric. metricname:%s, type:%b", bm.Name, bm.Type)
					continue
				}
				ch <- promMetric
			case blip.GAUGE:
				promMetric, err := prom.NewConstMetric(
					newDesc(domain, validPrometheusName(bm.Name), "Generic gauge metric"),
					prom.GaugeValue,
					bm.Value,
				)
				if err != nil {
					log.Printf("Error converting blip metric to prom metric. metricname:%s, type:%b", bm.Name, bm.Type)
					continue
				}
				ch <- promMetric
			default:
				log.Printf("Unknown metric type found. metricname: %s, type: %b", bm.Name, bm.Type)
			}
		}
	}
}

func (s *Exporter) TransformToPromTxtFmt() (bytes.Buffer, error) {
	var buf bytes.Buffer

	s.registry.MustRegister(s)
	// Gather calls the Collect method of the exporter
	mfs, err := s.registry.Gather()

	if err != nil {
		return bytes.Buffer{}, fmt.Errorf("Unable to convert blip metrics to Prom metrics. Error: %s", err)
	}

	for _, mf := range mfs {
		// Converts the MetricFamily protobufs to prom text format.
		expfmt.MetricFamilyToText(&buf, mf)
	}
	return buf, nil
}
