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

	DEFAULT_CONFIG_FILE = "blip.yaml"
	DEFAULT_STRICT      = "" // disabled
	DEFAULT_DEBUG       = "" // disabled
	DEFAULT_DATABASE    = "blip"

	INTERNAL_PLAN_NAME = "blip"
)

var envvar = regexp.MustCompile(`\${([\w_.-]+)(?:(\:\-)([\w_.-]*))?}`)

func interpolateEnv(v string) string {
	if !strings.Contains(v, "${") {
		return v
	}
	m := envvar.FindStringSubmatch(v)
	if len(m) != 4 {
		// @todo error
	}
	v2 := os.Getenv(m[1])
	if v2 == "" && m[2] != "" {
		return m[3]
	}
	return envvar.ReplaceAllString(v, v2)
}

// Config represents the Blip startup configuration.
type Config struct {
	//
	// Blip server
	API           ConfigAPI           `yaml:"api,omitempty"`
	HTTP          ConfigHTTP          `yaml:"http,omitempty"`
	MonitorLoader ConfigMonitorLoader `yaml:"monitor-loader,omitempty"`
	Sinks         ConfigSinks         `yaml:"sinks,omitempty"`
	Strict        bool                `yaml:"strict"`

	//
	// Defaults for monitors
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
		Sinks:         DefaultConfigSinks(),

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

func (c *Config) InterpolateEnvVars() {
	for k, v := range c.Tags {
		c.Tags[k] = interpolateEnv(v)
	}

	c.API.InterpolateEnvVars()
	c.HTTP.InterpolateEnvVars()
	c.Sinks.InterpolateEnvVars()
	c.MonitorLoader.InterpolateEnvVars()

	c.AWS.InterpolateEnvVars()
	c.Exporter.InterpolateEnvVars()
	c.HA.InterpolateEnvVars()
	c.Heartbeat.InterpolateEnvVars()
	c.MySQL.InterpolateEnvVars()
	c.Plans.InterpolateEnvVars()
	c.TLS.InterpolateEnvVars()
}

// ///////////////////////////////////////////////////////////////////////////
// Blip server
// ///////////////////////////////////////////////////////////////////////////

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

func (c *ConfigAPI) InterpolateEnvVars() {
	c.Bind = interpolateEnv(c.Bind)
}

// --------------------------------------------------------------------------

type ConfigHTTP struct {
	Proxy string `yaml:"proxy,omityempty"`
}

func DefaultConfigHTTP() ConfigHTTP {
	return ConfigHTTP{}
}

func (c ConfigHTTP) Validate() error {
	return nil
}

func (c *ConfigHTTP) InterpolateEnvVars() {
	c.Proxy = interpolateEnv(c.Proxy)
}

// --------------------------------------------------------------------------

type ConfigMonitorLoader struct {
	Freq     string                   `yaml:"freq,omitempty"`
	Files    []string                 `yaml:"files,omitempty"`
	StopLoss string                   `yaml:"stop-loss,omitempty"`
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

func (c *ConfigMonitorLoader) InterpolateEnvVars() {
	c.Freq = interpolateEnv(c.Freq)
	c.StopLoss = interpolateEnv(c.StopLoss)
	for i := range c.Files {
		c.Files[i] = interpolateEnv(c.Files[i])
	}
}

// ///////////////////////////////////////////////////////////////////////////
// Monitor
// ///////////////////////////////////////////////////////////////////////////

type ConfigMonitor struct {
	MonitorId      string `yaml:"id"`
	MyCnf          string `yaml:"mycnf,omitempty"`
	Socket         string `yaml:"socket,omitempty"`
	Hostname       string `yaml:"hostname,omitempty"`
	Username       string `yaml:"username,omitempty"`
	Password       string `yaml:"password,omitempty"`
	PasswordFile   string `yaml:"password-file,omitempty"`
	TimeoutConnect string `yaml:"timeout-connect,omitempty"`

	// Tags are passed to each metric sink. Tags inherit from config.tags,
	// but these monitor.tags take precedent (are not overwritten by config.tags).
	Tags map[string]string `yaml:"tags,omitempty"`

	AWS       ConfigAWS              `yaml:"aws,omitempty"`
	Exporter  ConfigExporter         `yaml:"exporter,omitempty"`
	HA        ConfigHighAvailability `yaml:"ha,omitempty"`
	Heartbeat ConfigHeartbeat        `yaml:"heartbeat,omitempty"`
	Plans     ConfigPlans            `yaml:"plans,omitempty"`
	Sinks     ConfigSinks            `yaml:"sinks,omitempty"`
	TLS       ConfigTLS              `yaml:"tls,omitempty"`
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
		Sinks:     DefaultConfigSinks(),
		TLS:       DefaultConfigTLS(),
	}
}

func (c ConfigMonitor) Validate() error {
	return nil
}

func (c *ConfigMonitor) ApplyDefaults(b Config) {
	if c.MyCnf == "" && b.MySQL.MyCnf != "" {
		c.MyCnf = b.MySQL.MyCnf
	}
	if c.Username == "" && b.MySQL.Username != "" {
		c.Username = b.MySQL.Username
	}
	if c.Password == "" && b.MySQL.Password != "" {
		c.Password = b.MySQL.Password
	}
	if c.TimeoutConnect == "" && b.MySQL.TimeoutConnect != "" {
		c.MyCnf = b.MySQL.TimeoutConnect
	}

	for bk, bv := range b.Tags {
		if _, ok := c.Tags[bk]; ok {
			continue
		}
		c.Tags[bk] = bv
	}

	c.AWS.ApplyDefaults(b)
	c.Exporter.ApplyDefaults(b)
	c.HA.ApplyDefaults(b)
	c.Heartbeat.ApplyDefaults(b)
	c.Plans.ApplyDefaults(b)
	c.Sinks.ApplyDefaults(b)
	c.TLS.ApplyDefaults(b)
}

func (c *ConfigMonitor) InterpolateEnvVars() {
	c.MonitorId = interpolateEnv(c.MonitorId)
	c.MyCnf = interpolateEnv(c.MyCnf)
	c.Socket = interpolateEnv(c.Socket)
	c.Hostname = interpolateEnv(c.Hostname)
	c.Username = interpolateEnv(c.Username)
	c.Password = interpolateEnv(c.Password)
	c.PasswordFile = interpolateEnv(c.PasswordFile)
	c.TimeoutConnect = interpolateEnv(c.TimeoutConnect)
	for k, v := range c.Tags {
		c.Tags[k] = interpolateEnv(v)
	}
	c.AWS.InterpolateEnvVars()
	c.Exporter.InterpolateEnvVars()
	c.HA.InterpolateEnvVars()
	c.Heartbeat.InterpolateEnvVars()
	c.Plans.InterpolateEnvVars()
	c.Sinks.InterpolateEnvVars()
	c.TLS.InterpolateEnvVars()
}

func (c *ConfigMonitor) InterpolateMonitor() {
	c.MonitorId = c.interpolateMon(c.MonitorId)
	c.MyCnf = c.interpolateMon(c.MyCnf)
	c.Socket = c.interpolateMon(c.Socket)
	c.Hostname = c.interpolateMon(c.Hostname)
	c.Username = c.interpolateMon(c.Username)
	c.Password = c.interpolateMon(c.Password)
	c.PasswordFile = c.interpolateMon(c.PasswordFile)
	c.TimeoutConnect = c.interpolateMon(c.TimeoutConnect)

	c.AWS.InterpolateMonitor(c)
	c.Exporter.InterpolateMonitor(c)
	c.HA.InterpolateMonitor(c)
	c.Heartbeat.InterpolateMonitor(c)
	c.Plans.InterpolateMonitor(c)
	c.Sinks.InterpolateMonitor(c)
	c.TLS.InterpolateMonitor(c)
}

var monvar = regexp.MustCompile(`%{([\w_-]+)\.([\w_-]+)}`)

func (c *ConfigMonitor) interpolateMon(v string) string {
	if !strings.Contains(v, "%{monitor.") {
		return v
	}
	m := monvar.FindStringSubmatch(v)
	if len(m) != 3 {
		// @todo error
	}
	return monvar.ReplaceAllString(v, c.fieldValue(m[2]))
}

func (c *ConfigMonitor) fieldValue(f string) string {
	switch strings.ToLower(f) {
	case "monitorid", "monitor-id":
		return c.MonitorId
	case "mycnf":
		return c.MyCnf
	case "socket":
		return c.Socket
	case "hostname":
		return c.Hostname
	case "username":
		return c.Username
	case "password":
		return c.Password
	case "passwordfile", "password-file":
		return c.PasswordFile
	case "timeoutconnect", "timeout-connect":
		return c.TimeoutConnect
	default:
		return ""
	}
}

// ///////////////////////////////////////////////////////////////////////////
// Defaults for monitors
// ///////////////////////////////////////////////////////////////////////////

type ConfigAWS struct {
	AuthToken         bool   `yaml:"auth-token"`
	PasswordSecret    string `yaml:"password-secret,omitempty"`
	Region            string `yaml:"region,omitempty"`
	DisableAutoRegion bool   `yaml:"disable-auto-region"`
	DisableAutoTLS    bool   `yaml:"disable-auto-tls"`
}

const (
	DEFAULT_AWS_ROLE = ""
)

func DefaultConfigAWS() ConfigAWS {
	return ConfigAWS{}
}

func (c ConfigAWS) Validate() error {
	return nil
}

func (c *ConfigAWS) ApplyDefaults(b Config) {
	if c.PasswordSecret == "" && b.AWS.PasswordSecret != "" {
		c.PasswordSecret = b.AWS.PasswordSecret
	}
	// @todo
	/*
		if c.AuthToken == "" && b.AWS.AuthToken != "" {
			c.AuthToken = b.AWS.AuthToken
		}
	*/
}

func (c *ConfigAWS) InterpolateEnvVars() {
	c.PasswordSecret = interpolateEnv(c.PasswordSecret)
}

func (c *ConfigAWS) InterpolateMonitor(m *ConfigMonitor) {
	c.PasswordSecret = m.interpolateMon(c.PasswordSecret)
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

func (c *ConfigExporter) ApplyDefaults(b Config) {
}

func (c *ConfigExporter) InterpolateEnvVars() {
}

func (c *ConfigExporter) InterpolateMonitor(m *ConfigMonitor) {
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

func (c *ConfigHeartbeat) ApplyDefaults(b Config) {
}

func (c *ConfigHeartbeat) InterpolateEnvVars() {
}

func (c *ConfigHeartbeat) InterpolateMonitor(m *ConfigMonitor) {
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

func (c *ConfigHighAvailability) ApplyDefaults(b Config) {
}

func (c *ConfigHighAvailability) InterpolateEnvVars() {
	c.Freq = interpolateEnv(c.Freq)
	c.Role = interpolateEnv(c.Role)
	c.Table = interpolateEnv(c.Table)
	c.CreateTable = interpolateEnv(c.CreateTable)
}

func (c *ConfigHighAvailability) InterpolateMonitor(m *ConfigMonitor) {
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

func (c *ConfigMySQL) ApplyDefaults(b Config) {
}

func (c *ConfigMySQL) InterpolateEnvVars() {
}

func (c *ConfigMySQL) InterpolateMonitor(m *ConfigMonitor) {
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

func (c *ConfigPlans) ApplyDefaults(b Config) {
}

func (c *ConfigPlans) InterpolateEnvVars() {
	for i := range c.Files {
		c.Files[i] = interpolateEnv(c.Files[i])
	}

}

func (c *ConfigPlans) InterpolateMonitor(m *ConfigMonitor) {
}

// --------------------------------------------------------------------------

type ConfigSinks map[string]map[string]string

func DefaultConfigSinks() ConfigSinks {
	return ConfigSinks{}
}

func (c ConfigSinks) Validate() error {
	return nil
}

func (c ConfigSinks) ApplyDefaults(b Config) {
	for bk, bv := range b.Sinks {
		if _, ok := c[bk]; ok {
			continue
		}
		c[bk] = bv
	}
}

func (c ConfigSinks) InterpolateEnvVars() {
	for _, opts := range c {
		for k, v := range opts {
			opts[k] = interpolateEnv(v)
		}
	}
}

func (c ConfigSinks) InterpolateMonitor(m *ConfigMonitor) {
	for _, opts := range c {
		for k, v := range opts {
			opts[k] = m.interpolateMon(v)
		}
	}
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

func (c *ConfigTLS) ApplyDefaults(b Config) {
}

func (c *ConfigTLS) InterpolateEnvVars() {
	c.Cert = interpolateEnv(c.Cert)
	c.Key = interpolateEnv(c.Key)
	c.CA = interpolateEnv(c.CA)
}

func (c *ConfigTLS) InterpolateMonitor(m *ConfigMonitor) {
	c.Cert = m.interpolateMon(c.Cert)
	c.Key = m.interpolateMon(c.Key)
	c.CA = m.interpolateMon(c.CA)
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
