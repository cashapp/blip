package heartbeat

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"github.com/cashapp/blip"
)

// Reader reads heartbeats from a writer. It runs in a separate goroutine and
// reports replication lag for repl.lag metric collectors.
type Reader interface {
	Start() error
	Stop()
	Lag(context.Context) (int64, time.Time, error)
}

// --------------------------------------------------------------------------

// Internal "repo" used by package functions AddReader and RemoveReaders to store
// Reader instances for all monitors' repl.lag collectors.
var mux = &sync.Mutex{}
var readers = map[*sql.DB]map[string]Reader{}

// AddReader adds one Reader unique to latter four arguments. This is called
// only by the repl.lag metrics collector (metrics/repl/lag.go) in Prepare.
func AddReader(reader Reader, db *sql.DB, planName, levelName, mode string) {
	mux.Lock()
	defer mux.Unlock()
	plm := planName + levelName + mode
	all, ok := readers[db]
	if !ok {
		readers[db] = map[string]Reader{}
		all = readers[db]
	}
	r, ok := all[plm]
	if ok {
		r.Stop() // old reader
	}
	all[plm] = r // new reader
	blip.Debug("added reader %p %s %s %s", db, planName, levelName, mode)
}

// RemoveReaders stops and removes all readers previously added to the db.
// The repl.lag metrics collector (in Prepare) adds readers using its db
// because it does not know the monitor ID or any other unique identifiers.
// From the perspective of that metrics collector, its db and current plan
// are unique, so when the plan changes (in Prepare), it stops all readers
// from previous plans (if any) by calling this function.
func RemoveReaders(db *sql.DB) {
	mux.Lock()
	all, ok := readers[db]
	if ok {
		for k := range all {
			all[k].Stop()
		}
	}
	delete(readers, db)
	mux.Unlock()
	blip.Debug("removed readers %p", db)
}

// ResetReaders stops and removes all readers. It is used for testing.
func ResetReaders() {
	mux.Lock()
	for k1 := range readers {
		for k2 := range readers[k1] {
			readers[k1][k2].Stop()
		}
	}
	readers = map[*sql.DB]map[string]Reader{}
	mux.Unlock()
}
