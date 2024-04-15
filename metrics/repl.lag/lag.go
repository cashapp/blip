// Copyright 2022 Block, Inc.

package repllag

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/cashapp/blip"
	"github.com/cashapp/blip/heartbeat"
	"github.com/cashapp/blip/sqlutil"
)

const (
	DOMAIN = "repl.lag"

	OPT_HEARTBEAT_SOURCE_ID   = "heartbeat-source-id"
	OPT_HEARTBEAT_SOURCE_ROLE = "heartbeat-source-role"
	OPT_HEARTBEAT_TABLE       = "heartbeat-table"
	OPT_LAG_SOURCE            = "lag-source"
	OPT_REPL_CHECK            = "repl-check"
	OPT_REPORT_NO_HEARTBEAT   = "report-no-heartbeat"
	OPT_REPORT_NOT_A_REPLICA  = "report-not-a-replica"
	OPT_NETWORK_LATENCY       = "network-latency"

	LAG_SOURCE_BLIP = "blip"
	LAG_SOURCE_PFS  = "pfs"

	// MySQL8LagQuery is the query to calculate approximate lag
	// from replication worker stats in performance schema
	// This is only available in MySQL 8 (and above) and when performance schema is enabled
	MySQL8LagQuery = `SELECT TIMESTAMPDIFF(MICROSECOND,
max(LAST_APPLIED_TRANSACTION_ORIGINAL_COMMIT_TIMESTAMP),
max(LAST_APPLIED_TRANSACTION_END_APPLY_TIMESTAMP)
)/1000
FROM performance_schema.replication_applier_status_by_worker`
)

type Lag struct {
	db              *sql.DB
	lagReader       heartbeat.Reader
	lagSourceIn     map[string]string
	dropNoHeartbeat map[string]bool
	dropNotAReplica map[string]bool
	replCheck       string
}

var _ blip.Collector = &Lag{}

func NewLag(db *sql.DB) *Lag {
	return &Lag{
		db:              db,
		lagSourceIn:     map[string]string{},
		dropNoHeartbeat: map[string]bool{},
		dropNotAReplica: map[string]bool{},
	}
}

func (c *Lag) Domain() string {
	return DOMAIN
}

func (c *Lag) Help() blip.CollectorHelp {
	return blip.CollectorHelp{
		Domain:      DOMAIN,
		Description: "Replication lag",
		Options: map[string]blip.CollectorHelpOption{
			OPT_LAG_SOURCE: {
				Name:    OPT_LAG_SOURCE,
				Desc:    "How to collect Lag from",
				Default: "auto",
				Values: map[string]string{
					"auto": "Auto-determine best source",
					"blip": "Native Blip heartbeat replication lag",
					"pfs":  "MySQL8 Performance Schema",
					///"legacy": "Second_Behind_Slave|Replica from SHOW SHOW|REPLICA STATUS",
				},
			},
			OPT_HEARTBEAT_TABLE: {
				Name:    OPT_HEARTBEAT_TABLE,
				Desc:    "Heartbeat table",
				Default: blip.DEFAULT_HEARTBEAT_TABLE,
			},
			OPT_HEARTBEAT_SOURCE_ID: {
				Name: OPT_HEARTBEAT_SOURCE_ID,
				Desc: "Source ID as reported by heartbeat writer; mutually exclusive with " + OPT_HEARTBEAT_SOURCE_ROLE,
			},
			OPT_HEARTBEAT_SOURCE_ROLE: {
				Name: OPT_HEARTBEAT_SOURCE_ROLE,
				Desc: "Source role as reported by heartbeat writer; mutually exclusive with " + OPT_HEARTBEAT_SOURCE_ID,
			},
			OPT_REPL_CHECK: {
				Name: OPT_REPL_CHECK,
				Desc: "MySQL global variable (without @@) to check if instance is a replica",
			},
			OPT_REPORT_NO_HEARTBEAT: {
				Name:    OPT_REPORT_NO_HEARTBEAT,
				Desc:    "Report no heartbeat as -1",
				Default: "no",
				Values: map[string]string{
					"yes": "Enabled: report no heartbeat as repl.lag.current = -1",
					"no":  "Disabled: drop repl.lag.current if no heartbeat",
				},
			},
			OPT_REPORT_NOT_A_REPLICA: {
				Name:    OPT_REPORT_NOT_A_REPLICA,
				Desc:    "Report not a replica as -1",
				Default: "no",
				Values: map[string]string{
					"yes": "Enabled: report not a replica repl.lag.current = -1",
					"no":  "Disabled: drop repl.lag.current if not a replica",
				},
			},
			OPT_NETWORK_LATENCY: {
				Name:    OPT_NETWORK_LATENCY,
				Desc:    "Network latency (milliseconds)",
				Default: "50",
			},
		},
		Metrics: []blip.CollectorMetric{
			{
				Name: "current",
				Type: blip.GAUGE,
				Desc: "Current replication lag (milliseconds)",
			},
		},
	}
}

// Prepare prepares lag collectors for all levels in the plan that contain the "repl.lag" domain
func (c *Lag) Prepare(ctx context.Context, plan blip.Plan) (func(), error) {
LEVEL:
	for levelName, level := range plan.Levels {
		dom, ok := level.Collect[DOMAIN]
		if !ok {
			continue LEVEL // not collected in this level
		}
		c.dropNotAReplica[levelName] = !blip.Bool(dom.Options[OPT_REPORT_NOT_A_REPLICA])
		c.replCheck = sqlutil.CleanObjectName(dom.Options[OPT_REPL_CHECK]) // @todo sanitize better

		if source, ok := dom.Options[OPT_LAG_SOURCE]; ok {
			if len(source) > 0 && source != "auto" {
				switch source {
				case LAG_SOURCE_PFS:
					c.lagSourceIn[levelName] = LAG_SOURCE_PFS
					// Try collecting, discard metrics
					_, err := c.collectPFS(ctx, levelName)
					return nil, err
				case LAG_SOURCE_BLIP:
					return c.prepareBlip(levelName, plan.MonitorId, plan.Name, dom.Options)
				default:
					return nil, fmt.Errorf("invalid lag source: %s; valid values: auto, pfs, blip", source)
				}
			}
		}

		// -------------------------------------------------------------------------
		// Auto source (default)
		// -------------------------------------------------------------------------
		// Try PFS first
		var err error
		if _, err = c.collectPFS(ctx, levelName); err == nil {
			c.lagSourceIn[levelName] = LAG_SOURCE_PFS
			return nil, nil
		}

		// then Blip HeartBeat
		cleanup, err := c.prepareBlip(levelName, plan.MonitorId, plan.Name, dom.Options)
		if err == nil {
			return cleanup, nil
		}

		return nil, fmt.Errorf("auto lag source failed, last error: %s", err)
	}
	return nil, nil
}

func (c *Lag) Collect(ctx context.Context, levelName string) ([]blip.MetricValue, error) {
	switch c.lagSourceIn[levelName] {
	case LAG_SOURCE_BLIP:
		return c.collectBlip(ctx, levelName)
	case LAG_SOURCE_PFS:
		return c.collectPFS(ctx, levelName)
	}

	panic(fmt.Sprintf("invalid lag source in Collect %s", c.lagSourceIn[levelName]))
}

// //////////////////////////////////////////////////////////////////////////
// Internal methods
// //////////////////////////////////////////////////////////////////////////

func (c *Lag) prepareBlip(levelName string, monitorID string, planName string, options map[string]string) (func(), error) {
	if c.lagReader != nil {
		return nil, nil
	}

	c.dropNoHeartbeat[levelName] = !blip.Bool(options[OPT_REPORT_NO_HEARTBEAT])

	table := options[OPT_HEARTBEAT_TABLE]
	if table == "" {
		table = blip.DEFAULT_HEARTBEAT_TABLE
	}
	netLatency := 50 * time.Millisecond
	if s, ok := options[OPT_NETWORK_LATENCY]; ok {
		n, err := strconv.Atoi(s)
		if err != nil {
			blip.Debug("%s: invalid network-latency: %s: %s (ignoring; using default 50 ms)", monitorID, s, err)
		} else {
			netLatency = time.Duration(n) * time.Millisecond
		}
	}
	// Only 1 reader per plan
	c.lagReader = heartbeat.NewBlipReader(heartbeat.BlipReaderArgs{
		MonitorId:  monitorID,
		DB:         c.db,
		Table:      table,
		SourceId:   options[OPT_HEARTBEAT_SOURCE_ID],
		SourceRole: options[OPT_HEARTBEAT_SOURCE_ROLE],
		ReplCheck:  c.replCheck,
		Waiter: heartbeat.SlowFastWaiter{
			MonitorId:      monitorID,
			NetworkLatency: netLatency,
		},
	})
	go c.lagReader.Start()
	blip.Debug("%s: started reader: %s/%s (network latency: %s)", monitorID, planName, levelName, netLatency)
	c.lagSourceIn[levelName] = LAG_SOURCE_BLIP
	var cleanup func()
	if c.lagReader != nil {
		cleanup = func() {
			blip.Debug("%s: stopping reader", monitorID)
			c.lagReader.Stop()
		}
	}
	return cleanup, nil
}

func (c *Lag) collectBlip(ctx context.Context, levelName string) ([]blip.MetricValue, error) {
	lag, err := c.lagReader.Lag(ctx)
	if err != nil {
		return nil, err
	}
	if !lag.Replica {
		if c.dropNotAReplica[levelName] {
			return nil, nil
		}
	} else if lag.Milliseconds == -1 && c.dropNoHeartbeat[levelName] {
		return nil, nil
	}
	m := blip.MetricValue{
		Name:  "current",
		Type:  blip.GAUGE,
		Value: float64(lag.Milliseconds),
		Meta:  map[string]string{"source": lag.SourceId},
	}
	return []blip.MetricValue{m}, nil
}

func (c *Lag) collectPFS(ctx context.Context, levelName string) ([]blip.MetricValue, error) {
	var defaultLag []blip.MetricValue
	if c.dropNotAReplica[levelName] {
		defaultLag = nil
	} else {
		// send -1 for lag
		m := blip.MetricValue{
			Name:  "current",
			Type:  blip.GAUGE,
			Value: float64(-1),
		}
		defaultLag = []blip.MetricValue{m}
	}

	// if isReplCheck is supplied, check if it's a replica
	isRepl := 1
	if c.replCheck != "" {
		query := "SELECT @@" + c.replCheck
		if err := c.db.QueryRowContext(ctx, query).Scan(&isRepl); err != nil {
			return nil, fmt.Errorf("checking if instance is replica failed, please check value of %s. Err: %s", OPT_REPL_CHECK, err.Error())
		}
	}

	if isRepl == 0 {
		return defaultLag, nil
	}
	// instance is a replica or replCheck is not set

	var lagValue sql.NullString
	if err := c.db.QueryRowContext(ctx, MySQL8LagQuery).Scan(&lagValue); err != nil {
		return nil, fmt.Errorf("could not check replication lag, check that this is a MySQL 8.0 replica, and that performance_schema is enabled. Err: %s", err.Error())
	}
	if !lagValue.Valid {
		// it is a MySQL 8 instance and performance schema is enabled, otherwise the query would have returned error
		// if replCheck is empty, we can assume based on the query that it's not a replica and return nil or -1
		if c.replCheck == "" {
			return defaultLag, nil
		} else {
			// it's a replica, so lagValue should be valid, raise error
			return nil, fmt.Errorf("instance is a MySQL 8 replica and performance schema is enabled, still could not calculate lag, please check manually. Lag: %q", lagValue.String)
		}
	}

	f, ok := sqlutil.Float64(lagValue.String)
	if !ok {
		return nil, fmt.Errorf("couldn't convert replica lag from performance schema into float. Lag: %s", lagValue.String)
	}
	m := blip.MetricValue{
		Name:  "current",
		Type:  blip.GAUGE,
		Value: f,
	}
	return []blip.MetricValue{m}, nil
}
