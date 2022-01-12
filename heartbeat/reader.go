package heartbeat

import (
	"context"
	"time"
)

// Reader reads heartbeats from a writer. It runs in a separate goroutine and
// reports replication lag for repl.lag metric collectors. Specific implementations
// are defined in other files (e.g. blip_reader.go) and created in metrics/repl/lag.go.
type Reader interface {
	Start() error
	Stop()
	Lag(context.Context) (int64, time.Time, error)
}
