package repl

import (
	"context"
	"database/sql"
	"time"

	"github.com/cashapp/blip"
	"github.com/cashapp/blip/heartbeat"
)

type LagReader interface {
	Lag(context.Context) (int64, time.Time, error)
}

const (
	OPT_WRITER = "writer"
	OPT_TABLE  = "table"
	OPT_SOURCE = "source"
)

// Lag collects global system variables for the var.global domain.
type Lag struct {
	db      *sql.DB
	atLevel map[string]LagReader
}

func NewLag(db *sql.DB) *Lag {
	return &Lag{
		db:      db,
		atLevel: map[string]LagReader{},
	}
}

const (
	blip_domain = "repl.lag"
)

func (c *Lag) Domain() string {
	return blip_domain
}

func (c *Lag) Help() blip.CollectorHelp {
	return blip.CollectorHelp{
		Domain:      blip_domain,
		Description: "Replication lag",
		Options: map[string]blip.CollectorHelpOption{
			OPT_WRITER: {
				Name:    OPT_WRITER,
				Desc:    "Type of heartbeat writer",
				Default: "blip",
				Values: map[string]string{
					"blip":         "Native Blip replication lag",
					"pt-heartbeat": "Percona pt-heartbeat",
					"pfs":          "MySQL Performance Schema",
					"legacy":       "Second_Behind_Slave|Replica from SHOW SHOW|REPLICA STATUS",
				},
			},
			OPT_TABLE: {
				Name: OPT_TABLE,
				Desc: "Heartbeat table",
				Values: map[string]string{
					"table": "Blip heartbeat table",
				},
			},
			OPT_SOURCE: {
				Name: OPT_SOURCE,
				Desc: "Source MySQL instance for blip and pt-heartbeat writers",
				Values: map[string]string{
					"monitor-id": "Monitor ID",
				},
			},
		},
	}
}

// Prepares queries for all levels in the plan that contain the "var.global" domain
func (c *Lag) Prepare(ctx context.Context, plan blip.Plan) error {

	// Stop and remove all readers from previous plans, if any
	heartbeat.RemoveReaders(c.db)

LEVEL:
	for _, level := range plan.Levels {
		dom, ok := level.Collect[blip_domain]
		if !ok {
			continue LEVEL // not collected in this level
		}

		table := dom.Options[OPT_TABLE]
		source := dom.Options[OPT_SOURCE]

		switch dom.Options[OPT_WRITER] {
		case "blip":
			r := heartbeat.NewBlipReader(c.db, table, source)
			c.atLevel[level.Name] = r
			go r.Start()
			heartbeat.AddReader(r, c.db, plan.Name, level.Name, dom.Options[OPT_WRITER])
		}
	}

	return nil
}

func (c *Lag) Collect(ctx context.Context, levelName string) ([]blip.MetricValue, error) {
	r := c.atLevel[levelName]
	if r == nil {
		return nil, nil
	}
	lagMs, _, err := r.Lag(ctx)
	if err != nil {
		return nil, err
	}
	m := blip.MetricValue{
		Type:  blip.GAUGE,
		Value: float64(lagMs),
	}
	return []blip.MetricValue{m}, nil
}
