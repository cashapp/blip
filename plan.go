// Copyright 2022 Block, Inc.

package blip

import (
	"fmt"
	"regexp"
	"time"
)

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
	// When a monitor (M) loads plans from a table (config.monitors.plans.table),
	// the table is filtered: WHERE monitorId = config.monitors.M.id.
	MonitorId string `yaml:"-"`

	// Source of plan: file name, table name, "plugin", or "blip" (internal plans).
	Source string `yaml:"-"`
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
	Metrics []string          `yaml:"metrics,omitempty"`
	Options map[string]string `yaml:"options,omitempty"`
	Errors  map[string]string `yaml:"errors,omitempty"`
}

const metricPattern = `^[a-zA-Z0-9_-]*$`

var validMetricRegex = regexp.MustCompile(metricPattern)

func (p Plan) Validate() error {
	freqs := map[time.Duration]string{}

	for levelName := range p.Levels {

		// Validate freq: set, valid, and no duplicates
		freq := p.Levels[levelName].Freq
		if freq == "" {
			return fmt.Errorf("at %s: freq not set (Go time duration string required)", levelName)
		}
		d, err := time.ParseDuration(freq)
		if err != nil {
			return fmt.Errorf("at %s: invalid freq: %s: %s", levelName, freq, err)
		}
		if firstLevelName, ok := freqs[d]; ok {
			return fmt.Errorf("at %s: duplicate freq: %s (%s): first seen at %s", levelName, freq, d, firstLevelName)
		}
		freqs[d] = levelName

		// Validate that every metric matches metricPattern (help prevent SQL injection)
		for domainName := range p.Levels[levelName].Collect {
			for _, metricName := range p.Levels[levelName].Collect[domainName].Metrics {
				if !validMetricRegex.MatchString(metricName) {
					return fmt.Errorf("at %s/%s: invalid metric: %s (does not match /%s/)",
						levelName, domainName, metricName, metricPattern)
				}
			}
		}
	}

	return nil
}

func (p *Plan) InterpolateEnvVars() {
	for levelName := range p.Levels {
		for domainName := range p.Levels[levelName].Collect {
			for k, v := range p.Levels[levelName].Collect[domainName].Options {
				p.Levels[levelName].Collect[domainName].Options[k] = interpolateEnv(v)
			}
		}
	}
}

func (p *Plan) InterpolateMonitor(mon *ConfigMonitor) {
	for levelName := range p.Levels {
		for domainName := range p.Levels[levelName].Collect {
			for k, v := range p.Levels[levelName].Collect[domainName].Options {
				p.Levels[levelName].Collect[domainName].Options[k] = mon.interpolateMon(v)
			}
		}
	}
}
