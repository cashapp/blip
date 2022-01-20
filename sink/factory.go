package sink

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/cashapp/blip"
	"github.com/cashapp/blip/event"
)

func Register(name string, f blip.SinkFactory) error {
	r.Lock()
	defer r.Unlock()
	_, ok := r.factory[name]
	if ok {
		if blip.Strict {
			return fmt.Errorf("sink %s already registered", name)
		}
		blip.Debug("re-register sink %s", name)
	}
	r.factory[name] = f
	event.Sendf(event.REGISTER_SINK, name)
	return nil
}

func List() []string {
	r.Lock()
	defer r.Unlock()
	names := []string{}
	for k := range r.factory {
		names = append(names, k)
	}
	return names
}

func Make(args blip.SinkFactoryArgs) (blip.Sink, error) {
	r.Lock()
	defer r.Unlock()
	f, ok := r.factory[args.SinkName]
	if !ok {
		return nil, fmt.Errorf("sink %s not registered", args.SinkName)
	}
	return f.Make(args)
}

// --------------------------------------------------------------------------

func init() {
	Register("chronosphere", f)
	Register("signalfx", f)
	Register("log", f)
}

type repo struct {
	*sync.Mutex
	factory map[string]blip.SinkFactory
}

var r = &repo{
	Mutex:   &sync.Mutex{},
	factory: map[string]blip.SinkFactory{},
}

type factory struct {
	HTTPClient blip.HTTPClientFactory
}

var f = &factory{}

func InitFactory(factories blip.Factories) {
	f.HTTPClient = factories.HTTPClient
}

func (f *factory) Make(args blip.SinkFactoryArgs) (blip.Sink, error) {
	// Built-in log sink is special (simple), so return early if that
	if args.SinkName == "log" {
		return NewLogSink(args.MonitorId)
	}

	// ----------------------------------------------------------------------
	// Built-in sinks use Retry to serialize access and retry on Send
	// error. First build the specific sink, then return it wrapped in Retry.

	// Parse config.metrics.sink.*.send-timeout, which  is a Retry option
	retryArgs := RetryArgs{
		MonitorId: args.MonitorId,
	}
	if v, ok := args.Options["buffer-size"]; ok {
		n, err := strconv.Atoi(v)
		if err != nil {
			return nil, err
		}
		if n < 0 {
			return nil, fmt.Errorf("invalid retry buffer-size: %d: must be greater than zero", n)
		}
		retryArgs.BufferSize = uint(n)
	}
	if v, ok := args.Options["send-timeout"]; ok {
		d, err := time.ParseDuration(v)
		if err != nil {
			return nil, err
		}
		retryArgs.SendTimeout = d
	}
	if v, ok := args.Options["send-retry-wait"]; ok {
		d, err := time.ParseDuration(v)
		if err != nil {
			return nil, err
		}
		retryArgs.SendRetryWait = d
	}

	// Make specific built-in sink
	var err error
	switch args.SinkName {
	case "chronosphere":
		retryArgs.Sink, err = NewChronosphere(args.MonitorId, args.Options, args.Tags)
	case "signalfx":
		httpClient, err := f.HTTPClient.MakeForSink("signalfx", args.MonitorId, args.Options, args.Tags)
		if err != nil {
			return nil, err
		}
		retryArgs.Sink, err = NewSignalFx(args.MonitorId, args.Options, args.Tags, httpClient)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("sink %s not registered", args.SinkName)
	}
	if err != nil {
		return nil, err
	}

	// Return built-in sink as Retry, which implements blip.Sink
	return NewRetry(retryArgs), nil
}