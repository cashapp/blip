package prom

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"

	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"

	"github.com/square/blip"
	"github.com/square/blip/metrics"
	"github.com/square/blip/prom/tr"
)

// Exporter is a pseudo-sink that emulates mysqld_exporter.
type Exporter struct {
	monitorId    string
	db           *sql.DB
	promRegistry *prom.Registry
	mcList       map[string]blip.Collector // keyed on domain
	levelName    string
}

func NewExporter(monitorId string, db *sql.DB) *Exporter {
	e := &Exporter{
		monitorId:    monitorId,
		db:           db,
		promRegistry: prom.NewRegistry(),
		mcList:       map[string]blip.Collector{},
	}
	e.promRegistry.MustRegister(e)
	return e
}

func (e *Exporter) Status() error {
	return nil
}

func (e *Exporter) Prepare(plan blip.Plan) error {
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
				mc, err = metrics.Make(
					domain,
					blip.CollectorFactoryArgs{
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
		domain := e.mcList[i].Domain()
		tr := tr.Translator(domain)
		if tr == nil {
			blip.Debug("no translator registered for %s", domain)
			continue
		}
		tr.Translate(values, ch)
	}
}
