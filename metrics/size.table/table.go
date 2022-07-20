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

	OPT_ALL = "all"

	TABLE_SIZES_QUERY = `
    SELECT table_schema AS db, table_name as tbl,
           data_length + index_length AS tbl_size_bytes
      FROM information_schema.TABLES
     WHERE table_schema NOT IN ('performance_schema', 'information_schema', 'mysql');`
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
			OPT_ALL: {
				Name:    OPT_ALL,
				Desc:    "Returns total sizes of all tables in all databses",
				Default: "yes",
				Values: map[string]string{
					"yes": "Collect all table sizes",
				},
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

		t.query[level.Name] = TABLE_SIZES_QUERY
		if dom.Options[OPT_ALL] == "yes" {
			t.total[level.Name] = true
		}
	}
	return nil, nil
}

func (t *Table) Collect(ctx context.Context, levelName string) ([]blip.MetricValue, error) {
	if !t.total[levelName] {
		return nil, nil
	}

	rows, err := t.db.QueryContext(ctx, TABLE_SIZES_QUERY)
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

	metrics = append(metrics, blip.MetricValue{
		Name:  "bytes",
		Type:  blip.GAUGE,
		Group: map[string]string{"db": "", "table": ""},
		Value: total,
	})

	return metrics, err
}
