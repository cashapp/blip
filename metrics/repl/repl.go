// Copyright 2022 Block, Inc.

package repl

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/cashapp/blip"
	"github.com/cashapp/blip/sqlutil"
)

const (
	DOMAIN = "repl"

	NOT_A_REPLICA = -1
)

type replMetrics struct {
	chedkRunning bool
}

type Repl struct {
	db      *sql.DB
	atLevel map[string]replMetrics
}

var _ blip.Collector = &Repl{}

func NewRepl(db *sql.DB) *Repl {
	return &Repl{
		db:      db,
		atLevel: map[string]replMetrics{},
	}
}

func (c *Repl) Domain() string {
	return DOMAIN
}

func (c *Repl) Help() blip.CollectorHelp {
	return blip.CollectorHelp{
		Domain:      DOMAIN,
		Description: "Replication status",
		Options:     map[string]blip.CollectorHelpOption{},
		Metrics: []blip.CollectorMetric{
			{
				Name: "running",
				Type: blip.GAUGE,
				Desc: "1=running (no error), 0=not running, -1=not a replica",
			},
		},
	}
}

var statusQuery = "SHOW SLAVE STATUS" // SHOW REPLICA STATUS as of 8.022

func (c *Repl) Prepare(ctx context.Context, plan blip.Plan) (func(), error) {
	haveVersion := false

LEVEL:
	for _, level := range plan.Levels {
		dom, ok := level.Collect[DOMAIN]
		if !ok {
			continue LEVEL // not collected in this level
		}

		if len(dom.Metrics) == 0 {
			return nil, fmt.Errorf("no metrics specified, expect at least one collector metric (run 'blip --print-domains' to list collector metrics)")
		}

		m := replMetrics{}
		for i := range dom.Metrics {
			switch dom.Metrics[i] {
			case "running":
				m.chedkRunning = true
			default:
				return nil, fmt.Errorf("invalid collector metric: %s (run 'blip --print-domains' to list collector metrics)", dom.Metrics[i])
			}
		}

		c.atLevel[level.Name] = m

		// SHOW REPLICA STATUS as of 8.022
		if haveVersion {
			continue
		}
		major, _, patch := sqlutil.MySQLVersion(ctx, c.db)
		if major == -1 {
			blip.Debug("failed to get/parse MySQL version, ignoring")
			continue
		}
		haveVersion = true
		if major == 8 && patch >= 22 {
			statusQuery = "SHOW REPLICA STATUS"
		}
		blip.Debug("mysql %d.x.%d %s", major, patch, statusQuery)
	}
	return nil, nil
}

func (c *Repl) Collect(ctx context.Context, levelName string) ([]blip.MetricValue, error) {
	rm, ok := c.atLevel[levelName]
	if !ok {
		return nil, nil
	}

	// Return SHOW SLAVE|REPLICA STATUS as map[string]string, which can be nil
	// if MySQL is not a replica
	replStatus, err := sqlutil.RowToMap(ctx, c.db, statusQuery)
	if err != nil {
		return nil, fmt.Errorf("%s failed: %s", statusQuery, err)
	}

	metrics := []blip.MetricValue{}

	// Report repl.running: 1=running, 0=not running, -1=not a replica
	//
	// NOTE: values are literal, not passed through sqlutil.Float64, so
	//       we look for "Yes" not 1, which works in this specific case.
	if rm.chedkRunning {
		var running float64 // 0 = not running by default
		if len(replStatus) == 0 {
			// no SHOW SLAVE|REPLICA STATUS output = not a replica
			running = float64(NOT_A_REPLICA)
		} else if replStatus["Slave_IO_Running"] == "Yes" && replStatus["Slave_SQL_Running"] == "Yes" && replStatus["Last_Errno"] == "0" {
			// running if a replica and those ^ 3 conditions are true
			running = 1
		}
		m := blip.MetricValue{
			Name:  "running",
			Type:  blip.GAUGE,
			Value: running,
		}
		metrics = append(metrics, m)
	}

	// @todo collect other repl status metrics

	return metrics, nil
}
