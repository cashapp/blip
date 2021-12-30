package monitor

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"

	"github.com/cashapp/blip"
	"github.com/cashapp/blip/prom"
)

// Exporter emulates a Prometheus mysqld_exporter. It implement prom.Exporter.
type Exporter struct {
	cfg    blip.ConfigExporter
	engine *Engine
	// --
	promRegistry *prometheus.Registry
	*sync.Mutex
	levelName string
	prepared  bool
}

func NewExporter(cfg blip.ConfigExporter, engine *Engine) *Exporter {
	e := &Exporter{
		cfg:          cfg,
		engine:       engine,
		promRegistry: prometheus.NewRegistry(),
		Mutex:        &sync.Mutex{},
	}
	e.promRegistry.MustRegister(e)
	return e
}

// --------------------------------------------------------------------------
// Implement Prometheus collector

// Scrape collects and returns metrics in Prometheus exposition format.
// This function is called in response to GET /metrics.
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

func (e *Exporter) Describe(descs chan<- *prometheus.Desc) {
	// Left empty intentionally to make the collector unchecked.
}

var noop = func() {}

// Collect collects metrics. It is called indirectly via Scrpe.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	e.Lock()
	if !e.prepared {
		plan := blip.PromPlan()
		for levelName := range plan.Levels {
			e.levelName = levelName
		}
		if err := e.engine.Prepare(ctx, plan, noop, noop); err != nil {
			blip.Debug(err.Error())
			e.Unlock()
			return
		}
		e.prepared = true
	}
	e.Unlock()

	metrics, err := e.engine.Collect(ctx, e.levelName)
	if err != nil {
		blip.Debug(err.Error())
		// @todo
	}

	for domain, vals := range metrics.Values {
		tr := prom.Translator(domain)
		if tr == nil {
			blip.Debug("no translator registered for %s", domain)
			continue
		}
		tr.Translate(vals, ch)
	}
}
