package null

import (
	"github.com/square/blip"
)

// Sink discards metrics.
type sink struct {
}

func NewSink(opts map[string]string) (sink, error) {
	return sink{}, nil
}

func (s sink) Send(m blip.Metrics) error {
	return nil
}

func (s sink) Status() error {
	return nil
}
