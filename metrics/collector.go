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

// Register registers a factory that makes one or more collector by domain name.
// This is function is one several integration points because it allows users
// to plug in new metric collectors by providing a factory to make them.
// Blip calls this function in an init function to register the built-in metric
// collectors.
//
// See types in the blip package for more details.
func Register(domain string, f blip.CollectorFactory) error {
	r.Lock()
	defer r.Unlock()
	_, ok := r.factory[domain]
	if ok {
		// @todo This should probably be ignored so users can override built-in
		//		 factories/collectors with their own, e.g. user-provided status.global
		//		 replaces built-in one.
		return fmt.Errorf("%s already registered", domain)
	}
	r.factory[domain] = f
	event.Sendf(event.REGISTER_METRICS, domain)
	return nil
}

// Make makes a metric collector for the domain using a previously registered factory.
//
// See types in the blip package for more details.
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

// Register built-in collectors using built-in factories.
func init() {
	for _, mc := range builtinCollectors {
		Register(mc, f)
	}
}

// repo holds registered blip.CollectorFactory. There's a single package
// instance below.
type repo struct {
	*sync.Mutex
	factory map[string]blip.CollectorFactory
}

// Internal package instance of repo that holds all collector factories registered
// by calls to Register, which includes the built-in factories.
var r = &repo{
	Mutex:   &sync.Mutex{},
	factory: map[string]blip.CollectorFactory{},
}

// factory is the built-in factory for creating all built-in collectors.
// There's a single package instance below. It implements blip.CollectorFactory.
type factory struct{}

var _ blip.CollectorFactory = &factory{}

// Internet package instance of factory that makes all built-it collectors.
// This factory is registered in the init func above.
var f = factory{}

// Make makes a metric collector for the domain. This is the built-in factory
// that makes the built-in collectors: status.global, var.global, and so on.
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

// List of built-in collectors. To add one, add its domain name here, and add
// the same domain in the switch statement above (in factory.Make).
var builtinCollectors = []string{
	"status.global",
	"var.global",
	"size.data",
	"size.binlogs",
	"innodb",
}
