package sink

import (
	"fmt"

	"github.com/square/blip/sink/log"
	"github.com/square/blip/sink/null"
	"github.com/square/blip/sink/signalfx"
)

type Factory interface {
	Make(sinkName string, opts map[string]string) (Sink, error)
}

type factory struct {
}

func NewFactory() factory {
	return factory{}
}

func (f factory) Make(sinkName string, opts map[string]string) (Sink, error) {
	switch sinkName {
	case "log":
		return log.NewSink(opts)
	case "null":
		return null.NewSink(opts)
	case "signalfx":
		return signalfx.NewSink(opts)
	}
	return nil, fmt.Errorf("invalid sink: %s", sinkName)
}
