// Copyright 2022 Block, Inc.

package percona

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/cashapp/blip"
	"github.com/cashapp/blip/metrics/util"
	"github.com/cashapp/blip/sqlutil"
)

/*
Percona root@localhost:(none)> SELECT time, count, total FROM INFORMATION_SCHEMA.QUERY_RESPONSE_TIME WHERE TIME != 'TOO LONG';\G
+----------------+-------+----------------+
| time           | count | total          |
+----------------+-------+----------------+
|       0.000001 | 0     |       0.000000 |
|       0.000010 | 0     |       0.000000 |
|       0.000100 | 0     |       0.000000 |
|       0.001000 | 0     |       0.000000 |
|       0.010000 | 0     |       0.000000 |
|       0.100000 | 0     |       0.000000 |
|       1.000000 | 0     |       0.000000 |
|      10.000000 | 0     |       0.000000 |
|     100.000000 | 0     |       0.000000 |
|    1000.000000 | 0     |       0.000000 |
|   10000.000000 | 0     |       0.000000 |
|  100000.000000 | 0     |       0.000000 |
| 1000000.000000 | 0     |       0.000000 |
+----------------+-------+----------------+
*/

const (
	blip_domain = "percona.response-time"
)

const (
	OPT_PERCENTILES           = "percentiles"
	OPT_OPTIONAL              = "optional"
	OPT_FLUSH_QRT             = "flush"
	default_percentile_option = "999"
)

const (
	query      = "SELECT time, count, total FROM INFORMATION_SCHEMA.QUERY_RESPONSE_TIME WHERE TIME!='TOO LONG';"
	flushQuery = "SET GLOBAL query_response_time_flush=1"
)

type QRT struct {
	db          *sql.DB
	available   bool
	percentiles map[string]map[float64]float64
	optional    map[string]bool
	flushQrt    map[string]bool
}

func NewQRT(db *sql.DB) *QRT {
	return &QRT{
		db:          db,
		percentiles: map[string]map[float64]float64{},
		optional:    map[string]bool{},
		flushQrt:    map[string]bool{},
		available:   true,
	}
}

func (c *QRT) Domain() string {
	return blip_domain
}

func (c *QRT) Help() blip.CollectorHelp {
	return blip.CollectorHelp{
		Domain:      blip_domain,
		Description: "Collect QRT (Query Response Time) metrics",
		Options: map[string]blip.CollectorHelpOption{
			OPT_PERCENTILES: {
				Name:    OPT_PERCENTILES,
				Desc:    "Comma-separated list of percentiles formatted as 999, 0.999 or 99.9",
				Default: default_percentile_option,
				Values:  map[string]string{},
			},
			OPT_OPTIONAL: {
				Name:    OPT_OPTIONAL,
				Desc:    "If collecting QRT metrics is optional. This will fail if QRT is required but not available.",
				Default: "yes",
				Values: map[string]string{
					"yes": "Optional",
					"no":  "Required",
				},
			},
			OPT_FLUSH_QRT: {
				Name:    OPT_FLUSH_QRT,
				Desc:    "If Query Response Time should be flushed after each retrieval.",
				Default: "yes",
				Values: map[string]string{
					"yes": "Flush Query Response Time (QRT) after each retrieval.",
					"no":  "Do not flush Query Response Time (QRT) after each retrieval.",
				},
			},
		},
	}
}

// Prepare Prepares options for all levels in the plan that contain the percona.response-time domain
func (c *QRT) Prepare(ctx context.Context, plan blip.Plan) (func(), error) {
	_, err := c.db.Query(query)
	if err != nil {
		blip.Debug("error running qrt query: %v", err.Error())
		c.available = false
	}

LEVEL:
	for _, level := range plan.Levels {
		dom, ok := level.Collect[blip_domain]
		if !ok {
			continue LEVEL
		}

		err := c.prepareLevel(dom, level)
		if err != nil {
			return nil, err
		}
	}
	return nil, nil
}

// Collect Collects query response time metrics for a particular level
func (c *QRT) Collect(ctx context.Context, levelName string) ([]blip.MetricValue, error) {
	if !c.available {
		if !c.optional[levelName] {
			errorStr := fmt.Sprintf("%s: required qrt metrics couldn't be collected", levelName)
			panic(errorStr)
		} else {
			return nil, nil
		}
	}

	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {

		return nil, err
	}
	defer rows.Close()

	var buckets []QRTBucket

	var time string
	var count uint64
	var total string
	for rows.Next() {
		if err := rows.Scan(&time, &count, &total); err != nil {
			return nil, err
		}

		validatedTime, ok := sqlutil.Float64(strings.TrimSpace(time))
		if !ok {
			return nil, fmt.Errorf("%s: qrt: time could't be parsed into a valid float: %s ", levelName, time)
		}

		validatedTotal, ok := sqlutil.Float64(strings.TrimSpace(total))
		if !ok {
			return nil, fmt.Errorf("%s: qrt: total couldn't be parsed into a valid float: %s ", levelName, total)
		}

		buckets = append(buckets, QRTBucket{Time: validatedTime, Count: count, Total: validatedTotal})
	}

	h := NewQRTHistogram(buckets)

	var metrics []blip.MetricValue
	for percentile := range c.percentiles[levelName] {
		// Get value of percentile (e.g. p999) and actual percentile (e.g. p997).
		// The latter is reported as meta so user can discard percentile if the
		// actual percentile is too far off, which can happen if bucket range is
		// configured too small.
		value, actualPercentile := h.Percentile(percentile)
		m := blip.MetricValue{
			Type:  blip.GAUGE,
			Name:  "response_time",
			Value: value * 1000000, // convert seconds to microseconds for consistency with PFS quantiles
			Meta: map[string]string{
				util.FormatPercentile(percentile): fmt.Sprintf("%.3f", actualPercentile),
			},
		}
		metrics = append(metrics, m)
	}

	if c.flushQrt[levelName] {
		_, err = c.db.Exec(flushQuery)
		if err != nil {
			return nil, err
		}
	}

	return metrics, nil
}

// prepareLevel sanitizes options for a particular level based on user-provided options
func (c *QRT) prepareLevel(dom blip.Domain, level blip.Level) error {
	if optional, ok := dom.Options[OPT_OPTIONAL]; ok && optional == "no" {
		c.optional[level.Name] = false
	} else {
		c.optional[level.Name] = true // default
	}

	if flushQrt, ok := dom.Options[OPT_FLUSH_QRT]; ok && flushQrt == "no" {
		c.flushQrt[level.Name] = false
	} else {
		c.flushQrt[level.Name] = true // default
	}

	c.percentiles[level.Name] = map[float64]float64{}

	var percentilesStr string
	if percentilesOption, ok := dom.Options[OPT_PERCENTILES]; ok {
		percentilesStr = percentilesOption
	} else {
		percentilesStr = default_percentile_option
	}

	percentilesList := strings.Split(strings.TrimSpace(percentilesStr), ",")

	for _, percentileStr := range percentilesList {
		percentile, err := util.ParsePercentileStr(percentileStr)
		if err != nil {
			return err
		}

		c.percentiles[level.Name][percentile] = percentile
	}
	return nil
}
