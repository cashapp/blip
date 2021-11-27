package blip

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v2"
)

const (
	ENV_STRICT = "BLIP_STRICT"
	ENV_DEBUG  = "BLIP_DEBUG"

	DEFAULT_CONFIG_FILE = "blip.yaml"
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

func setBool(c *bool, b *bool) *bool {
	if c == nil && b != nil {
		c = new(bool)
		*c = *b
	}
	return c
}

func LoadConfig(filePath string, cfg Config) (Config, error) {
	file, err := filepath.Abs(filePath)
	if err != nil {
		return Config{}, err
	}
	Debug("config file: %s (%s)", filePath, file)

	// Config file must exist
	if _, err := os.Stat(file); err != nil {
		if cfg.Strict {
			return Config{}, fmt.Errorf("config file %s does not exist", filePath)
		}
		Debug("config file doesn't exist")
		return cfg, nil
	}

	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		// err includes file name, e.g. "read config file: open <file>: no such file or directory"
		return Config{}, fmt.Errorf("cannot read config file: %s", err)
	}

	if err := yaml.Unmarshal(bytes, &cfg); err != nil {
		return cfg, fmt.Errorf("cannot decode YAML in %s: %s", file, err)
	}

	return cfg, nil
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
	Disable bool   `yaml:"disable,omitempty"`
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
	MonitorId string `yaml:"id"`
	Socket    string `yaml:"socket,omitempty"`
	Hostname  string `yaml:"hostname,omitempty"`
	// ConfigMySQL:
	MyCnf          string `yaml:"mycnf,omitempty"`
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

	if c.Sinks == nil {
		c.Sinks = ConfigSinks{}
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

// --------------------------------------------------------------------------

type ConfigAWS struct {
	AuthToken         *bool  `yaml:"auth-token,omitempty"`
	PasswordSecret    string `yaml:"password-secret,omitempty"`
	Region            string `yaml:"region,omitempty"`
	DisableAutoRegion *bool  `yaml:"disable-auto-region,omitempty"`
	DisableAutoTLS    *bool  `yaml:"disable-auto-tls,omitempty"`
}

func DefaultConfigAWS() ConfigAWS {
	return ConfigAWS{}
}

func (c ConfigAWS) Validate() error {
	return nil
}

func (c *ConfigAWS) ApplyDefaults(b Config) {
	if c.PasswordSecret == "" {
		c.PasswordSecret = b.AWS.PasswordSecret
	}
	if c.Region == "" {
		c.Region = b.AWS.Region
	}

	c.AuthToken = setBool(c.AuthToken, b.AWS.AuthToken)
	c.DisableAutoRegion = setBool(c.DisableAutoRegion, b.AWS.DisableAutoRegion)
	c.DisableAutoTLS = setBool(c.DisableAutoTLS, b.AWS.DisableAutoTLS)

}

func (c *ConfigAWS) InterpolateEnvVars() {
	c.PasswordSecret = interpolateEnv(c.PasswordSecret)
	c.Region = interpolateEnv(c.Region)
}

func (c *ConfigAWS) InterpolateMonitor(m *ConfigMonitor) {
	c.PasswordSecret = m.interpolateMon(c.PasswordSecret)
	c.Region = m.interpolateMon(c.Region)
}

// --------------------------------------------------------------------------

const (
	EXPORTER_MODE_DUAL   = "dual"   // Blip and exporter run together
	EXPORTER_MODE_LEGACY = "legacy" // only exporter runs

	DEFAULT_EXPORTER_LISTEN_ADDR = "127.0.0.1:9104"
	DEFAULT_EXPORTER_PATH        = "/metrics"
)

type ConfigExporter struct {
	Flags map[string]string `yaml:"flags,omitempty"`
	Mode  string            `yaml:"mode,omitempty"`
}

func DefaultConfigExporter() ConfigExporter {
	return ConfigExporter{}
}

func (c ConfigExporter) Validate() error {
	if c.Mode != "" && (c.Mode != EXPORTER_MODE_DUAL && c.Mode != EXPORTER_MODE_LEGACY) {
		return fmt.Errorf("invalid mode: %s; valid values: dual, legacy", c.Mode)
	}
	return nil
}

func (c *ConfigExporter) ApplyDefaults(b Config) {
	if c.Mode == "" && b.Exporter.Mode != "" {
		c.Mode = b.Exporter.Mode
	}
	if len(b.Exporter.Flags) > 0 {
		if c.Flags == nil {
			c.Flags = map[string]string{}
		}
		for k, v := range b.Exporter.Flags {
			c.Flags[k] = v
		}
	}
}

func (c *ConfigExporter) InterpolateEnvVars() {
	interpolateEnv(c.Mode)
	for k := range c.Flags {
		c.Flags[k] = interpolateEnv(c.Flags[k])
	}
}

func (c *ConfigExporter) InterpolateMonitor(m *ConfigMonitor) {
	m.interpolateMon(c.Mode)
	for k := range c.Flags {
		c.Flags[k] = m.interpolateMon(c.Flags[k])
	}
}

// --------------------------------------------------------------------------

type ConfigHeartbeat struct {
	Table             string   `yaml:"table,omitempty"`
	Source            []string `yaml:"source,omitempty"`
	Disable           *bool    `yaml:"disable,omitempty"`
	DisableRead       *bool    `yaml:"disable-read,omitempty"`
	DisableWrite      *bool    `yaml:"disable-write,omitempty"`
	DisableAutoSource *bool    `yaml:"disable-auto-source,omitempty"`
}

const (
	DEFAULT_HEARTBEAT_TABLE = "blip.heartbeat"
)

func DefaultConfigHeartbeat() ConfigHeartbeat {
	return ConfigHeartbeat{
		Table: DEFAULT_HEARTBEAT_TABLE,
	}
}

func (c ConfigHeartbeat) Validate() error {
	return nil
}

func (c *ConfigHeartbeat) ApplyDefaults(b Config) {
	if c.Table == "" {
		c.Table = b.Heartbeat.Table
	}
	if len(c.Source) == 0 && len(b.Heartbeat.Source) > 0 {
		c.Source = make([]string, len(b.Heartbeat.Source))
		copy(c.Source, b.Heartbeat.Source)
	}
	c.Disable = setBool(c.Disable, b.Heartbeat.Disable)
	c.DisableRead = setBool(c.DisableRead, b.Heartbeat.DisableRead)
	c.DisableWrite = setBool(c.DisableWrite, b.Heartbeat.DisableWrite)
	c.DisableAutoSource = setBool(c.DisableAutoSource, b.Heartbeat.DisableAutoSource)
}

func (c *ConfigHeartbeat) InterpolateEnvVars() {
	c.Table = interpolateEnv(c.Table)
	for i := range c.Source {
		c.Source[i] = interpolateEnv(c.Source[i])
	}
}

func (c *ConfigHeartbeat) InterpolateMonitor(m *ConfigMonitor) {
	c.Table = m.interpolateMon(c.Table)
	for i := range c.Source {
		c.Source[i] = m.interpolateMon(c.Source[i])
	}
}

// --------------------------------------------------------------------------

// Not implemented yet; placeholders

type ConfigHighAvailability struct{}

func DefaultConfigHA() ConfigHighAvailability {
	return ConfigHighAvailability{}
}

func (c ConfigHighAvailability) Validate() error {
	return nil
}

func (c *ConfigHighAvailability) ApplyDefaults(b Config) {
}

func (c *ConfigHighAvailability) InterpolateEnvVars() {
}

func (c *ConfigHighAvailability) InterpolateMonitor(m *ConfigMonitor) {
}

// --------------------------------------------------------------------------

type ConfigMySQL struct {
	MyCnf          string `yaml:"mycnf,omitempty"`
	Username       string `yaml:"username,omitempty"`
	Password       string `yaml:"password,omitempty"`
	PasswordFile   string `yaml:"password-file,omitempty"`
	TimeoutConnect string `yaml:"timeout-connect,omitempty"`
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
	if c.MyCnf == "" {
		c.MyCnf = b.MySQL.MyCnf
	}
	if c.Username == "" {
		c.Username = b.MySQL.Username
	}
	if c.Password == "" {
		c.Password = b.MySQL.Password
	}
	if c.PasswordFile == "" {
		c.PasswordFile = b.MySQL.Password
	}
	if c.TimeoutConnect == "" {
		c.TimeoutConnect = b.MySQL.TimeoutConnect
	}
}

func (c *ConfigMySQL) InterpolateEnvVars() {
	c.MyCnf = interpolateEnv(c.MyCnf)
	c.Username = interpolateEnv(c.Username)
	c.Password = interpolateEnv(c.Password)
	c.PasswordFile = interpolateEnv(c.PasswordFile)
	c.TimeoutConnect = interpolateEnv(c.TimeoutConnect)
}

func (c *ConfigMySQL) InterpolateMonitor(m *ConfigMonitor) {
	c.MyCnf = m.interpolateMon(c.MyCnf)
	c.Username = m.interpolateMon(c.Username)
	c.Password = m.interpolateMon(c.Password)
	c.PasswordFile = m.interpolateMon(c.PasswordFile)
	c.TimeoutConnect = m.interpolateMon(c.TimeoutConnect)
}

// --------------------------------------------------------------------------

type ConfigPlans struct {
	Files   []string           `yaml:"files,omitempty"`
	Table   string             `yaml:"table,omitempty"`
	Monitor *ConfigMonitor     `yaml:"monitor,omitempty"`
	Adjust  ConfigPlanAdjuster `yaml:"adjust,omitempty"`
}

const (
	DEFAULT_PLANS_FILES  = "plan.yaml"
	DEFAULT_PLANS_TABLE  = "blip.plans"
	DEFAULT_ADJUST_AFTER = "2s"
)

func DefaultConfigPlans() ConfigPlans {
	return ConfigPlans{}
}

func (c ConfigPlans) Validate() error {
	return nil
}

func (c *ConfigPlans) ApplyDefaults(b Config) {
	if len(c.Files) == 0 && len(b.Plans.Files) > 0 {
		c.Files = make([]string, len(b.Plans.Files))
		copy(c.Files, b.Plans.Files)
	}
	c.Adjust.ApplyDefaults(b)
}

func (c *ConfigPlans) InterpolateEnvVars() {
	for i := range c.Files {
		c.Files[i] = interpolateEnv(c.Files[i])
	}
	c.Table = interpolateEnv(c.Table)

	c.Adjust.InterpolateEnvVars()
}

func (c *ConfigPlans) InterpolateMonitor(m *ConfigMonitor) {
	for i := range c.Files {
		c.Files[i] = m.interpolateMon(c.Files[i])
	}
	c.Table = m.interpolateMon(c.Table)

	c.Adjust.InterpolateMonitor(m)
}

type ConfigPlanAdjuster struct {
	Offline  ConfigStatePlan `yaml:"offline,omitempty"`
	Standby  ConfigStatePlan `yaml:"standby,omitempty"`
	ReadOnly ConfigStatePlan `yaml:"read-only,omitempty"`
	Active   ConfigStatePlan `yaml:"active,omitempty"`
}

type ConfigStatePlan struct {
	After string `yaml:"after,omitempty"`
	Plan  string `yaml:"plan,omitempty"`
}

func (c *ConfigPlanAdjuster) ApplyDefaults(b Config) {
	if c.Offline.After == "" {
		c.Offline.After = b.Plans.Adjust.Offline.After
	}
	if c.Offline.Plan == "" {
		c.Offline.Plan = b.Plans.Adjust.Offline.Plan
	}

	if c.Standby.After == "" {
		c.Standby.After = b.Plans.Adjust.Standby.After
	}
	if c.Standby.Plan == "" {
		c.Standby.Plan = b.Plans.Adjust.Standby.Plan
	}

	if c.ReadOnly.After == "" {
		c.ReadOnly.After = b.Plans.Adjust.ReadOnly.After
	}
	if c.ReadOnly.Plan == "" {
		c.ReadOnly.Plan = b.Plans.Adjust.ReadOnly.Plan
	}

	if c.Active.After == "" {
		c.Active.After = b.Plans.Adjust.Active.After
	}
	if c.Active.Plan == "" {
		c.Active.Plan = b.Plans.Adjust.Active.Plan
	}
}

func (c *ConfigPlanAdjuster) InterpolateEnvVars() {
	c.Offline.After = interpolateEnv(c.Offline.After)
	c.Offline.Plan = interpolateEnv(c.Offline.Plan)

	c.Standby.After = interpolateEnv(c.Standby.After)
	c.Standby.Plan = interpolateEnv(c.Standby.Plan)

	c.ReadOnly.After = interpolateEnv(c.ReadOnly.After)
	c.ReadOnly.Plan = interpolateEnv(c.ReadOnly.Plan)

	c.Active.After = interpolateEnv(c.Active.After)
	c.Active.Plan = interpolateEnv(c.Active.Plan)
}

func (c *ConfigPlanAdjuster) InterpolateMonitor(m *ConfigMonitor) {
	c.Offline.Plan = m.interpolateMon(c.Offline.Plan)
	c.Standby.Plan = m.interpolateMon(c.Standby.Plan)
	c.ReadOnly.Plan = m.interpolateMon(c.ReadOnly.Plan)
	c.Active.Plan = m.interpolateMon(c.Active.Plan)
}

func (c ConfigPlanAdjuster) Enabled() bool {
	return c.Offline.Plan != "" ||
		c.Standby.Plan != "" ||
		c.ReadOnly.Plan != "" ||
		c.Active.Plan != ""
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
		opts := c[bk]
		if opts != nil {
			continue
		}
		c[bk] = map[string]string{}
		for k, v := range bv {
			c[bk][k] = v
		}
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
	if c.Cert == "" {
		c.Cert = b.TLS.Cert
	}
	if c.Key == "" {
		c.Key = b.TLS.Key
	}
	if c.CA == "" {
		c.CA = b.TLS.CA
	}
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
