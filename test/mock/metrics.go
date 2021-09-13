package mock

import (
	"context"

	"github.com/square/blip"
)

type MetricsCollector struct {
}

func (c MetricsCollector) Domain() string {
	return "mock"
}

func (c MetricsCollector) Help() blip.CollectorHelp {
	return blip.CollectorHelp{}
}

func (c MetricsCollector) Prepare(blip.Plan) error {
	return nil
}

func (c MetricsCollector) Collect(ctx context.Context, levelName string) (blip.Metrics, error) {
	return blip.Metrics{}, nil
}
