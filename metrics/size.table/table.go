// Copyright 2022 Block, Inc.

package sizetable

import (
	"context"
	"database/sql"

	"github.com/cashapp/blip"
	"github.com/cashapp/blip/sqlutil"
)

const (
	DOMAIN = "size.table"

	opt_total         = "total"
	OPT_SCHEMA_FILTER = "schema"
)

// Table collects table sizes for domain size.table.
type Table struct {
	db *sql.DB
	// --
	query map[string]string
	total map[string]bool
}

// Verify collector implements blip.Collector interface.
var _ blip.Collector = &Table{}

// NewTable makes a new Table collector,
func NewTable(db *sql.DB) *Table {
	return &Table{
		db:    db,
		query: map[string]string{},
		total: map[string]bool{},
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
		Description: "Table sizes",
		Options: map[string]blip.CollectorHelpOption{
			opt_total: {
				Name:    opt_total,
				Desc:    "Returns total size of all tables",
				Default: "yes",
				Values: map[string]string{
					"yes": "Includes total size of all tables",
					"no":  "Excludes total size of all tables",
				},
			},
			OPT_SCHEMA_FILTER: {
				Name:    OPT_SCHEMA_FILTER,
				Desc:    "Excludes performance_schema, information_schema, mysql, and sys",
				Default: "yes",
				Values: map[string]string{
					"no":  "Exclude schemas, mysql, and sys",
					"yes": "Include schemas, mysql, and sys",
				},
			},
		},
		Groups: []blip.CollectorKeyValue{
			{Key: "db", Value: "the database name for the corresponding table size, or empty string for all dbs"},
			{Key: "table", Value: "the table name for the corresponding table size, or empty string for all tables"},
		},
		Metrics: []blip.CollectorMetric{
			{
				Name: "bytes",
				Type: blip.GAUGE,
				Desc: "Table size",
			},
		},
	}
}

// Prepare prepares the collector for the given plan.
func (t *Table) Prepare(ctx context.Context, plan blip.Plan) (func(), error) {
LEVEL:
	for _, level := range plan.Levels {
		dom, ok := level.Collect[DOMAIN]
		if !ok {
			continue LEVEL // not collected in this level
		}

		q, err := TableSizeQuery(dom.Options, t.Help())
		if err != nil {
			return nil, err
		}
		t.query[level.Name] = q

		if dom.Options[opt_total] == "yes" {
			t.total[level.Name] = true
		} else {
			t.total[level.Name] = false
		}
	}
	return nil, nil
}

func (t *Table) Collect(ctx context.Context, levelName string) ([]blip.MetricValue, error) {
	q, ok := t.query[levelName]
	if !ok {
		return nil, nil
	}
	rows, err := t.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	metrics := []blip.MetricValue{}

	var (
		dbName  string
		tblName string
		val     string
	)

	total := float64(0)
	for rows.Next() {
		if err = rows.Scan(&dbName, &tblName, &val); err != nil {
			return nil, err
		}

		m := blip.MetricValue{
			Name:  "bytes",
			Type:  blip.GAUGE,
			Group: map[string]string{"db": dbName, "table": tblName},
		}
		var ok bool
		m.Value, ok = sqlutil.Float64(val)
		if !ok {
			continue
		}
		total += m.Value
		metrics = append(metrics, m)
	}

	if t.total[levelName] {
		metrics = append(metrics, blip.MetricValue{
			Name:  "bytes",
			Type:  blip.GAUGE,
			Group: map[string]string{"db": "", "table": ""},
			Value: total,
		})
	}

	return metrics, err
}
