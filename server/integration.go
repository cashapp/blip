package server

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"

	"github.com/square/blip"
	"github.com/square/blip/aws"
	"github.com/square/blip/dbconn"
	"github.com/square/blip/event"
	"github.com/square/blip/monitor"
)

func Defaults() (Plugins, Factories) {
	// Plugins are optional, but factories are required
	awsConfig := aws.NewConfigFactory()
	dbMaker := dbconn.NewConnFactory(awsConfig, nil)
	factories := Factories{
		AWSConfig:  awsConfig,
		DbConn:     dbMaker,
		Monitor:    nil, // deferred, created in server.Boot
		HTTPClient: httpClientFactory{},
	}
	return Plugins{}, factories
}

type Plugins struct {
	InitEventSink    func() event.Sink
	LoadConfig       func(blip.Config) (blip.Config, error)
	LoadLevelPlans   func(blip.Config) ([]blip.Plan, error)
	LoadMonitors     func(blip.Config) ([]blip.ConfigMonitor, error)
	ModifyDB         func(*sql.DB)
	TransformMetrics func(*blip.Metrics) error
}

type Factories struct {
	AWSConfig  aws.ConfigFactory
	DbConn     dbconn.Factory
	Monitor    monitor.Factory
	HTTPClient HTTPClientFactory
}

type HTTPClientFactory interface {
	Make(cfg blip.ConfigHTTP, usedFor string) (*http.Client, error)
}

type httpClientFactory struct{}

func (f httpClientFactory) Make(cfg blip.ConfigHTTP, usedFor string) (*http.Client, error) {
	client := &http.Client{}
	if cfg.Proxy != "" {
		proxyFunc := func(req *http.Request) (url *url.URL, err error) {
			return url.Parse(cfg.Proxy)
		}
		client.Transport = &http.Transport{Proxy: proxyFunc}
	}
	return client, nil
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
