// Copyright 2022 Block, Inc.

package queryglobal

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/cashapp/blip"
	"github.com/cashapp/blip/metrics/util"
)

const (
	DOMAIN = "query.global"

	OPT_RESPONSE_TIME_PERCENTILES = "response-time-percentiles"
	OPT_RESPONSE_TIME_OPTIONAL    = "response-time-optional"
	OPT_TRUNCATE_TABLE            = "truncate-table"
	DEFAULT_PERCENTILE_OPTION     = "999"

	BASE_QUERY     = "SELECT ROUND(bucket_quantile * 100, 1) AS p, ROUND(bucket_timer_high / 1000000, 3) AS us FROM performance_schema.events_statements_histogram_global"
	TRUNCATE_QUERY = "TRUNCATE TABLE performance_schema.events_statements_histogram_global"
)

type Global struct {
	db *sql.DB
	// --
	optional      map[string]bool      // keyed on level
	percentiles   map[string][]float64 // keyed on level
	truncateTable map[string]bool      // keyed on level
	available     bool
}

var _ blip.Collector = &Global{}

func NewGlobal(db *sql.DB) *Global {
	return &Global{
		db:            db,
		optional:      map[string]bool{},
		percentiles:   map[string][]float64{},
		truncateTable: map[string]bool{},
		available:     true,
	}
}

// Domain returns the Blip metric domain name (DOMAIN const).
func (c *Global) Domain() string {
	return DOMAIN
}

// Help returns the output for blip --print-domains.
func (c *Global) Help() blip.CollectorHelp {
	return blip.CollectorHelp{
		Domain:      DOMAIN,
		Description: "Collect metrics for all queries",
		Options: map[string]blip.CollectorHelpOption{
			OPT_RESPONSE_TIME_OPTIONAL: {
				Name:    OPT_RESPONSE_TIME_OPTIONAL,
				Desc:    "If collecting response time metrics is optional. This will fail if it is required but not available",
				Default: "yes",
				Values: map[string]string{
					"yes": "Optional",
					"no":  "Required",
				},
			},
			OPT_RESPONSE_TIME_PERCENTILES: {
				Name:    OPT_RESPONSE_TIME_PERCENTILES,
				Desc:    "Comma-separated list of response time percentiles formatted as 999, 0.999 or 99.9",
				Default: DEFAULT_PERCENTILE_OPTION,
				//Values:  map[string]string{},
			},
			OPT_TRUNCATE_TABLE: {
				Name:    OPT_TRUNCATE_TABLE,
				Desc:    "If the source table should be truncated to reset data after each retrieval",
				Default: "yes",
				Values: map[string]string{
					"yes": "Truncate source table after each retrieval",
					"no":  "Do not truncate source table after each retrieval",
				},
			},
		},
		Metrics: []blip.CollectorMetric{
			{
				Name: "response_time",
				Type: blip.GAUGE,
				Desc: "Query response time (microseconds)",
			},
		},
	}
}

// Prepare prepares the collector for the given plan.
func (c *Global) Prepare(ctx context.Context, plan blip.Plan) (func(), error) {
	_, err := c.db.Query(BASE_QUERY)
	if err != nil {
		blip.Debug("response time metric is not available: %v", err.Error())
		c.available = false
	}

LEVEL:
	for _, level := range plan.Levels {
		dom, ok := level.Collect[DOMAIN]
		if !ok {
			continue LEVEL // not collected at this level
		}

		err := c.prepareLevel(dom, level)
		if err != nil {
			return nil, err
		}
	}
	return nil, nil
}

// Collect collects metrics at the given level.
func (c *Global) Collect(ctx context.Context, levelName string) ([]blip.MetricValue, error) {
	if !c.available {
		if !c.optional[levelName] {
			panic(fmt.Sprintf("%s: required response time metrics couldn't be collected", levelName))
		} else {
			return nil, nil
		}
	}

	var metrics []blip.MetricValue
	for _, percentile := range c.percentiles[levelName] {
		where := fmt.Sprintf(" WHERE bucket_quantile >= %f ORDER BY bucket_quantile LIMIT 1", percentile)
		query := BASE_QUERY + where

		var p float64
		var us float64
		err := c.db.QueryRowContext(ctx, query).Scan(&p, &us)
		if err != nil {
			return nil, fmt.Errorf("%s: error running query for response time metric: %s", levelName, err)
		}

		m := blip.MetricValue{
			Type:  blip.GAUGE,
			Name:  "response_time",
			Value: us,
			Meta: map[string]string{
				util.FormatPercentile(percentile): fmt.Sprintf("%.1f", p),
			},
		}
		metrics = append(metrics, m)
	}

	if c.truncateTable[levelName] {
		_, err := c.db.Exec(TRUNCATE_QUERY)
		if err != nil {
			return nil, err
		}
	}

	return metrics, nil
}

// prepareLevel sanitizes options for given level based on user specified options
func (c *Global) prepareLevel(dom blip.Domain, level blip.Level) error {
	if optional, ok := dom.Options[OPT_RESPONSE_TIME_OPTIONAL]; ok && optional == "no" {
		c.optional[level.Name] = false
	} else {
		c.optional[level.Name] = true // default
	}

	if truncate, ok := dom.Options[OPT_TRUNCATE_TABLE]; ok && truncate == "no" {
		c.truncateTable[level.Name] = false
	} else {
		c.truncateTable[level.Name] = true // default
	}

	var percentilesStr string
	if percentilesOption, ok := dom.Options[OPT_RESPONSE_TIME_PERCENTILES]; ok {
		percentilesStr = percentilesOption
	} else {
		percentilesStr = DEFAULT_PERCENTILE_OPTION
	}

	percentilesList := strings.Split(strings.TrimSpace(percentilesStr), ",")
	var percentiles []float64
	for _, percentileStr := range percentilesList {
		percentile, err := util.ParsePercentileStr(percentileStr)
		if err != nil {
			return err
		}

		percentiles = append(percentiles, percentile)
	}
	c.percentiles[level.Name] = percentiles

	return nil
}
