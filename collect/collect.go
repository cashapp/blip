package collect

import "database/sql"

// Plan represents different levels of metrics collection.
type Plan struct {
	// Name is the name of the plan (required).
	//
	// When loaded from config.plans.files, Name is the exact name of the config.
	// The first file is the default plan if config.plan.default is not specified.
	//
	// When loaded from a config.plans.table, Name is the name column. The name
	// column cannot be NULL. The plan table is ordered by name (ascending) and
	// the first plan is the default if config.plan.default is not specified.
	//
	// config.plan.adjust.readonly and .active refer to Name.
	Name string

	// Levels are the collection frequencies that constitue the plan (required).
	Levels map[string]Level

	// MonitorId is the optional monitorId column from a plan table.
	//
	// When default plans are loaded from a table (config.plans.table),
	// the talbe is not filtered; all plans in the table are loaded.
	//
	// When a monitor (M) loads plans from a table (config.monitors.M.plans.table),
	// the table is filtered: WHERE monitorId = config.monitors.M.id.
	MonitorId string `yaml:"-"`

	// First is true for the first plan loaded from any source. PlanLoader uses
	// this to return the first plan when there are multiple plans but no LPA to
	// set plans based on state.
	firstRow  bool
	firstFile bool
	internal  bool
}

// Level is one collection frequency in a plan.
type Level struct {
	Name    string            `yaml:"-"`
	Freq    string            `yaml:"freq"`
	Collect map[string]Domain `yaml:"collect"`
}

// Domain is one metric domain for collecting related metrics.
type Domain struct {
	Name    string            `yaml:"-"`
	Options map[string]string `yaml:"options,omitempty"`
	Metrics []string          `yaml:"metrics,omitempty"`
}

// Metrics are raw metrics from one collector.
type Metrics struct {
	Values map[string]float64
}

// Help represents information about a collector.
type Help struct {
	Domain      string
	Description string
	Options     [][]string // { {"key", "Description of key", "default;val1;val2}, ... }
}

type Args struct {
	MonitorId string
	DB        *sql.DB
}
