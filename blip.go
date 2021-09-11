// Package blip provides high-level data structs and const for integrating with Blip.
package blip

import (
	"fmt"
	"log"
	"os"
	"path"
	"runtime"
	"strings"
	"time"
)

const VERSION = "0.0.0"

var SHA = ""

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
	debugLog  = log.New(os.Stderr, "DEBUG ", log.LstdFlags|log.Lmicroseconds)
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
}

func Debug(msg string, v ...interface{}) {
	if !Debugging {
		return
	}
	_, file, line, _ := runtime.Caller(1)
	msg = fmt.Sprintf("%s:%d %s", path.Base(file), line, msg)
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

func SanitizeTable(table string) string {
	v := strings.SplitN(table, ".", 2)
	if len(v) == 1 {
		return "`" + DEFAULT_DATABASE + "`.`" + v[0] + "`"
	}
	return "`" + v[0] + "`.`" + v[1] + "`"
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
