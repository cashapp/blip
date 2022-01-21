// Copyright 2022 Block, Inc.

package repllag

import (
	"context"
	"database/sql"
	"time"

	"github.com/cashapp/blip"
	"github.com/cashapp/blip/heartbeat"
)

const (
	DOMAIN = "repl.lag"

	OPT_AUTO   = "auto"
	OPT_WRITER = "writer"
	OPT_TABLE  = "table"
	OPT_SOURCE = "source"

	DEFAULT_WRITER = "blip"
	DEFAULT_TABLE  = "blip.heartbeat"
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
				Default: DEFAULT_WRITER,
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
				Default: DEFAULT_TABLE,
			},
			OPT_SOURCE: {
				Name: OPT_SOURCE,
				Desc: "Source MySQL instance (suggested: %%{monitor.meta.repl-source})",
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

		source := dom.Options[OPT_SOURCE]
		if source == "" {
			blip.Debug("%s: no source, ignoring", plan.MonitorId)
			continue
		}

		table := dom.Options[OPT_TABLE]
		if table == "" {
			table = DEFAULT_TABLE
		}

		switch dom.Options[OPT_WRITER] {
		case DEFAULT_WRITER, "":
			if c.lagReader == nil {
				// Only 1 reader per plan
				c.lagReader = heartbeat.NewBlipReader(
					c.db,
					table,
					source,
					&heartbeat.SlowFastWaiter{NetworkLatency: 50 * time.Millisecond}, // todo OPT_NETWORK_LATENCY
				)
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
	m := blip.MetricValue{
		Name:  "lag",
		Type:  blip.GAUGE,
		Value: float64(lag),
	}
	return []blip.MetricValue{m}, nil
}
