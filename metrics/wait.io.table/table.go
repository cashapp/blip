// Copyright 2022 Block, Inc.

package waitiotable

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/cashapp/blip"
	"github.com/cashapp/blip/errors"
)

const (
	DOMAIN = "wait.io.table"

	OPT_EXCLUDE          = "exclude"
	OPT_INCLUDE          = "include"
	OPT_TRUNCATE_TABLE   = "truncate-table"
	OPT_TRUNCATE_TIMEOUT = "truncate-timeout"
	OPT_ALL              = "all"

	OPT_EXCLUDE_DEFAULT = "mysql.*,information_schema.*,performance_schema.*,sys.*"

	TRUNCATE_QUERY = "TRUNCATE TABLE performance_schema.table_io_waits_summary_by_table"

	ERR_TRUNCATE_FAILED = "truncate-failed"
)

var (
	columnNames = []string{
		"sum_timer_wait",
		"min_timer_wait",
		"avg_timer_wait",
		"max_timer_wait",
		"count_read",
		"sum_timer_read",
		"min_timer_read",
		"avg_timer_read",
		"max_timer_read",
		"count_write",
		"sum_timer_write",
		"min_timer_write",
		"avg_timer_write",
		"max_timer_write",
		"count_fetch",
		"sum_timer_fetch",
		"min_timer_fetch",
		"avg_timer_fetch",
		"max_timer_fetch",
		"count_insert",
		"sum_timer_insert",
		"min_timer_insert",
		"avg_timer_insert",
		"max_timer_insert",
		"count_update",
		"sum_timer_update",
		"min_timer_update",
		"avg_timer_update",
		"max_timer_update",
		"count_delete",
		"sum_timer_delete",
		"min_timer_delete",
		"avg_timer_delete",
		"max_timer_delete",
	}

	columnExists map[string]struct{}
)

func init() {
	columnExists = make(map[string]struct{}, len(columnNames))
	for _, name := range columnNames {
		columnExists[name] = struct{}{}
	}
}

type tableOptions struct {
	query            string
	truncate         bool
	truncateTimeout  time.Duration
	hadTruncateError bool
	stop             bool

	errPolicy map[string]*errors.Policy
}

// Table collects table io for domain wait.io.table.
type Table struct {
	db *sql.DB
	// --
	options map[string]*tableOptions
}

// Verify collector implements blip.Collector interface.
var _ blip.Collector = &Table{}

// NewTable makes a new Table collector,
func NewTable(db *sql.DB) *Table {
	return &Table{
		db:      db,
		options: map[string]*tableOptions{},
	}
}

// Domain returns the Blip metric domain name (DOMAIN const).
func (t *Table) Domain() string {
	return DOMAIN
}

// Help returns the output for blip --print-domains.
func (t *Table) Help() blip.CollectorHelp {
	return blip.CollectorHelp{
		Domain:      DOMAIN,
		Description: "Table IO Waits",
		Options: map[string]blip.CollectorHelpOption{
			OPT_INCLUDE: {
				Name: OPT_INCLUDE,
				Desc: "Comma-separated list of database or table names to include (overrides option " + OPT_EXCLUDE + ")",
			},
			OPT_EXCLUDE: {
				Name:    OPT_EXCLUDE,
				Desc:    "Comma-separated list of database or table names to exclude (ignored if " + OPT_INCLUDE + " is set)",
				Default: OPT_EXCLUDE_DEFAULT,
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
			OPT_TRUNCATE_TIMEOUT: {
				Name:    OPT_TRUNCATE_TIMEOUT,
				Desc:    "The amount of time to attempt to truncate the source table before timing out",
				Default: "2s",
			},
			OPT_ALL: {
				Name:    OPT_ALL,
				Desc:    "Collect all metrics",
				Default: "no",
				Values: map[string]string{
					"yes": "All metrics (ignore metrics list)",
					"no":  "Specified metrics",
				},
			},
		},
		Groups: []blip.CollectorKeyValue{
			{Key: "db", Value: "the database name for the corresponding table io, or empty string for all dbs"},
			{Key: "tbl", Value: "the table name for the corresponding table io, or empty string for all tables"},
		},
		Errors: map[string]blip.CollectorHelpError{
			ERR_TRUNCATE_FAILED: {
				Name:    ERR_TRUNCATE_FAILED,
				Handles: "Truncation failures on 'performance_schema.table_io_waits_summary_by_table'",
				Default: errors.NewPolicy("").String(),
			},
		},
	}
}

// Prepare prepares the collector for the given plan.
func (t *Table) Prepare(ctx context.Context, plan blip.Plan) (func(), error) {
LEVEL:
	for _, level := range plan.Levels {
		o := tableOptions{}

		dom, ok := level.Collect[DOMAIN]
		if !ok {
			continue LEVEL // not collected in this level
		}
		if dom.Options == nil {
			dom.Options = make(map[string]string)
		}
		if _, ok := dom.Options[OPT_EXCLUDE]; !ok {
			dom.Options[OPT_EXCLUDE] = OPT_EXCLUDE_DEFAULT
		}

		o.query = TableIoWaitQuery(dom.Options, dom.Metrics)

		if truncate, ok := dom.Options[OPT_TRUNCATE_TABLE]; ok && truncate == "no" {
			o.truncate = false
		} else {
			o.truncate = true // default
		}

		if truncateTimeout, ok := dom.Options[OPT_TRUNCATE_TIMEOUT]; ok && o.truncate {
			if duration, err := time.ParseDuration(truncateTimeout); err != nil {
				return nil, fmt.Errorf("Invalid truncate duration: %v", err)
			} else {
				o.truncateTimeout = duration
			}
		} else {
			o.truncateTimeout = 2 * time.Second // default
		}

		o.errPolicy = map[string]*errors.Policy{}

		if o.truncate {
			o.errPolicy[ERR_TRUNCATE_FAILED] = errors.NewPolicy(dom.Errors[ERR_TRUNCATE_FAILED])
			blip.Debug("error policy: %s=%s", ERR_TRUNCATE_FAILED, o.errPolicy[ERR_TRUNCATE_FAILED])
		}

		t.options[level.Name] = &o
	}
	return nil, nil
}

func (t *Table) Collect(ctx context.Context, levelName string) ([]blip.MetricValue, error) {
	o, ok := t.options[levelName]
	if !ok {
		return nil, nil
	}

	if o.stop {
		blip.Debug("stopped by previous error")
		return nil, nil
	}

	rows, err := t.db.QueryContext(ctx, o.query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var (
		metrics []blip.MetricValue
		dbName  string
		tblName string
		values  []interface{}
	)

	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("Failed to get columns for wait.io.table: %v", err)
	}

	values = make([]interface{}, len(cols))
	values[0] = new(string)
	values[1] = new(string)

	for i := 2; i < len(cols); i++ {
		values[i] = new(int64)
	}

	for rows.Next() {
		if err = rows.Scan(values...); err != nil {
			return nil, err
		}

		dbName = *values[0].(*string)
		tblName = *values[1].(*string)

		for i := 2; i < len(cols); i++ {
			m := blip.MetricValue{
				Name:  cols[i],
				Type:  blip.COUNTER,
				Group: map[string]string{"db": dbName, "tbl": tblName},
			}
			m.Value = float64(*values[i].(*int64))
			metrics = append(metrics, m)
		}

	}

	if o.truncate {
		trCtx, cancelFn := context.WithTimeout(ctx, o.truncateTimeout)
		defer cancelFn()
		_, err := t.db.ExecContext(trCtx, TRUNCATE_QUERY)
		if err != nil {
			return t.collectTruncateError(err, o, metrics)
		} else if o.hadTruncateError {
			// Upon success we can start emitting metrics normally again but we need to
			// wait for the next collect for the truncate to have taken effect.
			// Emit whatever metrics would be emitted based on the error policy.
			metrics, err = t.collectTruncateError(fmt.Errorf("Truncate successful, waiting for next metric collection."), o, metrics)
		}

		o.hadTruncateError = false
	}

	return metrics, err
}

func (t *Table) collectTruncateError(err error, options *tableOptions, collectedMetrics []blip.MetricValue) ([]blip.MetricValue, error) {
	var ep *errors.Policy = options.errPolicy[ERR_TRUNCATE_FAILED]

	// Mark as having failed a truncate. This is used to
	// skip sending metrics until we have a successful truncate.
	options.hadTruncateError = true

	// Stop trying to collect if error policy retry="stop". This affects
	// future calls to Collect; don't return yet because we need to check
	// the metric policy: drop or zero. If zero, we must report one zero val.
	if ep.Retry == errors.POLICY_RETRY_NO {
		options.stop = true
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
		metrics := make([]blip.MetricValue, 0, len(collectedMetrics))
		for _, existingMetric := range collectedMetrics {
			metrics = append(metrics, blip.MetricValue{
				Type:  existingMetric.Type,
				Name:  existingMetric.Name,
				Value: 0,
			})
		}
	}

	return metrics, reportedErr
}
