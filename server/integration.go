package server

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"

	"github.com/square/blip"
	"github.com/square/blip/collect"
	"github.com/square/blip/dbconn"
	"github.com/square/blip/event"
	"github.com/square/blip/metrics"
	"github.com/square/blip/sink"
)

func Defaults() (Plugins, Factories) {
	// Plugins are optional
	// Factories are required
	mcMaker := metrics.NewCollectorFactory()
	dbMaker := dbconn.NewConnFactory()
	factories := Factories{
		MakeMetricsCollector: mcMaker,
		MakeDbConn:           dbMaker,
		MakeDbMon:            nil, // deferred, created in server.Boot
		MakeMetricSink:       sink.NewFactory(),
	}
	return Plugins{}, factories
}

type Plugins struct {
	InitEventSink    func() event.Sink
	LoadConfig       func(blip.Config) (blip.Config, error)
	LoadLevelPlans   func(blip.Config) ([]collect.Plan, error)
	LoadMonitors     func(blip.Config) ([]blip.ConfigMonitor, error)
	LoadMetricSinks  func(blip.Config) ([]sink.Sink, error)
	TransformMetrics func(*blip.Metrics) error
}

type Factories struct {
	MakeMetricsCollector metrics.CollectorFactory
	MakeDbConn           dbconn.Factory
	MakeDbMon            DbMonFactory
	MakeMetricSink       sink.Factory
}

func LoadConfig(filePath string, cfg blip.Config) (blip.Config, error) {
	file, err := filepath.Abs(filePath)
	if err != nil {
		return blip.Config{}, err
	}
	blip.Debug("config file: %s (%s)", filePath, file)

	// Config file must exist
	if _, err := os.Stat(file); err != nil {
		if cfg.Strict {
			return blip.Config{}, fmt.Errorf("config file %s does not exist", filePath)
		}
		blip.Debug("config file doesn't exist")
		return cfg, nil
	}

	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		// err includes file name, e.g. "read config file: open <file>: no such file or directory"
		return blip.Config{}, fmt.Errorf("cannot read config file: %s", err)
	}

	if err := yaml.Unmarshal(bytes, &cfg); err != nil {
		return cfg, fmt.Errorf("cannot decode YAML in %s: %s", file, err)
	}

	return cfg, nil
}
