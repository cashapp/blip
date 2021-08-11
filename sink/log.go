package sink

import (
	"context"
	"log"

	"github.com/square/blip"
)

// Sink logs metrics.
type logSink struct {
}

func NewLogSink() (logSink, error) {
	return logSink{}, nil
}

func (s logSink) Send(ctx context.Context, m *blip.Metrics) error {
	log.Printf("%+v", m.Values)
	return nil
}

func (s logSink) Status() error {
	return nil
}

func (s logSink) Name() string {
	return "log"
}
