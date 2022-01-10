package percona

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/cashapp/blip"
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
	default_percentile_option = "95"
)

const (
	query      = "SELECT time, count, total FROM INFORMATION_SCHEMA.QUERY_RESPONSE_TIME WHERE TIME!='TOO LONG';"
	flushQuery = "SET GLOBAL query_response_time_flush=1"
)

type Qrt struct {
	db          *sql.DB
	available   bool
	percentiles map[string]map[string]float64
	optional    map[string]bool
	flushQrt    map[string]bool
}

func NewQrt(db *sql.DB) *Qrt {
	return &Qrt{
		db:          db,
		percentiles: map[string]map[string]float64{},
		optional:    map[string]bool{},
		available:   true,
	}
}

func (c *Qrt) Domain() string {
	return blip_domain
}

func (c *Qrt) Help() blip.CollectorHelp {
	return blip.CollectorHelp{
		Domain:      blip_domain,
		Description: "Collect QRT (Query Response Time) metrics",
		Options: map[string]blip.CollectorHelpOption{
			OPT_PERCENTILES: {
				Name:    OPT_PERCENTILES,
				Desc:    "Query Response Time Percentiles, in format (percentile1,percentile2)",
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
func (c *Qrt) Prepare(ctx context.Context, plan blip.Plan) error {
	if !c.available {
		return fmt.Errorf("%s: qrt metrics not available", plan.Name)
	}
	_, err := c.db.Query(query)
	if err != nil {
		c.available = false
		return fmt.Errorf("%s: qrt metrics not available", plan.Name)
	}

LEVEL:
	for _, level := range plan.Levels {
		dom, ok := level.Collect[blip_domain]
		if !ok {
			continue LEVEL
		}

		err := c.prepareLevel(dom, level)

		if err != nil {
			return err
		}
	}
	return nil
}

// Collect Collects query response time metrics for a particular level
func (c *Qrt) Collect(ctx context.Context, levelName string) ([]blip.MetricValue, error) {
	var metrics []blip.MetricValue
	if !c.available {
		if !c.optional[levelName] {
			errorStr := fmt.Sprintf("%s: required qrt metrics couldn't be collected", levelName)
			panic(errorStr)
		} else {
			return metrics, nil
		}
	}

	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {

		return nil, err
	}
	defer rows.Close()

	var h QRTHistogram
	var buckets []QRTBucket

	var time string
	var count uint64
	var total string
	for rows.Next() {
		if err := rows.Scan(&time, &count, &total); err != nil {
			continue
		}

		validatedTime, ok := sqlutil.Float64(strings.TrimSpace(time))
		if !ok {
			continue
		}

		validatedTotal, ok := sqlutil.Float64(strings.TrimSpace(total))
		if !ok {
			continue
		}

		buckets = append(buckets, NewQRTBucket(validatedTime, count, validatedTotal))
	}

	h = NewQRTHistogram(buckets)

	for name, val := range c.percentiles[levelName] {
		m := blip.MetricValue{Type: blip.GAUGE}
		m.Name = name
		m.Value = h.Percentile(val) * 100 // QRT is in sec and ODS want it in ms

		metrics = append(metrics, m)
	}

	if c.flushQrt[levelName] {
		_, err = c.db.Exec(flushQuery)
		if err != nil {
			return nil, err
		}
	}

	return metrics, err
}

// prepareLevel sanitizes options for a particular level based on user-provided options
func (c *Qrt) prepareLevel(dom blip.Domain, level blip.Level) error {
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

	c.percentiles[level.Name] = map[string]float64{}

	var percentilesStr string
	if percentilesOption, ok := dom.Options[OPT_PERCENTILES]; ok {
		percentilesStr = percentilesOption
	} else {
		percentilesStr = default_percentile_option
	}

	percentilesList := strings.Split(strings.TrimSpace(percentilesStr), ",")

	for _, percentileStr := range percentilesList {
		f, err := strconv.ParseFloat(percentileStr, 64)
		if err != nil {
			return fmt.Errorf("%s: could not parse percentile value in qrt collector %s into a number", level.Name, percentileStr)
		}

		percentile := f / 100.0 // percentilesStr are provided as whole percentage numbers (e.g. 50, 60)
		percentileAsDigitString := strings.Replace(percentileStr, ".", "", -1)
		percentileMetricName := fmt.Sprintf("query_response_pctl%s", percentileAsDigitString)
		c.percentiles[level.Name][percentileMetricName] = percentile
	}
	return nil
}