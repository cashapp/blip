package blip

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
)

const (
	ENV_STRICT = "BLIP_STRICT"
	ENV_DEBUG  = "BLIP_DEBUG"
)

const (
	DEFAULT_CONFIG_FILE = "blip.yaml"
	DEFAULT_STRICT      = "" // disabled
	DEFAULT_DEBUG       = "" // disabled
	DEFAULT_DATABASE    = "blip"
)

const INTERNAL_PLAN_NAME = "blip"

// Config represents the Blip startup configuration.
type Config struct {
	//
	// Server configs
	API           ConfigAPI                    `yaml:"api,omitempty"`
	MonitorLoader ConfigMonitorLoader          `yaml:"monitor-loader,omitempty"`
	Sinks         map[string]map[string]string `yaml:"sinks,omitempty"`
	Strict        bool                         `yaml:"strict"`

	//
	// Monitor default configs
	//
	AWS       ConfigAWS              `yaml:"aws,omitempty"`
	Exporter  ConfigExporter         `yaml:"exporter,omitempty"`
	HA        ConfigHighAvailability `yaml:"ha,omitempty"`
	Heartbeat ConfigHeartbeat        `yaml:"heartbeat,omitempty"`
	MySQL     ConfigMySQL            `yaml:"mysql,omitempty"`
	Plans     ConfigPlans            `yaml:"plans,omitempty"`
	Tags      map[string]string      `yaml:"tags,omitempty"`
	TLS       ConfigTLS              `yaml:"tls,omitempty"`

	Monitors []ConfigMonitor `yaml:"monitors,omitempty"`
}

func DefaultConfig(strict bool) Config {
	if strict {
		return Config{
			Strict:   strict,
			API:      DefaultConfigAPI(),
			Monitors: []ConfigMonitor{},
		}
	}

	return Config{
		API:           DefaultConfigAPI(),
		MonitorLoader: DefaultConfigMonitorLoader(),
		Sinks:         map[string]map[string]string{},

		AWS:       DefaultConfigAWS(),
		Exporter:  DefaultConfigExporter(),
		HA:        DefaultConfigHA(),
		Heartbeat: DefaultConfigHeartbeat(),
		MySQL:     DefaultConfigMySQL(),
		Plans:     DefaultConfigPlans(),
		TLS:       DefaultConfigTLS(),

		// Default config does not have any monitors. If a real config file
		// does not specify any, Server.LoadMonitors() will attemp to
		// auto-detect a local MySQL instance, starting with DefaultConfigMontor().
		Monitors: []ConfigMonitor{},
	}
}

func (c Config) Validate() error {
	return nil
}

func (c *Config) Interpolate() {
	for k, v := range c.Tags {
		c.Tags[k] = interpolateEnv(v)
	}

	for sink, kv := range c.Sinks {
		for k, v := range kv {
			kv[k] = interpolateEnv(v)
		}
		c.Sinks[sink] = kv
	}

	c.API.Interpolate()
	c.MonitorLoader.Interpolate()

	c.AWS.Interpolate()
	c.Exporter.Interpolate()
	c.HA.Interpolate()
	c.Heartbeat.Interpolate()
	c.MySQL.Interpolate()
	c.Plans.Interpolate()
	c.TLS.Interpolate()
}

// --------------------------------------------------------------------------

type ConfigMySQL struct {
	MyCnf          string `yaml:"mycnf,omitempty"`
	Username       string `yaml:"username,omitempty"`
	Password       string `yaml:"password,omitempty"`
	TimeoutConnect string `yaml:"timeoutConnect"`
}

const (
	DEFAULT_MONITOR_USERNAME        = "blip"
	DEFAULT_MONITOR_TIMEOUT_CONNECT = "5s"
)

func DefaultConfigMySQL() ConfigMySQL {
	return ConfigMySQL{
		Username:       DEFAULT_MONITOR_USERNAME,
		TimeoutConnect: DEFAULT_MONITOR_TIMEOUT_CONNECT,
	}
}

func (c ConfigMySQL) Validate() error {
	return nil
}

func (c *ConfigMySQL) Interpolate() {
}

// --------------------------------------------------------------------------

type ConfigMonitor struct {
	Id             string `yaml:"id"`
	MyCnf          string `yaml:"mycnf"`
	Socket         string `yaml:"socket"`
	Hostname       string `yaml:"hostname"`
	Username       string `yaml:"username"`
	Password       string `yaml:"password"`
	PasswordFile   string `yaml:"password-file"`
	TimeoutConnect string `yaml:"timeoutConnect"`

	Tags map[string]string `yaml:"tags"`

	AWS       ConfigAWS              `yaml:"aws"`
	Exporter  ConfigExporter         `yaml:"exporter,omitempty"`
	HA        ConfigHighAvailability `yaml:"ha"`
	Heartbeat ConfigHeartbeat        `yaml:"heartbeat"`
	Plans     ConfigPlans            `yaml:"plans"`
	TLS       ConfigTLS              `yaml:"tls"`
}

func DefaultConfigMonitor() ConfigMonitor {
	return ConfigMonitor{
		Username:       DEFAULT_MONITOR_USERNAME,
		TimeoutConnect: DEFAULT_MONITOR_TIMEOUT_CONNECT,

		Tags: map[string]string{},

		AWS:       DefaultConfigAWS(),
		Exporter:  DefaultConfigExporter(),
		HA:        DefaultConfigHA(),
		Heartbeat: DefaultConfigHeartbeat(),
		Plans:     DefaultConfigPlans(),
		TLS:       DefaultConfigTLS(),
	}
}

func (mon ConfigMonitor) WithDefaults(cfg Config) ConfigMonitor {
	if cfg.MySQL.Username != "" {
		mon.Username = cfg.MySQL.Username
	}
	if cfg.MySQL.Password != "" {
		mon.Password = cfg.MySQL.Password
	}
	if cfg.MySQL.TimeoutConnect != "" {
		mon.TimeoutConnect = cfg.MySQL.TimeoutConnect
	}

	for k, v := range cfg.Tags {
		mon.Tags[k] = v
	}

	return mon
}

func (c ConfigMonitor) Validate() error {
	return nil
}

func (c *ConfigMonitor) Interpolate() {
}

// --------------------------------------------------------------------------

type ConfigMonitorLoader struct {
	Freq     string                   `yaml:"freq,omitempty"`
	StopLoss string                   `yaml:"stop-loss,omitempty"`
	Files    []string                 `yaml:"files,omitempty"`
	AWS      ConfigMonitorLoaderAWS   `yaml:"aws,omitempty"`
	Local    ConfigMonitorLoaderLocal `yaml:"local,omitempty"`
}

type ConfigMonitorLoaderAWS struct {
	DisableAuto bool `yaml:"disable-auto"`
}

type ConfigMonitorLoaderLocal struct {
	DisableAuto     bool `yaml:"disable-auto"`
	DisableAutoRoot bool `yaml:"disable-auto-root"`
}

func DefaultConfigMonitorLoader() ConfigMonitorLoader {
	return ConfigMonitorLoader{}
}

func (c ConfigMonitorLoader) Validate() error {
	// StopLoss match N%
	return nil
}

func (c *ConfigMonitorLoader) Interpolate() {
}

// --------------------------------------------------------------------------

type ConfigPlans struct {
	Files   []string           `yaml:"files,omitempty"`
	Table   string             `yaml:"table,omitempty"`
	Monitor *ConfigMonitor     `yaml:"monitor,omitempty"`
	Adjust  ConfigPlanAdjuster `yaml:"adjust,omitempty"`
}

type ConfigPlanAdjuster struct {
	Freq     string `yaml:"freq"`
	ReadOnly string `yaml:"readonlyPlan"`
	Active   string `yaml:"activePlan"`
}

const (
	DEFAULT_PLANS_FILES = "plan.yaml"
	DEFAULT_PLANS_TABLE = "blip.plans"
)

func DefaultConfigPlans() ConfigPlans {
	return ConfigPlans{
		Table: DEFAULT_PLANS_TABLE,
		Files: []string{DEFAULT_PLANS_FILES},
	}
}

func (c ConfigPlans) Validate() error {
	return nil
}

func (c *ConfigPlans) Interpolate() {
}

// --------------------------------------------------------------------------

type ConfigAPI struct {
	Bind    string `yaml:"bind"`
	Disable bool   `yaml:"disable"`
}

const (
	DEFAULT_API_BIND = "127.0.0.1:9070"
)

func DefaultConfigAPI() ConfigAPI {
	return ConfigAPI{
		Bind: DEFAULT_API_BIND,
	}
}

func (c ConfigAPI) Validate() error {
	return nil
}

func (c *ConfigAPI) Interpolate() {
}

// --------------------------------------------------------------------------

type ConfigAWS struct {
	PasswordSecret string `yaml:"password-secret"`
	AuthToken      bool   `yaml:"auth-token"`
	Role           string `yaml:"role"`
}

const (
	DEFAULT_AWS_ROLE = ""
)

func DefaultConfigAWS() ConfigAWS {
	return ConfigAWS{
		Role: DEFAULT_AWS_ROLE,
	}
}

func (c ConfigAWS) Validate() error {
	return nil
}

func (c *ConfigAWS) Interpolate() {
}

// --------------------------------------------------------------------------

type ConfigExporter struct {
	Bind string `yaml:"bind,omitempty"`
}

const (
	DEFAULT_EXPORTER_BIND = "" // disabled
)

func DefaultConfigExporter() ConfigExporter {
	return ConfigExporter{
		Bind: DEFAULT_EXPORTER_BIND,
	}
}

func (c ConfigExporter) Validate() error {
	return nil
}

func (c *ConfigExporter) Interpolate() {
}

// --------------------------------------------------------------------------

type ConfigHeartbeat struct {
	Freq        string `yaml:"freq"`
	Table       string `yaml:"table"`
	CreateTable string `yaml:"createTable"`
	Optional    bool   `yaml:"optional"`
}

const (
	DEFAULT_HEARTBEAT_FREQ         = "500ms"
	DEFAULT_HEARTBEAT_TABLE        = "blip.heartbeat"
	DEFAULT_HEARTBEAT_CREATE_TABLE = "try"
	DEFAULT_HEARTBEAT_OPTIONAL     = true
)

func DefaultConfigHeartbeat() ConfigHeartbeat {
	return ConfigHeartbeat{
		Freq:        DEFAULT_HEARTBEAT_FREQ,
		Table:       DEFAULT_HEARTBEAT_TABLE,
		CreateTable: DEFAULT_HEARTBEAT_CREATE_TABLE,
		Optional:    DEFAULT_HEARTBEAT_OPTIONAL,
	}
}

func (c ConfigHeartbeat) Validate() error {
	return nil
}

func (c *ConfigHeartbeat) Interpolate() {
}

// --------------------------------------------------------------------------

type ConfigHighAvailability struct {
	Freq        string `yaml:"freq"`
	Role        string `yaml:"role"`
	Table       string `yaml:"table"`
	CreateTable string `yaml:"createTable"`
	Optional    bool   `yaml:"optional"`
}

const (
	DEFAULT_HA_FREQ         = "" // disabled
	DEFAULT_HA_ROLE         = "primary"
	DEFAULT_HA_TABLE        = "blip.ha"
	DEFAULT_HA_CREATE_TABLE = "auto"
)

func DefaultConfigHA() ConfigHighAvailability {
	return ConfigHighAvailability{}
}

func (c ConfigHighAvailability) Validate() error {
	return nil
}

func (c *ConfigHighAvailability) Interpolate() {
	c.Freq = interpolateEnv(c.Freq)
	c.Role = interpolateEnv(c.Role)
	c.Table = interpolateEnv(c.Table)
	c.CreateTable = interpolateEnv(c.CreateTable)
}

// --------------------------------------------------------------------------

type ConfigTLS struct {
	Cert string `yaml:"cert,omitempty"`
	Key  string `yaml:"key,omitempty"`
	CA   string `yaml:"ca,omitempty"`
}

func DefaultConfigTLS() ConfigTLS {
	return ConfigTLS{}
}

func (c ConfigTLS) Validate() error {
	return nil
}

func (c *ConfigTLS) Interpolate() {
	c.Cert = interpolateEnv(c.Cert)
	c.Key = interpolateEnv(c.Key)
	c.CA = interpolateEnv(c.CA)
}

func (c ConfigTLS) LoadTLS() (*tls.Config, error) {
	if c.Cert == "" && c.Key == "" && c.CA == "" {
		return nil, nil
	}

	tlsConfig := &tls.Config{}

	// Root CA (optional)
	if c.CA != "" {
		caCert, err := ioutil.ReadFile(c.CA)
		if err != nil {
			return nil, err
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		tlsConfig.RootCAs = caCertPool
	}

	// Cert and key
	if c.Cert != "" && c.Key != "" {
		cert, err := tls.LoadX509KeyPair(c.Cert, c.Key)
		if err != nil {
			return nil, err
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
		tlsConfig.BuildNameToCertificate()
	} else {
		return nil, fmt.Errorf("TLS cert or key not specififed; both are required")
	}

	return tlsConfig, nil
}

// ---------------------------------------------------------------------------

var envvar = regexp.MustCompile(`^\${([\w_.-]+)(?:(\:\-)([\w_.-]*))?}`)

func interpolateEnv(v string) string {
	if !strings.HasPrefix(v, "${") {
		return v
	}
	m := envvar.FindStringSubmatch(v)
	if len(m) != 2 {
		// @todo error
	}
	v2 := os.Getenv(m[1])
	if v2 == "" && m[2] != "" {
		return m[3]
	}
	return v2
}
