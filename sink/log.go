package sink

import (
	"context"
	"log"

	"github.com/square/blip"
	"github.com/square/blip/event"
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
	log.Printf("%+v", m.Values)
	return nil
}

func (s logSink) Status() error {
	return nil
}

func (s logSink) Name() string {
	return "log"
}

func (s logSink) MonitorId() string {
	return s.monitorId
}
