package status

import (
	"context"
	"database/sql"

	"github.com/square/blip"
	"github.com/square/blip/collect"
)

// Global collects global system variables for the var.global domain.
type Global struct {
	monitorId string
	db        *sql.DB
	plans     collect.Plan
	domain    string
	workIn    map[string][]string
	queryIn   map[string]string
	sourceIn  map[string]string
}

func NewGlobal(monitor blip.Monitor) *Global {
	return &Global{
		monitorId: monitor.MonitorId(),
		db:        monitor.DB(),
		domain:    "status.global",
		workIn:    map[string][]string{},
		queryIn:   make(map[string]string),
		sourceIn:  make(map[string]string),
	}
}

func (c *Global) Domain() string {
	return c.domain
}

func (c *Global) Help() collect.Help {
	return collect.Help{
		Domain:      c.domain,
		Description: "Collect global status",
		Options:     [][]string{},
	}
}

// Prepares queries for all levels in the plan that contain the "var.global" domain
func (c *Global) Prepare(plan collect.Plan) error {
	return nil
}

func (c *Global) Collect(ctx context.Context, levelName string) (collect.Metrics, error) {
	return collect.Metrics{}, nil
}
