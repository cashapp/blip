package mock

import (
	"context"

	"github.com/square/blip/collect"
)

type MetricsCollector struct {
}

func (c MetricsCollector) Domain() string {
	return "mock"
}

func (c MetricsCollector) Help() collect.Help {
	return collect.Help{}
}

func (c MetricsCollector) Prepare(collect.Plan) error {
	return nil
}

func (c MetricsCollector) Collect(ctx context.Context, levelName string) (collect.Metrics, error) {
	return collect.Metrics{}, nil
}
