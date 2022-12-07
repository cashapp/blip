// Copyright 2022 Block, Inc.

package iotable

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/cashapp/blip"
)

const (
	DOMAIN = "io.table"

	OPT_EXCLUDE        = "exclude"
	OPT_INCLUDE        = "include"
	OPT_TRUNCATE_TABLE = "truncate-table"
	OPT_ALL            = "all"

	TRUNCATE_QUERY = "TRUNCATE TABLE performance_schema.table_io_waits_summary_by_table"
)

var (
	metric_names = []string{
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

	metric_map map[string]struct{}
)

func init() {
	metric_map = make(map[string]struct{}, len(metric_names))
	for _, name := range metric_names {
		metric_map[name] = struct{}{}
	}
}

type tableOptions struct {
	query    string
	truncate bool
}

// Table collects table io for domain io.table.
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
		Description: "Table IO",
		Options: map[string]blip.CollectorHelpOption{
			OPT_INCLUDE: {
				Name: OPT_INCLUDE,
				Desc: "Comma-separated list of database or table names to include (overrides option " + OPT_EXCLUDE + ")",
			},
			OPT_EXCLUDE: {
				Name:    OPT_EXCLUDE,
				Desc:    "Comma-separated list of database or table names to exclude (ignored if " + OPT_INCLUDE + " is set)",
				Default: "mysql.*,information_schema.*,performance_schema.*,sys.*",
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
			dom.Options[OPT_EXCLUDE] = "mysql.*,information_schema.*,performance_schema.*,sys.*"
		}

		q, err := TableIoQuery(dom.Options, dom.Metrics)
		if err != nil {
			return nil, err
		}

		o.query = q

		if truncate, ok := dom.Options[OPT_TRUNCATE_TABLE]; ok && truncate == "no" {
			o.truncate = false
		} else {
			o.truncate = true // default
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
		return nil, fmt.Errorf("Failed to get columns for io.table: %v", err)
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
		_, err := t.db.Exec(TRUNCATE_QUERY)
		if err != nil {
			return nil, err
		}
	}

	return metrics, err
}
