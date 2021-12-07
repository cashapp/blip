package prom

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/square/blip"
	"github.com/square/blip/prom/tr"
)

type DomainTranslator interface {
	Translate(values []blip.MetricValue, ch chan<- prometheus.Metric)

	Names() (prefix, domain, shortDomin string)
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
	"status.global": tr.StatusGlobal{Domain: "global_status", ShortDomain: "status"},
	"var.global":    tr.Generic{Domain: "global_variables", ShortDomain: "var"},
	"innodb":        tr.InnoDBMetrics{Domain: "info_schema_innodb", ShortDomain: "innodb"},
}
