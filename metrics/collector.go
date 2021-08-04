package metrics

import (
	"context"
	"database/sql"

	"github.com/square/blip"
	"github.com/square/blip/collect"
	sysvar "github.com/square/blip/metrics/var"
)

type Db struct {
	DB        *sql.DB
	MonitorId string
}

// Collector collects metrics for a single metric domain.
type Collector interface {
	// Domain returns the collector's metric domain name, like "var.global".
	Domain() string

	// Help returns information about using the collector.
	Help() collect.Help

	// Prepare prepares a plan for future calls to Collect.
	Prepare(collect.Plan) error

	// Collect collects metrics for the given in the previously prepared plan.
	Collect(ctx context.Context, levelName string) (collect.Metrics, error)
}

type CollectorFactory interface {
	Make(domain string, monitor blip.Monitor) (Collector, error)
}

var _ CollectorFactory = collectotFactory{}

type collectotFactory struct {
}

func NewCollectorFactory() collectotFactory {
	return collectotFactory{}
}

func (f collectotFactory) Make(domain string, monitor blip.Monitor) (Collector, error) {
	switch domain {
	case "var.global":
		mc := sysvar.NewGlobal(monitor)
		return mc, nil
	}
	return MockCollector{}, nil
}

// --------------------------------------------------------------------------

type MockCollector struct {
}

var _ Collector = MockCollector{}

func (c MockCollector) Domain() string {
	return "mock"
}

func (c MockCollector) Help() collect.Help {
	return collect.Help{}
}

func (c MockCollector) Prepare(collect.Plan) error {
	return nil
}

func (c MockCollector) Collect(ctx context.Context, levelName string) (collect.Metrics, error) {
	return collect.Metrics{}, nil
}
