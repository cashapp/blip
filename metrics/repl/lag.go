package repl

import (
	"context"
	"database/sql"
	"time"

	"github.com/cashapp/blip"
	"github.com/cashapp/blip/heartbeat"
)

const (
	DOMAIN = "repl"

	OPT_AUTO   = "auto"
	OPT_WRITER = "writer"
	OPT_TABLE  = "table"
	OPT_SOURCE = "source"

	DEFAULT_WRITER = "blip"
	DEFAULT_TABLE  = "blip.heartbeat"
)

type Repl struct {
	db        *sql.DB
	lagReader map[string]heartbeat.Reader
	enabled   map[string]bool
}

func NewRepl(db *sql.DB) *Repl {
	return &Repl{
		db:        db,
		lagReader: map[string]heartbeat.Reader{},
		enabled:   map[string]bool{},
	}
}

func (c *Repl) Domain() string {
	return DOMAIN
}

func (c *Repl) Help() blip.CollectorHelp {
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
				Name:    OPT_SOURCE,
				Desc:    "Source MySQL instance",
				Default: "%%{monitor.meta.repl-source}",
			},
		},
		Metrics: []blip.CollectorMetric{
			{
				Name: "lag",
				Type: blip.GAUGE,
				Desc: "Replication lag (milliseconds)",
			},
		},
	}
}

// Prepares queries for all levels in the plan that contain the "var.global" domain
func (c *Repl) Prepare(ctx context.Context, plan blip.Plan) error {

	// Stop and remove all readers from previous plans, if any
	heartbeat.RemoveReaders(c.db)

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
			r := heartbeat.NewBlipReader(
				c.db,
				table,
				source,
				&heartbeat.SlowFastWaiter{NetworkLatency: 50 * time.Millisecond}, // todo OPT_NETWORK_LATENCY
			)
			c.lagReader[level.Name] = r
			go r.Start()
			heartbeat.AddReader(r, c.db, plan.Name, level.Name, dom.Options[OPT_WRITER])
		}
	}

	return nil
}

func (c *Repl) Collect(ctx context.Context, levelName string) ([]blip.MetricValue, error) {
	r := c.lagReader[levelName]
	if r == nil {
		return nil, nil
	}
	lag, _, err := r.Lag(ctx)
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
