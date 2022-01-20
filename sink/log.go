package sink

import (
	"context"
	"fmt"

	"github.com/cashapp/blip"
)

// Sink logs metrics.
type logSink struct {
	monitorId string
}

func NewLogSink(monitorId string) (logSink, error) {
	return logSink{monitorId: monitorId}, nil
}

func (s logSink) Send(ctx context.Context, m *blip.Metrics) error {
	fmt.Printf("in %s: %+v\n", m.End.Sub(m.Begin), m.Values)
	return nil
}

func (s logSink) Status() string {
	return "swimmingly"
}

func (s logSink) Name() string {
	return "log"
}
