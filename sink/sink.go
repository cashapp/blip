package sink

import (
	"github.com/square/blip"
)

// Sink sends metrics to an external destination.
type Sink interface {
	Send(blip.Metrics) error
	Status() error
}

type TransformMetrics struct {
	Plugin func(*blip.Metrics) error
	Sinks  []Sink
}

func (s TransformMetrics) Send(m blip.Metrics) error {
	s.Plugin(&m)
	for i := range s.Sinks {
		if err := s.Sinks[i].Send(m); err != nil {
			return err
		}
	}
	return nil
}

func (s TransformMetrics) Status() error {
	return nil
}
