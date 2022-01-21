// Copyright 2022 Block, Inc.

package mock

import (
	"context"

	"github.com/cashapp/blip"
)

type Sink struct {
	SendFunc func(ctx context.Context, m *blip.Metrics) error
}

var _ blip.Sink = Sink{}

func (s Sink) Send(ctx context.Context, m *blip.Metrics) error {
	if s.SendFunc != nil {
		return s.SendFunc(ctx, m)
	}
	return nil
}

func (s Sink) Name() string {
	return "mock.Sink"
}
