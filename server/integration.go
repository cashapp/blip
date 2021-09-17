package server

import (
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
)

func Defaults() (blip.Plugins, blip.Factories) {
	// Plugins are optional, but factories are required
	awsConfig := aws.NewConfigFactory()
	dbMaker := dbconn.NewConnFactory(awsConfig, nil)
	factories := blip.Factories{
		AWSConfig:  awsConfig,
		DbConn:     dbMaker,
		HTTPClient: httpClientFactory{},
	}
	return blip.Plugins{}, factories
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
