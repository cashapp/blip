// Copyright 2022 Block, Inc.

package queryresponsetime

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	myerr "github.com/go-mysql/errors"

	"github.com/cashapp/blip"
	"github.com/cashapp/blip/errors"
	"github.com/cashapp/blip/metrics/util"
)

const (
	DOMAIN = "query.response-time"

	OPT_PERCENTILES           = "percentiles"
	OPT_REAL_PERCENTILES      = "real-percentiles"
	OPT_TRUNCATE_TABLE        = "truncate-table"
	DEFAULT_PERCENTILE_OPTION = "999"

	ERR_NO_TABLE = "table-not-exist"

	BASE_QUERY     = "SELECT ROUND(bucket_quantile * 100, 1) AS p, ROUND(bucket_timer_high / 1000000, 3) AS us FROM performance_schema.events_statements_histogram_global"
	TRUNCATE_QUERY = "TRUNCATE TABLE performance_schema.events_statements_histogram_global"
)

type ResponseTime struct {
	db *sql.DB
	// --
	query         map[string]map[float64]string        // keyed on level: level -> percentile -> query
	setMeta       map[string]bool                      // keyed on level
	truncateTable map[string]bool                      // keyed on level
	errPolicy     map[string]map[string]*errors.Policy // keyed on level
	stop          map[string]bool                      // keyed on level
}

var _ blip.Collector = &ResponseTime{}

func NewResponseTime(db *sql.DB) *ResponseTime {
	return &ResponseTime{
		db:            db,
		query:         map[string]map[float64]string{},
		setMeta:       map[string]bool{},
		truncateTable: map[string]bool{},
		errPolicy:     map[string]map[string]*errors.Policy{},
		stop:          map[string]bool{},
	}
}

// Domain returns the Blip metric domain name (DOMAIN const).
func (c *ResponseTime) Domain() string {
	return DOMAIN
}

// Help returns the output for blip --print-domains.
func (c *ResponseTime) Help() blip.CollectorHelp {
	return blip.CollectorHelp{
		Domain:      DOMAIN,
		Description: "Collect metrics for query response time",
		Options: map[string]blip.CollectorHelpOption{
			OPT_PERCENTILES: {
				Name:    OPT_PERCENTILES,
				Desc:    "Comma-separated list of response time percentiles formatted as 999, 0.999 or 99.9",
				Default: DEFAULT_PERCENTILE_OPTION,
			},
			OPT_REAL_PERCENTILES: {
				Name:    OPT_REAL_PERCENTILES,
				Desc:    "If real percentiles are included in meta",
				Default: "yes",
				Values: map[string]string{
					"yes": "Include real percentiles in meta",
					"no":  "Exclude real percentiles in meta",
				},
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
		Errors: map[string]blip.CollectorHelpError{
			ERR_NO_TABLE: {
				Name:    ERR_NO_TABLE,
				Handles: "MySQL error 1146: Table 'performance_schema.events_statements_histogram_global' doesn't exist",
				Default: errors.NewPolicy("").String(),
			},
		},
	}
}

// Prepare prepares the collector for the given plan.
func (c *ResponseTime) Prepare(ctx context.Context, plan blip.Plan) (func(), error) {
LEVEL:
	for _, level := range plan.Levels {
		dom, ok := level.Collect[DOMAIN]
		if !ok {
			continue LEVEL // not collected at this level
		}

		if rp, ok := dom.Options[OPT_REAL_PERCENTILES]; ok && rp == "no" {
			c.setMeta[level.Name] = false
		} else {
			c.setMeta[level.Name] = true // default
		}

		if truncate, ok := dom.Options[OPT_TRUNCATE_TABLE]; ok && truncate == "no" {
			c.truncateTable[level.Name] = false
		} else {
			c.truncateTable[level.Name] = true // default
		}

		var percentilesStr string
		if percentilesOption, ok := dom.Options[OPT_PERCENTILES]; ok {
			percentilesStr = percentilesOption
		} else {
			percentilesStr = DEFAULT_PERCENTILE_OPTION
		}

		c.query[level.Name] = map[float64]string{}
		percentilesList := strings.Split(strings.TrimSpace(percentilesStr), ",")
		for _, percentileStr := range percentilesList {
			percentile, err := util.ParsePercentileStr(percentileStr)
			if err != nil {
				return nil, err
			}

			where := fmt.Sprintf(" WHERE bucket_quantile >= %f ORDER BY bucket_quantile LIMIT 1", percentile)
			query := BASE_QUERY + where
			c.query[level.Name][percentile] = query
		}

		// Apply custom error policies, if any
		c.errPolicy[level.Name] = map[string]*errors.Policy{}
		c.errPolicy[level.Name][ERR_NO_TABLE] = errors.NewPolicy(dom.Errors[ERR_NO_TABLE])
		blip.Debug("error policy: %s=%s", ERR_NO_TABLE, c.errPolicy[level.Name][ERR_NO_TABLE])
	}

	return nil, nil
}

// Collect collects metrics at the given level.
func (c *ResponseTime) Collect(ctx context.Context, levelName string) ([]blip.MetricValue, error) {
	if c.stop[levelName] {
		blip.Debug("stopped by previous error")
		return nil, nil
	}

	var metrics []blip.MetricValue
	for percentile, query := range c.query[levelName] {
		metricName := util.FormatPercentile(percentile)
		var p float64
		var us float64
		err := c.db.QueryRowContext(ctx, query).Scan(&p, &us)
		if err != nil {
			return c.collectError(err, levelName, metricName)
		}

		m := blip.MetricValue{
			Type:  blip.GAUGE,
			Name:  metricName,
			Value: us,
		}
		if c.setMeta[levelName] {
			m.Meta = map[string]string{
				metricName: fmt.Sprintf("%.1f", p),
			}
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

func (c *ResponseTime) collectError(err error, levelName string, metricName string) ([]blip.MetricValue, error) {
	var ep *errors.Policy
	switch myerr.MySQLErrorCode(err) {
	case 1146:
		ep = c.errPolicy[levelName][ERR_NO_TABLE]
	default:
		return nil, err
	}

	// Stop trying to collect if error policy retry="stop". This affects
	// future calls to Collect; don't return yet because we need to check
	// the metric policy: drop or zero. If zero, we must report one zero val.
	if ep.Retry == errors.POLICY_RETRY_NO {
		c.stop[levelName] = true
	}

	// Report
	var reportedErr error
	if ep.ReportError() {
		reportedErr = err
	} else {
		blip.Debug("error policy=ignore: %v", err)
	}

	var metrics []blip.MetricValue
	if ep.Metric == errors.POLICY_METRIC_ZERO {
		metrics = []blip.MetricValue{{
			Type:  blip.GAUGE,
			Name:  metricName,
			Value: 0,
		}}
	}

	return metrics, reportedErr
}
