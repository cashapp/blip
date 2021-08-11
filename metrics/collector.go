package metrics

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"github.com/square/blip"
	"github.com/square/blip/collect"
	"github.com/square/blip/event"
	sysvar "github.com/square/blip/metrics/var"
)

// Collector collects metrics for a single metric domain.
type Collector interface {
	// Domain returns the collector's metric domain domain, like "var.global".
	Domain() string

	// Help returns information about using the collector.
	Help() collect.Help

	// Prepare prepares a plan for future calls to Collect.
	Prepare(collect.Plan) error

	// Collect collects metrics for the given in the previously prepared plan.
	Collect(ctx context.Context, levelName string) ([]blip.MetricValue, error)
}

type FactoryArgs struct {
	MonitorId string
	DB        *sql.DB
}

type CollectorFactory interface {
	Make(domain string, args FactoryArgs) (Collector, error)
}

type factory struct {
}

func (f factory) Make(domain string, args FactoryArgs) (Collector, error) {
	switch domain {
	case "var.global":
		mc := sysvar.NewGlobal(args.DB)
		return mc, nil
	}
	return nil, fmt.Errorf("invalid domain")
}

var DefaultFactory = factory{}

func RegisterDefaults() {
	Register("var.global", DefaultFactory)
	Register("status.global", DefaultFactory)
}

// --------------------------------------------------------------------------

type repo struct {
	*sync.Mutex
	factory map[string]CollectorFactory
}

var collectorRepo = &repo{
	Mutex:   &sync.Mutex{},
	factory: map[string]CollectorFactory{},
}

func Register(domain string, f CollectorFactory) error {
	collectorRepo.Lock()
	defer collectorRepo.Unlock()
	_, ok := collectorRepo.factory[domain]
	if ok {
		return fmt.Errorf("%s already registered", domain)
	}
	collectorRepo.factory[domain] = f
	event.Sendf(event.REGISTER_METRICS, domain)
	return nil
}

func Make(domain string, args FactoryArgs) (Collector, error) {
	collectorRepo.Lock()
	defer collectorRepo.Unlock()
	f, ok := collectorRepo.factory[domain]
	if !ok {
		return nil, fmt.Errorf("%s not registeres", domain)
	}
	return f.Make(domain, args)
}
