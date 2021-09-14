package metrics

import (
	"fmt"
	"sync"

	"github.com/square/blip"
	"github.com/square/blip/event"
	"github.com/square/blip/metrics/innodb"
	"github.com/square/blip/metrics/size"
	"github.com/square/blip/metrics/status"
	sysvar "github.com/square/blip/metrics/var"
)

func Register(domain string, f blip.CollectorFactory) error {
	r.Lock()
	defer r.Unlock()
	_, ok := r.factory[domain]
	if ok {
		return fmt.Errorf("%s already registered", domain)
	}
	r.factory[domain] = f
	event.Sendf(event.REGISTER_METRICS, domain)
	return nil
}

func Make(domain string, args blip.CollectorFactoryArgs) (blip.Collector, error) {
	r.Lock()
	defer r.Unlock()
	f, ok := r.factory[domain]
	if !ok {
		return nil, fmt.Errorf("%s not registeres", domain)
	}
	return f.Make(domain, args)
}

// --------------------------------------------------------------------------

func init() {
	for _, mc := range defaultCollectors {
		Register(mc, f)
	}
}

// repo holds registered blip.CollectorFactory
type repo struct {
	*sync.Mutex
	factory map[string]blip.CollectorFactory
}

var r = &repo{
	Mutex:   &sync.Mutex{},
	factory: map[string]blip.CollectorFactory{},
}

type factory struct{}

var f = factory{}

func (f factory) Make(domain string, args blip.CollectorFactoryArgs) (blip.Collector, error) {
	switch domain {
	case "status.global":
		mc := status.NewGlobal(args.DB)
		return mc, nil
	case "var.global":
		mc := sysvar.NewGlobal(args.DB)
		return mc, nil
	case "size.data":
		mc := size.NewData(args.DB)
		return mc, nil
	case "size.binlogs":
		mc := size.NewBinlogs(args.DB)
		return mc, nil
	case "innodb":
		mc := innodb.NewMetrics(args.DB)
		return mc, nil
	}
	return nil, fmt.Errorf("collector for domain %s not registered", domain)
}

var defaultCollectors = []string{
	"status.global",
	"var.global",
	"size.data",
	"size.binlogs",
	"innodb",
}
