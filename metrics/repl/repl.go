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
)

type replMetrics struct {
	hasError bool
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
				Name: "error",
				Type: blip.GAUGE,
				Desc: "1 if instance is a replica and Last_Errno is not zero, else 0",
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
			case "error":
				m.hasError = true
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

	replStatus, err := sqlutil.RowToMap(ctx, c.db, statusQuery)
	if err != nil {
		return nil, fmt.Errorf("%s failed: %s", statusQuery, err)
	}

	metrics := []blip.MetricValue{}

	if rm.hasError {
		var hasError float64
		if len(replStatus) > 0 && replStatus["Last_Errno"] != "0" {
			hasError = 1
		}
		m := blip.MetricValue{
			Name:  "error",
			Type:  blip.GAUGE,
			Value: hasError,
		}
		metrics = append(metrics, m)
	}

	return metrics, nil
}
