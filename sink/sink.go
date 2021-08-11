package sink

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/square/blip"
	"github.com/square/blip/event"
)

// Sink sends metrics to an external destination.
type Sink interface {
	Send(context.Context, *blip.Metrics) error
	Status() error
	Name() string
}

type Factory interface {
	Make(name string, opts map[string]string) (Sink, error)
}

type factory struct{}

func (f factory) Make(name string, opts map[string]string) (Sink, error) {
	switch name {
	case "signalfx":
		sfx, err := NewSignalFxSink(opts)
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
		return NewLogSink()
	}
	return nil, fmt.Errorf("%s not registered", name)
}

var defaultFactory = factory{}

func RegisterDefaults() {
	Register("log", defaultFactory)
	Register("signalfx", defaultFactory)
}

// --------------------------------------------------------------------------

type repo struct {
	*sync.Mutex
	factory map[string]Factory
}

var sinkRepo = &repo{
	Mutex:   &sync.Mutex{},
	factory: map[string]Factory{},
}

func Register(name string, f Factory) error {
	sinkRepo.Lock()
	defer sinkRepo.Unlock()
	_, ok := sinkRepo.factory[name]
	if ok {
		return fmt.Errorf("%s already registered", name)
	}
	sinkRepo.factory[name] = f
	event.Sendf(event.REGISTER_SINK, name)
	return nil
}

func Make(name string, opts map[string]string) (Sink, error) {
	sinkRepo.Lock()
	defer sinkRepo.Unlock()
	f, ok := sinkRepo.factory[name]
	if !ok {
		return nil, fmt.Errorf("%s not registered", name)
	}
	return f.Make(name, opts)
}
