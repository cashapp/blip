package sysvar

import (
	"context"

	"github.com/square/blip/collect"
	"github.com/square/blip/db"
)

const (
	OPT_SOURCE = "source"
)

// Global collects global system variables for the var.global domain.
type Global struct {
	in    db.Instance
	plans collect.Plan
	// --
	domain string
	workIn map[string][]string
}

func NewGlobal(in db.Instance) *Global {
	return &Global{
		in: in,
		// --
		domain: "var.global",
		workIn: map[string][]string{},
	}
}

func (c *Global) Domain() string {
	return c.domain
}

func (c *Global) Help() collect.Help {
	return collect.Help{
		Domain:      c.domain,
		Description: "Collect global status variables (sysvars)",
		Options: [][]string{
			{
				OPT_SOURCE,
				"Where to collect sysvars from",
				"auto (auto-determine best source); pfs (performance_schema.global_variables); show (SHOW GLOBAL STATUS)",
			},
		},
	}
}

func (c *Global) Prepare(plan collect.Plan) error {
LEVEL:
	for levelName, level := range plan.Levels {
		dom, ok := level.Collect[c.domain]
		if !ok {
			// This domain not collected in this level
			continue LEVEL
		}

		// Handle options, if any
		if dom.Options != nil {
			if src, _ := dom.Options[OPT_SOURCE]; ok {
				switch src {
				case "auto", "":
				case "pfs":
				case "show":
				default:
					// @todo: error, invalid option
				}
			}
		}

		// Save metrics to collect for this level
		c.workIn[levelName] = make([]string, len(dom.Metrics))
		copy(c.workIn[levelName], dom.Metrics)

		// Prepare satements for collecting metrics?
	}

	return nil
}

func (c *Global) Collect(ctx context.Context, levelName string) (collect.Metrics, error) {
	//dom := level[c.domain]

	rows, err := c.in.DB().QueryContext(ctx, "SHOW GLOBAL VARIABLES")
	if err != nil {
		return collect.Metrics{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var name, val string
		if err := rows.Scan(&name, &val); err != nil {
			// @todo
		}
	}

	return collect.Metrics{}, nil
}
