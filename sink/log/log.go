package log

import (
	"log"

	"github.com/square/blip"
)

// Sink logs metrics.
type sink struct {
}

func NewSink(opts map[string]string) (sink, error) {
	return sink{}, nil
}

func (s sink) Send(m blip.Metrics) error {
	log.Printf("%+v", m.Values)
	return nil
}

func (s sink) Status() error {
	return nil
}
