package blip

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path"
	"runtime"
	"time"
)

const VERSION = "0.0.0"

var SHA = ""

// Metrics are metrics collected for one plan level, from one database instance.
type Metrics struct {
	Ts     time.Time
	Plan   string
	Level  string
	DbId   string
	State  string
	Values map[string]float64
}

// Monitor provides information about a MySQL instance that Blip monitors.
type Monitor interface {
	MonitorId() string
	DB() *sql.DB
}

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

func Debug(msg string, v ...interface{}) {
	if !Debugging {
		return
	}
	_, file, line, _ := runtime.Caller(1)
	msg = fmt.Sprintf("%s:%d %s", path.Base(file), line, msg)
	debugLog.Printf(msg, v...)
}
