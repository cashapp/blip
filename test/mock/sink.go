package mock

import (
	"context"

	"github.com/square/blip"
)

type Sink struct {
	SendFunc      func(ctx context.Context, m *blip.Metrics) error
	MonitorIdFunc func() string
}

var _ blip.Sink = Sink{}

func (s Sink) Send(ctx context.Context, m *blip.Metrics) error {
	if s.SendFunc != nil {
		return s.SendFunc(ctx, m)
	}
	return nil
}

func (s Sink) Status() error {
	return nil
}

func (s Sink) Name() string {
	return "mock.Sink"
}

func (s Sink) MonitorId() string {
	if s.MonitorIdFunc != nil {
		return s.MonitorIdFunc()
	}
	return ""
}
