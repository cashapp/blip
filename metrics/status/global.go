package status

import (
	"context"
	"database/sql"
	"strings"

	"github.com/square/blip"
	"github.com/square/blip/sqlutil"
)

const (
	OPT_ALL = "all"
)

// Global collects global system variables for the var.global domain.
type Global struct {
	monitorId string
	db        *sql.DB
	plans     blip.Plan
	keep      map[string]map[string]bool
	all       map[string]bool
}

func NewGlobal(db *sql.DB) *Global {
	return &Global{
		db:   db,
		keep: map[string]map[string]bool{},
		all:  map[string]bool{},
	}
}

const (
	blip_domain = "status.global"
)

func (c *Global) Domain() string {
	return blip_domain
}

func (c *Global) Help() blip.CollectorHelp {
	return blip.CollectorHelp{
		Domain:      blip_domain,
		Description: "Collect global status variables (sysvars)",
		Options: map[string]blip.CollectorHelpOption{
			OPT_ALL: {
				Name: OPT_ALL,
				Desc: "Collect all sysvars",
				Values: map[string]string{
					"yes": "Enable",
					"no":  "Disable",
				},
			},
		},
	}
}

// Prepares queries for all levels in the plan that contain the "var.global" domain
func (c *Global) Prepare(plan blip.Plan) error {
LEVEL:
	for _, level := range plan.Levels {
		dom, ok := level.Collect[blip_domain]
		if !ok {
			continue LEVEL // not collected in this level
		}

		if all, ok := dom.Options[OPT_ALL]; ok && all == "yes" {
			c.all[level.Name] = true
		} else {
			metrics := make(map[string]bool, len(dom.Metrics))
			for i := range dom.Metrics {
				metrics[dom.Metrics[i]] = true
			}
			c.keep[level.Name] = metrics
		}
	}
	return nil
}

func (c *Global) Collect(ctx context.Context, levelName string) ([]blip.MetricValue, error) {
	rows, err := c.db.QueryContext(ctx, "SHOW GLOBAL STATUS")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	filter := !c.all[levelName]

	metrics := []blip.MetricValue{}

	var val string
	var name string
	var ok bool
	for rows.Next() {
		m := blip.MetricValue{Type: blip.COUNTER}

		if err = rows.Scan(&name, &val); err != nil {
			continue
		}

		m.Name = strings.ToLower(name)

		if filter && !c.keep[levelName][m.Name] {
			continue
		}

		m.Value, ok = sqlutil.Float64(val)
		if !ok {
			// log.Printf("Error parsing the metric: %s value: %s as float %s", m.Name, val, err)
			// Log error and continue to next row to retrieve next metric
			continue
		}

		if gauge[m.Name] {
			m.Type = blip.GAUGE
		}
		metrics = append(metrics, m)
	}

	return metrics, err
}

var gauge = map[string]bool{
	"threads_running":                true,
	"threads_connected":              true,
	"prepared_stmt_count":            true,
	"innodb_buffer_pool_pages_dirty": true,
	"innodb_buffer_pool_pages_free":  true,
	"innodb_buffer_pool_pages_total": true,
	"innodb_row_lock_current_waits":  true,
	"innodb_os_log_pending_writes":   true,
}
