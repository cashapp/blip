package metrics

import (
	"fmt"
	"sort"
	"sync"

	"github.com/cashapp/blip"
	"github.com/cashapp/blip/event"
	"github.com/cashapp/blip/metrics/innodb"
	"github.com/cashapp/blip/metrics/repl.lag"
	"github.com/cashapp/blip/metrics/size.binlog"
	"github.com/cashapp/blip/metrics/size.data"
	"github.com/cashapp/blip/metrics/status.global"
	"github.com/cashapp/blip/metrics/var.global"
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
	if ok && blip.Strict {
		return fmt.Errorf("%s already registered", domain)
	}
	r.factory[domain] = f
	event.Sendf(event.REGISTER_METRICS, domain)
	return nil
}

// List lists all registered metric collectors. It is used by the server API
// for GET /registered.
func List() []string {
	r.Lock()
	defer r.Unlock()
	names := []string{}
	for k := range r.factory {
		names = append(names, k)
	}
	return names
}

// Make makes a metric collector for the domain using a previously registered factory.
//
// See types in the blip package for more details.
func Make(domain string, args blip.CollectorFactoryArgs) (blip.Collector, error) {
	r.Lock()
	defer r.Unlock()
	f, ok := r.factory[domain]
	if !ok {
		return nil, blip.ErrInvalidDomain{Domain: domain}

	}
	return f.Make(domain, args)
}

func PrintDomains() string {
	r.Lock()
	domains := make([]string, 0, len(r.factory))
	for d := range r.factory {
		domains = append(domains, d)
	}
	sort.Strings(domains)
	r.Unlock()

	out := ""
	for _, domain := range domains {
		mc, _ := Make(domain, blip.CollectorFactoryArgs{})
		help := mc.Help()
		out += fmt.Sprintf("%s\n\t%s\n\n",
			help.Domain, help.Description,
		)

		opts := make([]string, 0, len(help.Options))
		for o := range help.Options {
			opts = append(opts, o)
		}

		if len(opts) > 0 {
			out += "\tOptions:\n"
			sort.Strings(opts)
			for _, optName := range opts {
				optHelp := help.Options[optName]
				out += "\t\t" + optName + ": " + optHelp.Desc
				if len(optHelp.Values) > 0 {
					out += "\n"
					valWidth := 0
					for val := range optHelp.Values {
						if len(val) > valWidth {
							valWidth = len(val)
						}
					}
					valLine := fmt.Sprintf("\t\t| %%-%ds = %%s", valWidth)

					for val, desc := range optHelp.Values {
						out += fmt.Sprintf(valLine, val, desc)
						if val == optHelp.Default {
							out += " (default)"
						}
						out += "\n"
					}
					out += "\n"
				} else if optHelp.Default != "" {
					out += " (default: " + optHelp.Default + ")\n\n"
				} else {
					out += "\n\n"
				}
			}
		} else {
			out += "\t(No options)\n\n"
		}

		if len(help.Groups) > 0 {
			out += "\tGroups:\n"
			for _, kv := range help.Groups {
				out += "\t\t" + kv.Key + " = " + kv.Value + "\n"
			}
			out += "\n"
		}

		if len(help.Meta) > 0 {
			out += "\tMeta:\n"
			for _, kv := range help.Meta {
				out += "\t\t" + kv.Key + " = " + kv.Value + "\n"
			}
			out += "\n"
		}

		if len(help.Metrics) > 0 {
			out += "\tMetrics:\n"
			for _, m := range help.Metrics {
				out += "\t\t" + m.Name
				switch m.Type {
				case blip.COUNTER:
					out += " (counter)"
				case blip.GAUGE:
					out += " (gauge)"
				default:
					out += " (unknown type)"
				}
				out += ": " + m.Desc + "\n"
			}
			out += "\n"
		}

		out += "\n"
	}

	return out
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
	case "innodb":
		return innodb.NewInnoDB(args.DB), nil
	case "repl.lag":
		return repllag.NewLag(args.DB), nil
	case "size.binlog":
		return sizebinlog.NewBinlog(args.DB), nil
	case "size.data":
		return sizedata.NewData(args.DB), nil
	case "status.global":
		return statusglobal.NewGlobal(args.DB), nil
	case "var.global":
		return varglobal.NewGlobal(args.DB), nil
	}
	return nil, blip.ErrInvalidDomain{Domain: domain}
}

// List of built-in collectors. To add one, add its domain name here, and add
// the same domain in the switch statement above (in factory.Make).
var builtinCollectors = []string{
	"innodb",
	"repl.lag",
	"size.binlog",
	"size.data",
	"status.global",
	"var.global",
}
