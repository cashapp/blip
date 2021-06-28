package blip

import (
	"time"

	"github.com/square/blip/collect"
	"github.com/square/blip/db"
)

const VERSION = "0.0.0"

var SHA = ""

// Config represents the Blip startup configuration.
type Config struct {
}

func DefaultConfig() Config {
	return Config{
		// Default values
	}
}

// Metrics are metrics collected for one plan level, from one database instance.
type Metrics struct {
	Ts     time.Time
	Plan   string
	Level  string
	DbId   string
	State  string
	Values map[string]float64
}

type Plugins struct {
	LoadConfig       func(Config) (Config, error)
	LoadLevelPlan    func(Config) (collect.Plan, error)
	LoadDbInstances  func(Config) ([]db.Instance, error)
	TransformMetrics func(*Metrics) error
}
