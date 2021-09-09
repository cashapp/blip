package tr

import (
	"regexp"
	"strings"
	"sync"

	prom "github.com/prometheus/client_golang/prometheus"

	"github.com/square/blip"
)

type DomainTranslator interface {
	Translate(values []blip.MetricValue, ch chan<- prom.Metric)
}

func Register(blipDomain string, tr DomainTranslator) error {
	trMux.Lock()
	defer trMux.Unlock()
	trRepo[blipDomain] = tr
	return nil
}

func Translator(domain string) DomainTranslator {
	trMux.Lock()
	defer trMux.Unlock()
	return trRepo[domain]
}

var trMux = &sync.Mutex{}

var trRepo = map[string]DomainTranslator{
	"status.global": StatusGlobal{Domain: "global_status"},
	"var.global":    GenericTr{Domain: "global_variables"},
	"innodb":        InnoDBMetrics{Domain: "info_schema_innodb"},
}

// --------------------------------------------------------------------------
// Copied from /percona/mysqld_exporter/collector/global_status.go
var nameRe = regexp.MustCompile("([^a-zA-Z0-9_])")

func validPrometheusName(s string) string {
	s = nameRe.ReplaceAllString(s, "_")
	s = strings.ToLower(s)
	return s
}

// --------------------------------------------------------------------------

type GenericTr struct {
	Domain string
}

func (tr GenericTr) Translate(values []blip.MetricValue, ch chan<- prom.Metric) {
	for i := range values {
		var promType prom.ValueType
		var help string
		switch values[i].Type {
		case blip.COUNTER:
			promType = prom.CounterValue
			help = "Generic counter metric."
		case blip.GAUGE:
			promType = prom.GaugeValue
			help = "Generic gauge metric."
		}

		ch <- prom.MustNewConstMetric(
			prom.NewDesc(
				prom.BuildFQName("mysql", tr.Domain, validPrometheusName(values[i].Name)),
				help,
				nil, nil,
			),
			promType,
			values[i].Value,
		)
	}
}
