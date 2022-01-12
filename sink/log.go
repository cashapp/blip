package sink

import (
	"context"
	"log"

	"github.com/cashapp/blip"
	"github.com/cashapp/blip/event"
)

// Sink logs metrics.
type logSink struct {
	event     event.MonitorSink
	monitorId string
}

func NewLogSink(monitorId string) (logSink, error) {
	return logSink{
		event:     event.MonitorSink{MonitorId: monitorId},
		monitorId: monitorId,
	}, nil
}

func (s logSink) Send(ctx context.Context, m *blip.Metrics) error {
	log.Printf("in %s: %+v", m.End.Sub(m.Begin), m.Values)
	return nil
}

func (s logSink) Status() string {
	return ""
}
