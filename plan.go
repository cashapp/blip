package blip

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

// Help represents information about a collector.
type CollectorHelp struct {
	Domain      string
	Description string
	Options     map[string]CollectorHelpOption
}

type CollectorHelpOption struct {
	Name    string
	Desc    string            // describes Name
	Default string            // key in Values
	Values  map[string]string // value => description
}

// --------------------------------------------------------------------------

func InternalLevelPlan() Plan {
	return Plan{
		Name: "blip",
		Levels: map[string]Level{
			"key-performance-indicators": Level{
				Name: "key-performance-indicators",
				Freq: "1s",
				Collect: map[string]Domain{
					"var.global": {
						Name: "var.global",
						Metrics: []string{
							"read_only",
						},
					},
				},
			},
			"sysvars": Level{
				Name: "sysvars",
				Freq: "5s",
				Collect: map[string]Domain{
					"var.global": {
						Name: "var.global",
						Metrics: []string{
							"max_connections",
						},
					},
				},
			},
		},
	}
}

func PromPlan() Plan {
	return Plan{
		Name: "mysqld_exporter",
		Levels: map[string]Level{
			"all": Level{
				Name: "all",
				Freq: "", // none, pulled/scaped on demand
				Collect: map[string]Domain{
					"status.global": {
						Name: "status.global",
						Options: map[string]string{
							"all": "yes",
						},
					},
					"var.global": {
						Name: "var.global",
						Options: map[string]string{
							"all": "yes",
						},
					},
					"innodb": {
						Name: "innodb",
						Options: map[string]string{
							"all": "enabled",
						},
					},
				},
			},
		},
	}
}
