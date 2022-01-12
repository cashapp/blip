package sizebinlog

import (
	"context"
	"database/sql"

	"github.com/cashapp/blip"
	"github.com/cashapp/blip/sqlutil"
)

const (
	DOMAIN = "size.binlog"
)

// Binlog collects metrics for the size.binlog domain. The source is SHOW BINARY LOGS.
type Binlog struct {
	db *sql.DB
	// --
	cols3 bool
}

var _ blip.Collector = &Binlog{}

func NewBinlog(db *sql.DB) *Binlog {
	return &Binlog{
		db: db,
	}
}

func (c *Binlog) Domain() string {
	return DOMAIN
}

func (c *Binlog) Help() blip.CollectorHelp {
	return blip.CollectorHelp{
		Domain:      DOMAIN,
		Description: "Total size of all binary logs in bytes",
		Options:     map[string]blip.CollectorHelpOption{},
		Metrics: []blip.CollectorMetric{
			{
				Name: "bytes",
				Type: blip.GAUGE,
				Desc: "Total size of all binary logs in bytes",
			},
		},
	}
}

func (c *Binlog) Prepare(ctx context.Context, plan blip.Plan) (func(), error) {
	// As of MySQL 8.0.14, SHOW BINARY LOGS has 3 cols instead of 2
	if ok, _ := sqlutil.MySQLVersionGTE("8.0.14", c.db, ctx); ok {
		c.cols3 = true
	}
	return nil, nil
}

func (c *Binlog) Collect(ctx context.Context, levelName string) ([]blip.MetricValue, error) {
	rows, err := c.db.QueryContext(ctx, "SHOW BINARY LOGS")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var (
		name  string
		val   string
		enc   string
		ok    bool
		n     float64
		total float64
	)
	for rows.Next() {
		if c.cols3 {
			err = rows.Scan(&name, &val, &enc) // 8.0.14+
		} else {
			err = rows.Scan(&name, &val)
		}
		if err != nil {
			return nil, err
		}
		n, ok = sqlutil.Float64(val)
		if !ok {
			continue
		}
		total += n
	}

	metrics := []blip.MetricValue{{
		Name:  "bytes",
		Value: total,
		Type:  blip.GAUGE,
	}}

	return metrics, err
}
