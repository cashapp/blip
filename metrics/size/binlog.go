package size

import (
	"context"
	"database/sql"
	"time"

	"github.com/hashicorp/go-version"

	"github.com/square/blip"
	"github.com/square/blip/collect"
)

const (
	binlog_domain = "size.binlogs"
)

// Binlogs collects data sizes for domain size.data.
type Binlogs struct {
	monitorId string
	db        *sql.DB
	cols3     bool
}

func NewBinlogs(db *sql.DB) *Binlogs {
	return &Binlogs{
		db: db,
	}
}

func (c *Binlogs) Domain() string {
	return binlog_domain
}

func (c *Binlogs) Help() collect.Help {
	return collect.Help{
		Domain:      binlog_domain,
		Description: "Collect size of binary logs",
		Options:     map[string]collect.HelpOption{},
	}
}

// Prepares queries for all levels in the plan that contain the "var.global" domain
func (c *Binlogs) Prepare(plan collect.Plan) error {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	var val string
	err := c.db.QueryRowContext(ctx, "SELECT @@version").Scan(&val)
	if err != nil {
		return err
	}

	v8014, _ := version.NewVersion("8.0.14")
	v, _ := version.NewVersion(val)
	if v.GreaterThanOrEqual(v8014) {
		c.cols3 = true
	}

	return nil
}

func (c *Binlogs) Collect(ctx context.Context, levelName string) ([]blip.MetricValue, error) {
	rows, err := c.db.QueryContext(ctx, "SHOW BINARY LOGS")
	if err != nil {
		// @todo Monitor should report this, maybe in status
		return nil, err
	}
	defer rows.Close()

	var name string
	var val string
	var enc string
	var ok bool
	var n float64
	var total float64
	for rows.Next() {
		if c.cols3 {
			err = rows.Scan(&name, &val, &enc) // 8.0.14+
		} else {
			err = rows.Scan(&name, &val)
		}
		if err != nil {
			blip.Debug(err.Error())
			continue
		}

		n, ok = collect.Float64(val)
		if !ok {
			continue
		}
		total += n
	}

	metrics := []blip.MetricValue{
		{
			Name:  "bytes",
			Value: total,
			Type:  blip.GAUGE,
		}}

	return metrics, err
}
