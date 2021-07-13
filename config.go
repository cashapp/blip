package blip

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
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
	API           ConfigAPI                    `yaml:"api"`
	MonitorLoader ConfigMonitorLoader          `yaml:"monitor-loader"`
	Sinks         map[string]map[string]string `yaml:"sinks"`
	Strict        bool                         `yaml:"strict"`

	//
	// Monitor default configs
	//
	AWS       ConfigAWS              `yaml:"aws"`
	Exporter  ConfigExporter         `yaml:"exporter"`
	HA        ConfigHighAvailability `yaml:"ha"`
	Heartbeat ConfigHeartbeat        `yaml:"heartbeat"`
	MySQL     ConfigMySQL            `yaml:"mysql"`
	Plans     ConfigPlans            `yaml:"plans"`
	TLS       ConfigTLS              `yaml:"tls"`

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

// --------------------------------------------------------------------------

type ConfigMonitor struct {
	Id       string `yaml:"id"`
	Hostname string `yaml:"hostname"`
	Socket   string `yaml:"socket"`
	Auth     string `yaml:"auth"` // file, aws-iam, aws-secrets-manaager
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	MyCnf    string `yaml:"mycnf"`

	Tags           map[string]string `yaml:"tags"`
	TimeoutConnect string            `yaml:"timeoutConnect"`

	AWS       ConfigAWS              `yaml:"aws"`
	Exporter  ConfigExporter         `yaml:"exporter"`
	HA        ConfigHighAvailability `yaml:"ha"`
	Heartbeat ConfigHeartbeat        `yaml:"heartbeat"`
	MySQL     ConfigMySQL            `yaml:"mysql"`
	Plans     ConfigPlans            `yaml:"plans"`
	TLS       ConfigTLS              `yaml:"tls"`
}

const (
	DEFAULT_MONITOR_ID              = "localhost"
	DEFAULT_MONITOR_HOSTNAME        = "127.0.0.1:3306"
	DEFAULT_MONITOR_USERNAME        = "blip"
	DEFAULT_MONITOR_PASSWORD        = "blip"
	DEFAULT_MONITOR_TIMEOUT_CONNECT = "5s"
)

func DefaultConfigMonitor() ConfigMonitor {
	return ConfigMonitor{
		Id:       DEFAULT_MONITOR_ID,
		Hostname: DEFAULT_MONITOR_HOSTNAME,
		Username: DEFAULT_MONITOR_USERNAME,
		Password: DEFAULT_MONITOR_PASSWORD,

		Tags:           map[string]string{},
		TimeoutConnect: DEFAULT_MONITOR_TIMEOUT_CONNECT,

		AWS:       DefaultConfigAWS(),
		Exporter:  DefaultConfigExporter(),
		HA:        DefaultConfigHA(),
		Heartbeat: DefaultConfigHeartbeat(),
		MySQL:     DefaultConfigMySQL(),
		Plans:     DefaultConfigPlans(),
		TLS:       DefaultConfigTLS(),
	}
}

func (c ConfigMonitor) Validate() error {
	return nil
}

// --------------------------------------------------------------------------

type ConfigMonitorLoader struct {
	Freq     string   `yaml:"freq"`
	StopLoss float64  `yaml:"stop-loss"`
	Files    []string `yaml:"files"`
	AWS      ConfigMonitorLoaderAWS
	Local    ConfigMonitorLoaderLocal
}

type ConfigMonitorLoaderAWS struct {
	Auto bool `yaml:"auto"`
}

type ConfigMonitorLoaderLocal struct {
	Auto bool `yaml:"auto"`
}

const (
	DEFAULT_MONITOR_LOADER_LOCAL_AUTO = true
)

func DefaultConfigMonitorLoader() ConfigMonitorLoader {
	return ConfigMonitorLoader{
		Local: ConfigMonitorLoaderLocal{
			Auto: DEFAULT_MONITOR_LOADER_LOCAL_AUTO,
		},
	}
}

func (c ConfigMonitorLoader) Validate() error {
	return nil
}

// --------------------------------------------------------------------------

type ConfigPlans struct {
	Files   []string           `yaml:"files"`
	Table   string             `yaml:"table"`
	Monitor *ConfigMonitor     `yaml:"monitor"`
	Adjust  ConfigPlanAdjuster `yaml:"adjust"`
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

func (c ConfigPlans) MultiState() bool {
	return c.Adjust.Freq != "" && c.Adjust.ReadOnly != ""
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

// --------------------------------------------------------------------------

type ConfigAWS struct {
	Role string `yaml:"role"`
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

// --------------------------------------------------------------------------

type ConfigExporter struct {
	Bind string `yaml:"bind"`
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
	return ConfigHighAvailability{
		Freq:        DEFAULT_HA_FREQ,
		Role:        DEFAULT_HA_ROLE,
		Table:       DEFAULT_HA_TABLE,
		CreateTable: DEFAULT_HA_CREATE_TABLE,
	}
}

func (c ConfigHighAvailability) Validate() error {
	return nil
}

// --------------------------------------------------------------------------

type ConfigMySQL struct {
	AutoConfig string `yaml:"autoConfig"`
}

const (
	DEFALUT_MYSQL_AUTOCONFIG = "try"
)

func DefaultConfigMySQL() ConfigMySQL {
	return ConfigMySQL{
		AutoConfig: DEFALUT_MYSQL_AUTOCONFIG,
	}
}

func (c ConfigMySQL) Validate() error {
	return nil
}

// --------------------------------------------------------------------------

type ConfigTLS struct {
	Cert string `yaml:"cert"`
	Key  string `yaml:"key"`
	CA   string `yaml:"ca"`
}

func DefaultConfigTLS() ConfigTLS {
	return ConfigTLS{}
}

func (c ConfigTLS) Validate() error {
	return nil
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
