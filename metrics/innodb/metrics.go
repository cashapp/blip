package innodb

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

/*
mysql> SELECT * FROM innodb_metrics WHERE name='trx_rseg_history_len' LIMIT 1\G
*************************** 1. row ***************************
           NAME: trx_rseg_history_len
      SUBSYSTEM: transaction
          COUNT: 0
      MAX_COUNT: 0
      MIN_COUNT: 0
      AVG_COUNT: NULL
    COUNT_RESET: 0
MAX_COUNT_RESET: 0
MIN_COUNT_RESET: 0
AVG_COUNT_RESET: NULL
   TIME_ENABLED: 2021-08-17 08:24:14
  TIME_DISABLED: NULL
   TIME_ELAPSED: 1905927
     TIME_RESET: NULL
         STATUS: enabled
           TYPE: value
        COMMENT: Length of the TRX_RSEG_HISTORY list
*/

// Metrics collects global system variables for the var.global domain.
type Metrics struct {
	monitorId string
	db        *sql.DB
	plans     blip.Plan
	query     map[string]string
}

func NewMetrics(db *sql.DB) *Metrics {
	return &Metrics{
		db:    db,
		query: map[string]string{},
	}
}

const (
	blip_domain = "innodb"
)

func (c *Metrics) Domain() string {
	return blip_domain
}

func (c *Metrics) Help() blip.CollectorHelp {
	return blip.CollectorHelp{
		Domain:      blip_domain,
		Description: "Collect Metrics metrics (information_schema.innodb_metrics)",
		Options: map[string]blip.CollectorHelpOption{
			OPT_ALL: {
				Name:    OPT_ALL,
				Desc:    "Collect all metrics",
				Default: "no",
				Values: map[string]string{
					"yes":     "All metrics (ignore metrics list)",
					"enabled": "Enabled metrics (ignore metrics list)",
					"no":      "Specified metrics",
				},
			},
		},
	}
}

const (
	base = "SELECT subsystem, name, count FROM information_schema.innodb_metrics"
)

// Prepares queries for all levels in the plan that contain the "var.global" domain
func (c *Metrics) Prepare(plan blip.Plan) error {
LEVEL:
	for _, level := range plan.Levels {
		dom, ok := level.Collect[blip_domain]
		if !ok {
			continue LEVEL // not collected in this level
		}

		all := strings.ToLower(dom.Options[OPT_ALL])
		switch all {
		case "all":
			c.query[level.Name] = base
		case "enabled":
			c.query[level.Name] = base + " WHERE status='enabled'"
		default:
			c.query[level.Name] = base + " WHERE name IN (" + sqlutil.INList(dom.Metrics, "'") + ")"
		}
	}
	return nil
}

func (c *Metrics) Collect(ctx context.Context, levelName string) ([]blip.MetricValue, error) {
	q := c.query[levelName]
	rows, err := c.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	metrics := []blip.MetricValue{}

	var (
		subsystem string
		name      string
		val       string
		ok        bool
	)
	for rows.Next() {
		m := blip.MetricValue{Type: blip.COUNTER}

		if err = rows.Scan(&subsystem, &name, &val); err != nil {
			continue
		}

		m.Name = strings.ToLower(name)
		m.Tags = map[string]string{"subsystem": subsystem}

		m.Value, ok = sqlutil.Float64(val)
		if !ok {
			// log.Printf("Error parsing the metric: %s value: %s as float %s", m.Name, val, err)
			// Log error and continue to next row to retrieve next metric
			continue
		}

		// Fixed as of 8.0.17: http://bugs.mysql.com/bug.php?id=75966
		if m.Value < 0 {
			m.Value = 0
		}

		if gauge[m.Name] {
			m.Type = blip.GAUGE
		}
		metrics = append(metrics, m)
	}

	return metrics, err
}

var gauge = map[string]bool{
	"buffer_pool_bytes_data":         true,
	"buffer_pool_bytes_dirty":        true,
	"buffer_pool_pages_data":         true,
	"buffer_pool_pages_dirty":        true,
	"buffer_pool_pages_free":         true,
	"buffer_pool_pages_misc":         true,
	"buffer_pool_pages_total":        true,
	"buffer_pool_size":               true,
	"ddl_pending_alter_table":        true,
	"file_num_open_files":            true,
	"innodb_page_size":               true,
	"lock_row_lock_time_avg":         true,
	"lock_row_lock_time_max":         true,
	"lock_threads_waiting":           true,
	"log_lsn_archived":               true,
	"log_lsn_buf_dirty_pages_added":  true,
	"log_lsn_buf_pool_oldest_approx": true,
	"log_lsn_buf_pool_oldest_lwm":    true,
	"log_lsn_checkpoint_age":         true,
	"log_lsn_current":                true,
	"log_lsn_last_checkpoint":        true,
	"log_lsn_last_flush":             true,
	"log_max_modified_age_async":     true,
	"log_max_modified_age_sync":      true,
	"os_log_pending_fsyncs":          true,
	"os_log_pending_writes":          true,
	"os_pending_reads":               true,
	"os_pending_writes":              true,
	"purge_dml_delay_usec":           true,
	"purge_resume_count":             true,
	"purge_stop_count":               true,
	"lock_row_lock_current_waits":    true,
	"trx_active_transactions":        true, // counter according to i_s.innodb_metrics.comment
	"trx_rseg_current_size":          true, // rseg size in pages
	"trx_rseg_history_len":           true, // history list length
}
