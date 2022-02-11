// Copyright 2022 Block, Inc.

package repllag

import (
	"context"
	"database/sql"
	"time"

	"github.com/cashapp/blip"
	"github.com/cashapp/blip/heartbeat"
	"github.com/cashapp/blip/sqlutil"
)

const (
	DOMAIN = "repl.lag"

	OPT_AUTO        = "auto"
	OPT_SOURCE_ID   = "source-id"
	OPT_SOURCE_ROLE = "source-role"
	OPT_TABLE       = "table"
	OPT_WRITER      = "writer"
	OPT_REPL_CHECK  = "repl-check"
)

type Lag struct {
	db        *sql.DB
	lagReader heartbeat.Reader
	atLevel   map[string]bool
}

var _ blip.Collector = &Lag{}

func NewLag(db *sql.DB) *Lag {
	return &Lag{
		db:      db,
		atLevel: map[string]bool{},
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
			OPT_AUTO: {
				Name:    OPT_AUTO,
				Desc:    "Try to work if configure",
				Default: "yes",
				Values: map[string]string{
					"yes": "Enable if configured, else disable (no error)",
					"no":  "Error unless configured",
				},
			},
			OPT_WRITER: {
				Name:    OPT_WRITER,
				Desc:    "Type of heartbeat writer",
				Default: "blip",
				Values: map[string]string{
					"blip": "Native Blip replication lag",
					//"pt-heartbeat": "Percona pt-heartbeat",
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
				Desc: "Source ID as reported by heartbeat writer; mutually exclusive with " + OPT_SOURCE_ROLE + " (suggested: %%{monitor.meta.repl-source-id})",
			},
			OPT_SOURCE_ROLE: {
				Name: OPT_SOURCE_ROLE,
				Desc: "Source role as reported by heartbeat writer; mutually exclusive with " + OPT_SOURCE_ID + " (suggested: %%{monitor.meta.repl-source-role})",
			},
			OPT_REPL_CHECK: {
				Name: OPT_REPL_CHECK,
				Desc: "MySQL global variable to check if instance is a replica",
			},
		},
		Metrics: []blip.CollectorMetric{
			{
				Name: "max",
				Type: blip.GAUGE,
				Desc: "Maximum replication lag (milliseconds) during collect interval",
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

		sourceId := dom.Options[OPT_SOURCE_ID]
		sourceRole := dom.Options[OPT_SOURCE_ROLE]
		if sourceId == "" && sourceRole == "" {
			blip.Debug("%s: no source id or role, ignoring", plan.MonitorId)
			continue
		}

		table := dom.Options[OPT_TABLE]
		if table == "" {
			table = blip.DEFAULT_HEARTBEAT_TABLE
		}

		switch dom.Options[OPT_WRITER] {
		case "", "blip": // "" == default == blip
			if c.lagReader == nil {
				// Only 1 reader per plan
				c.lagReader = heartbeat.NewBlipReader(heartbeat.BlipReaderArgs{
					MonitorId:  plan.MonitorId,
					DB:         c.db,
					Table:      table,
					SourceId:   sourceId,
					SourceRole: sourceRole,
					ReplCheck:  sqlutil.CleanObjectName(dom.Options[OPT_REPL_CHECK]),             // @todo sanitize better
					Waiter:     &heartbeat.SlowFastWaiter{NetworkLatency: 50 * time.Millisecond}, // @todo OPT_NETWORK_LATENCY
				})
				go c.lagReader.Start()
				blip.Debug("started heartbeat.Reader for %s %s", plan.Name, level.Name)
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
	lag, _, err := c.lagReader.Lag(ctx)
	if err != nil {
		return nil, err
	}
	if lag == heartbeat.NOT_A_REPLICA {
		return nil, nil
	}
	m := blip.MetricValue{
		Name:  "lag",
		Type:  blip.GAUGE,
		Value: float64(lag),
	}
	return []blip.MetricValue{m}, nil
}
