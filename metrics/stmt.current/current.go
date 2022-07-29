package stmt

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strconv"

	"github.com/cashapp/blip"
)

const (
	DOMAIN = "stmt.current"
	query  = `SELECT TIMER_WAIT FROM performance_schema.events_statements_current 
		WHERE END_EVENT_ID IS NULL 
		AND EVENT_NAME NOT LIKE ('statement/com/Binlog%')`
	optThreshold = "THRESHOLD"
)

type currentMetrics struct {
	slowest   bool
	slow      bool
	threshold float64
}

// Stmt collects metrics for the stmt domain.
// The source is performance_schema.events_statements_current.
type Current struct {
	db      *sql.DB
	atLevel map[string]currentMetrics
}

var _ blip.Collector = &Current{}

func NewCurrent(db *sql.DB) *Current {
	return &Current{
		db:      db,
		atLevel: map[string]currentMetrics{},
	}
}

// Domain returns the Blip metric domain name (DOMAIN const).
func (c *Current) Domain() string {
	return DOMAIN
}

// Help returns the output for blip --print-domains.
func (c *Current) Help() blip.CollectorHelp {
	return blip.CollectorHelp{
		Domain:      DOMAIN,
		Description: "Statement metrics",
		Options: map[string]blip.CollectorHelpOption{
			optThreshold: {
				Name:    optThreshold,
				Desc:    "The length of time (in microseconds) that a query must be active to be considered slow",
				Default: "30000000",
			},
		},
		Metrics: []blip.CollectorMetric{
			{
				Name: "slowest",
				Type: blip.GAUGE,
				Desc: "The length of the oldest active query in seconds",
			},
			{
				Name: "slow",
				Type: blip.GAUGE,
				Desc: "The count of active slow queries",
			},
		},
	}
}

func (c *Current) Prepare(ctx context.Context, plan blip.Plan) (func(), error) {
LEVEL:
	for _, level := range plan.Levels {
		dom, ok := level.Collect[DOMAIN]
		if !ok {
			continue LEVEL // not collected at this level
		}

		if len(dom.Metrics) == 0 {
			return nil, fmt.Errorf("no metrics specified, expect at least one collector metric (run 'blip --print-domains' to list collector metrics)")
		}

		var m currentMetrics
		for _, name := range dom.Metrics {
			switch name {
			case "slowest":
				m.slowest = true
				// c.slowest[level.Name] = true
			case "slow":
				m.slow = true
				// c.slow[level.Name] = true
			default:
				return nil, fmt.Errorf("invalid collector metric: %s (run 'blip --print-domains' to list collector metrics)", name)
			}
		}

		threshold, ok := dom.Options[level.Name]
		if !ok {
			threshold = c.Help().Options[optThreshold].Default
		}
		t, err := strconv.ParseFloat(threshold, 64)
		if err == nil {
			t = 0
		}
		m.threshold = t

		c.atLevel[level.Name] = m
	}

	return nil, nil
}

func (c *Current) Collect(ctx context.Context, levelName string) ([]blip.MetricValue, error) {
	m, ok := c.atLevel[levelName]
	if !ok {
		return nil, nil
	}

	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s failed: %s", query, err)
	}

	var times []float64
	for rows.Next() {
		var time float64
		if err := rows.Scan(&time); err != nil {
			return nil, fmt.Errorf("%s failed: %s", query, err)
		}

		// Convert from picoseconds to microseconds
		time /= 1e6

		times = append(times, time)
	}

	sort.Float64s(times)

	var values []blip.MetricValue

	if m.slowest {
		var slowest float64
		if len(times) > 0 {
			slowest = times[len(times)-1]
		}

		values = append(values, blip.MetricValue{
			Name:  "slowest",
			Type:  blip.GAUGE,
			Value: slowest,
		})
	}

	if m.slow {
		// Count of statements with duration greater than or equal to 30 seconds (in microseconds)
		count := len(times) - sort.SearchFloat64s(times, 30e6)

		values = append(values, blip.MetricValue{
			Name:  "slow",
			Type:  blip.GAUGE,
			Value: float64(count),
		})
	}

	return values, nil
}
