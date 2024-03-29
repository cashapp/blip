// Copyright 2022 Block, Inc.

package repllag

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	"github.com/cashapp/blip"
	"github.com/cashapp/blip/heartbeat"
	"github.com/cashapp/blip/sqlutil"
)

const (
	DOMAIN = "repl.lag"

	OPT_SOURCE_ID            = "source-id"
	OPT_SOURCE_ROLE          = "source-role"
	OPT_TABLE                = "table"
	OPT_WRITER               = "writer"
	OPT_REPL_CHECK           = "repl-check"
	OPT_REPORT_NO_HEARTBEAT  = "report-no-heartbeat"
	OPT_REPORT_NOT_A_REPLICA = "report-not-a-replica"
	OPT_NETWORK_LATENCY      = "network-latency"
)

type Lag struct {
	db              *sql.DB
	lagReader       heartbeat.Reader
	atLevel         map[string]bool
	dropNoHeartbeat map[string]bool
	dropNotAReplica map[string]bool
}

var _ blip.Collector = &Lag{}

func NewLag(db *sql.DB) *Lag {
	return &Lag{
		db:              db,
		atLevel:         map[string]bool{},
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
			OPT_WRITER: {
				Name:    OPT_WRITER,
				Desc:    "Type of heartbeat writer",
				Default: "blip",
				Values: map[string]string{
					"blip": "Native Blip replication lag",
					//"pfs":    "MySQL Performance Schema",
					///"legacy": "Second_Behind_Slave|Replica from SHOW SHOW|REPLICA STATUS",
				},
			},
			OPT_TABLE: {
				Name:    OPT_TABLE,
				Desc:    "Heartbeat table",
				Default: blip.DEFAULT_HEARTBEAT_TABLE,
			},
			OPT_SOURCE_ID: {
				Name: OPT_SOURCE_ID,
				Desc: "Source ID as reported by heartbeat writer; mutually exclusive with " + OPT_SOURCE_ROLE,
			},
			OPT_SOURCE_ROLE: {
				Name: OPT_SOURCE_ROLE,
				Desc: "Source role as reported by heartbeat writer; mutually exclusive with " + OPT_SOURCE_ID,
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

func (c *Lag) Prepare(ctx context.Context, plan blip.Plan) (func(), error) {
LEVEL:
	for _, level := range plan.Levels {
		dom, ok := level.Collect[DOMAIN]
		if !ok {
			continue LEVEL // not collected in this level
		}

		table := dom.Options[OPT_TABLE]
		if table == "" {
			table = blip.DEFAULT_HEARTBEAT_TABLE
		}

		c.dropNoHeartbeat[level.Name] = !blip.Bool(dom.Options[OPT_REPORT_NO_HEARTBEAT])
		c.dropNotAReplica[level.Name] = !blip.Bool(dom.Options[OPT_REPORT_NOT_A_REPLICA])

		netLatency := 50 * time.Millisecond
		if s, ok := dom.Options[OPT_NETWORK_LATENCY]; ok {
			n, err := strconv.Atoi(s)
			if err != nil {
				blip.Debug("%s: invalid network-latency: %s: %s (ignoring; using default 50 ms)", plan.MonitorId, s, err)
			} else {
				netLatency = time.Duration(n) * time.Millisecond
			}
		}

		switch dom.Options[OPT_WRITER] {
		case "", "blip": // "" == default == blip
			if c.lagReader == nil {
				// Only 1 reader per plan
				c.lagReader = heartbeat.NewBlipReader(heartbeat.BlipReaderArgs{
					MonitorId:  plan.MonitorId,
					DB:         c.db,
					Table:      table,
					SourceId:   dom.Options[OPT_SOURCE_ID],
					SourceRole: dom.Options[OPT_SOURCE_ROLE],
					ReplCheck:  sqlutil.CleanObjectName(dom.Options[OPT_REPL_CHECK]), // @todo sanitize better
					Waiter: heartbeat.SlowFastWaiter{
						MonitorId:      plan.MonitorId,
						NetworkLatency: netLatency,
					},
				})
				go c.lagReader.Start()
				blip.Debug("%s: started reader: %s/%s (network latency: %s)", plan.MonitorId, plan.Name, level.Name, netLatency)
			}
			c.atLevel[level.Name] = true
		}
	}

	var cleanup func()
	if c.lagReader != nil {
		cleanup = func() {
			blip.Debug("%s: stopping reader", plan.MonitorId)
			c.lagReader.Stop()
		}
	}

	return cleanup, nil
}

func (c *Lag) Collect(ctx context.Context, levelName string) ([]blip.MetricValue, error) {
	if !c.atLevel[levelName] {
		return nil, nil
	}
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
