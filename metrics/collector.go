package metrics

import (
	"context"

	"github.com/square/blip/collect"
)

// Collector collects metrics for a single metric domain.
type Collector interface {
	// Domain returns the collector's metric domain name, like "var.global".
	Domain() string

	// Help returns information about using the collector.
	Help() collect.Help

	// Prepare prepares a plan for future calls to Collect.
	Prepare(collect.Plan) error

	// Collect collects metrics for the given in the previously prepared plan.
	Collect(ctx context.Context, levelName string) (collect.Metrics, error)
}
