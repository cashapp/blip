package stmt

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/cashapp/blip"
)

const DOMAIN = "stmt"

// Stmt collects metrics for the event.stmt domain.
// The source is performance_schema.events_statements_current.
type Stmt struct {
	db      *sql.DB
	atLevel map[string][]stmtMetric
}

// Verify collector implements blip.Collector interface
var _ blip.Collector = &Stmt{}

// NewStmt makes a new Stmt collector.
func NewStmt(db *sql.DB) *Stmt {
	return &Stmt{
		db:      db,
		atLevel: map[string][]stmtMetric{},
	}
}

// Domain returns the Blip metric domain name (DOMAIN const).
func (c *Stmt) Domain() string {
	return DOMAIN
}

// Help returns the output for blip --print-domains.
func (c *Stmt) Help() blip.CollectorHelp {
	return blip.CollectorHelp{
		Domain:      DOMAIN,
		Description: "Statement metrics",
		Options:     map[string]blip.CollectorHelpOption{},
		Metrics: []blip.CollectorMetric{
			{
				Name: "oldestQuery",
				Type: blip.GAUGE,
				Desc: "The time of oldest query in seconds",
			},
			{
				Name: "activeLongRunningQueries",
				Type: blip.GAUGE,
				Desc: "The count of long running query",
			},
		},
	}
}

// Prepare prepares the collector for the given plan.
func (c *Stmt) Prepare(ctx context.Context, plan blip.Plan) (func(), error) {
LEVEL:
	for _, level := range plan.Levels {
		dom, ok := level.Collect[DOMAIN]
		if !ok {
			continue LEVEL // not collected at this level
		}

		if len(dom.Metrics) == 0 {
			return nil, fmt.Errorf("no metrics specified, expect at least one collector metric (run 'blip --print-domains' to list collector metrics)")
		}

		metrics := []stmtMetric{}
		for _, name := range dom.Metrics {
			// var metric stmtMetric

			switch name {
			case "oldestQuery":
				metrics = append(metrics, oldestQuery{metric{name}})
			case "activeLongRunningQueries":
				metrics = append(metrics, activeLongQueryCount{metric{name}})
			default:
				return nil, fmt.Errorf("invalid collector metric: %s (run 'blip --print-domains' to list collector metrics)", name)
			}
		}

		c.atLevel[level.Name] = metrics
	}

	return nil, nil
}

// Collect collects metrics at the given level.
func (c *Stmt) Collect(ctx context.Context, levelName string) ([]blip.MetricValue, error) {
	rm, ok := c.atLevel[levelName]
	if !ok {
		return nil, nil
	}

	metrics := []blip.MetricValue{}
	for _, m := range rm {
		metrics = append(metrics, blip.MetricValue{
			Name:  m.Name(),
			Type:  blip.GAUGE,
			Value: m.CollectMetric(c.db),
		})
	}

	return metrics, nil
}