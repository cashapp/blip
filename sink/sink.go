package sink

import (
	"fmt"
	"sync"
	"time"

	"github.com/square/blip"
	"github.com/square/blip/event"
)

func Register(name string, f blip.SinkFactory) error {
	r.Lock()
	defer r.Unlock()
	_, ok := r.factory[name]
	if ok {
		return fmt.Errorf("%s already registered", name)
	}
	r.factory[name] = f
	event.Sendf(event.REGISTER_SINK, name)
	return nil
}

func Make(name, monitorId string, opts map[string]string) (blip.Sink, error) {
	r.Lock()
	defer r.Unlock()
	f, ok := r.factory[name]
	if !ok {
		return nil, fmt.Errorf("%s not registered", name)
	}
	return f.Make(name, monitorId, opts)
}

// --------------------------------------------------------------------------

func init() {
	Register("log", f)
	Register("signalfx", f)
}

type repo struct {
	*sync.Mutex
	factory map[string]blip.SinkFactory
}

var r = &repo{
	Mutex:   &sync.Mutex{},
	factory: map[string]blip.SinkFactory{},
}

type factory struct{}

var f = factory{}

func (f factory) Make(name, monitorId string, opts map[string]string) (blip.Sink, error) {
	switch name {
	case "signalfx":
		sfx, err := NewSignalFxSink(monitorId, opts)
		if err != nil {
			return nil, err
		}
		var sendTimeout time.Duration
		if v, ok := opts["send-timeout"]; ok {
			d, err := time.ParseDuration(v)
			if err != nil {
				return nil, err
			}
			sendTimeout = d
		} else {
			sendTimeout = 1 * time.Second
		}
		rb := NewRetryBuffer(sfx, sendTimeout, 5)
		return rb, nil
	case "log":
		return NewLogSink(monitorId)
	}
	return nil, fmt.Errorf("%s not registered", name)
}
