package prom

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"

	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"

	"github.com/square/blip"
	"github.com/square/blip/collect"
	"github.com/square/blip/metrics"
	//"github.com/square/blip/status"
)

// Exporter is a pseudo-sink that emulates mysqld_exporter.
type Exporter struct {
	monitorId    string
	db           *sql.DB
	mcMaker      metrics.CollectorFactory
	promRegistry *prom.Registry
	mcList       map[string]metrics.Collector // keyed on domain
	levelName    string
}

func NewExporter(monitorId string, db *sql.DB, mcMaker metrics.CollectorFactory) *Exporter {
	e := &Exporter{
		monitorId:    monitorId,
		db:           db,
		mcMaker:      mcMaker,
		promRegistry: prom.NewRegistry(),
		mcList:       map[string]metrics.Collector{},
	}
	e.promRegistry.MustRegister(e)
	return e
}

func (e *Exporter) Status() error {
	return nil
}

func (e *Exporter) Prepare(plan collect.Plan) error {
	if len(plan.Levels) != 1 {
		return fmt.Errorf("multiple levels not supported")
	}
	for levelName, level := range plan.Levels {
		e.levelName = levelName

		for domain := range level.Collect {

			// Make collector if needed
			mc, ok := e.mcList[domain]
			if !ok {
				var err error
				mc, err = e.mcMaker.Make(
					domain,
					metrics.FactoryArgs{
						MonitorId: e.monitorId,
						DB:        e.db,
					},
				)
				if err != nil {
					return err // @todo
				}
				e.mcList[domain] = mc
			}

			if err := mc.Prepare(plan); err != nil {
				blip.Debug("%s: mc.Prepare error: %s", e.monitorId, err)
				return err // @todo
			}

		}
	}

	return nil
}

func (e *Exporter) Scrape() (string, error) {
	// Gather calls the Collect method of the exporter
	mfs, err := e.promRegistry.Gather()
	if err != nil {
		return "", fmt.Errorf("Unable to convert blip metrics to Prom metrics. Error: %s", err)
	}

	// Converts the MetricFamily protobufs to prom text format.
	var buf bytes.Buffer
	for _, mf := range mfs {
		expfmt.MetricFamilyToText(&buf, mf)
	}

	return buf.String(), nil
}

// --------------------------------------------------------------------------
// Implement Promtheus collector

func (e *Exporter) Describe(descs chan<- *prom.Desc) {
	// Left empty intentionally to make the collector unchecked.
}

func (e *Exporter) Collect(ch chan<- prom.Metric) {
	for i := range e.mcList {
		values, err := e.mcList[i].Collect(context.Background(), e.levelName)
		if err != nil {
			blip.Debug(err.Error())
			// @todo
		}
		_, domain := e.mcList[i].Domain()
		blip.Debug("collecting %s.......................", domain)
		for i := range values {
			switch values[i].Type {
			case blip.COUNTER:
				promMetric, err := prom.NewConstMetric(
					desc(domain, validPrometheusName(values[i].Name), "Generic counter metric"),
					prom.CounterValue,
					values[i].Value,
				)
				if err != nil {
					log.Printf("Error converting blip metric to prom metric. metricname:%s, type:%b: %s", values[i].Name, values[i].Type, err)
					continue
				}
				ch <- promMetric
			case blip.GAUGE:
				promMetric, err := prom.NewConstMetric(
					desc(domain, validPrometheusName(values[i].Name), "Generic gauge metric"),
					prom.GaugeValue,
					values[i].Value,
				)
				if err != nil {
					log.Printf("Error converting blip metric to prom metric. metricname:%s, type:%b: %s", values[i].Name, values[i].Type, err)
					continue
				}
				ch <- promMetric
			default:
				log.Printf("Unknown metric type found. metricname: %s, type: %b: %s", values[i].Name, values[i].Type, err)
			}
		}
	}
}

func desc(subsystem, name, help string) *prom.Desc {
	return prom.NewDesc(prom.BuildFQName("mysql", subsystem, name), help, nil, nil)
}

func validPrometheusName(s string) string {
	return strings.ToLower(s)
}
