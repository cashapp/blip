// Package blip provides high-level data structs and const for integrating with Blip.
package blip

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"runtime"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
)

const VERSION = "0.0.0"

var SHA = ""

type Plugins struct {
	LoadConfig       func(Config) (Config, error)
	LoadLevelPlans   func(Config) ([]Plan, error)
	LoadMonitors     func(Config) ([]ConfigMonitor, error)
	ModifyDB         func(*sql.DB)
	TransformMetrics func(*Metrics) error
}

type Factories struct {
	AWSConfig  AWSConfigFactory
	DbConn     DbFactory
	HTTPClient HTTPClientFactory
}

type AWSConfigFactory interface {
	Make(ConfigAWS) (aws.Config, error)
}

type DbFactory interface {
	Make(ConfigMonitor) (*sql.DB, error)
}

type HTTPClientFactory interface {
	Make(cfg ConfigHTTP, usedFor string) (*http.Client, error)
}

// Collector collects metrics for a single metric domain.
type Collector interface {
	// Domain returns the Blip domain prefix.
	Domain() string

	// Help returns information about using the collector.
	Help() CollectorHelp

	// Prepare prepares a plan for future calls to Collect.
	Prepare(Plan) error

	// Collect collects metrics for the given in the previously prepared plan.
	Collect(ctx context.Context, levelName string) ([]MetricValue, error)
}

type CollectorFactoryArgs struct {
	MonitorId string
	DB        *sql.DB
}

type CollectorFactory interface {
	Make(domain string, args CollectorFactoryArgs) (Collector, error)
}

// Metrics are metrics collected for one plan level, from one database instance.
type Metrics struct {
	Begin     time.Time                // when collection started
	End       time.Time                // when collection completed
	MonitorId string                   // ID of monitor (MySQL)
	Plan      string                   // plan name
	Level     string                   // level name
	State     string                   // state of monitor
	Values    map[string][]MetricValue // keyed on domain
}

type MetricValue struct {
	Name  string
	Value float64
	Type  byte
	Tags  map[string]string
}

// Sink sends metrics to an external destination.
type Sink interface {
	Send(context.Context, *Metrics) error
	Status() error
	Name() string
	MonitorId() string
}

type SinkFactory interface {
	Make(name, monitorId string, opts map[string]string) (Sink, error)
}

const (
	UNKNOWN byte = iota
	COUNTER
	GAUGE
	BOOL
	EVENT
)

const (
	STATE_NONE      = ""
	STATE_OFFLINE   = "offline"
	STATE_STANDBY   = "standby"
	STATE_READ_ONLY = "read-only"
	STATE_ACTIVE    = "active"
)

var (
	Strict    = false
	Debugging = false
	debugLog  = log.New(os.Stderr, "", log.LstdFlags|log.Lmicroseconds)
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
}

func Debug(msg string, v ...interface{}) {
	if !Debugging {
		return
	}
	_, file, line, _ := runtime.Caller(1)
	msg = fmt.Sprintf("DEBUG %s:%d %s", path.Base(file), line, msg)
	debugLog.Printf(msg, v...)
}

// True returns true if b is non-nil and true.
// This is convenience function related to *bool files in config structs,
// which is required for knowing when a bool config is explicitily set
// or not. If set, it's not changed; if not, it's set to the default value.
// That makes a good config experience but a less than ideal code experience
// because !*b will panic if b is nil, hence the need for this func.
func True(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

func MonitorId(cfg ConfigMonitor) string {
	switch {
	case cfg.MonitorId != "":
		return cfg.MonitorId
	case cfg.Hostname != "":
		return cfg.Hostname
	case cfg.Socket != "":
		return cfg.Socket
	}
	return ""
}
