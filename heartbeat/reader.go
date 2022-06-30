// Copyright 2022 Block, Inc.

package heartbeat

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cashapp/blip"
	"github.com/cashapp/blip/event"
	"github.com/cashapp/blip/status"
)

// Reader reads heartbeats from a writer. It runs in a separate goroutine and
// reports replication lag for the repl.lag metric collector, where it's also
// created in Prepare. Currently, there's only one implementation: BlipReader,
// but an implementation for pt-table-heartbeat is an idea.
type Reader interface {
	Start() error
	Stop()
	Lag(context.Context) (Lag, error)
}

type Lag struct {
	Milliseconds int64
	LastTs       time.Time
	SourceId     string
	SourceRole   string
	Replica      bool
}

var ReadTimeout = 2 * time.Second
var ReadErrorWait = 1 * time.Second
var ReplCheckWait = 2 * time.Second

// BlipReader reads heartbeats from BlipWriter.
type BlipReader struct {
	monitorId string
	db        *sql.DB
	table     string
	srcId     string
	srcRole   string
	replCheck string
	// --
	waiter LagWaiter
	*sync.Mutex
	lag      int64
	last     time.Time
	stopChan chan struct{}
	doneChan chan struct{}
	isRepl   bool
	event    event.MonitorReceiver
	query    string
}

type BlipReaderArgs struct {
	MonitorId  string
	DB         *sql.DB
	Table      string
	SourceId   string
	SourceRole string
	ReplCheck  string
	Waiter     LagWaiter
}

func NewBlipReader(args BlipReaderArgs) *BlipReader {
	r := &BlipReader{
		monitorId: args.MonitorId,
		db:        args.DB,
		table:     args.Table,
		srcId:     args.SourceId,
		srcRole:   args.SourceRole,
		replCheck: args.ReplCheck,
		// --
		waiter:   args.Waiter,
		Mutex:    &sync.Mutex{},
		stopChan: make(chan struct{}),
		doneChan: make(chan struct{}),
		isRepl:   true,
		event:    event.MonitorReceiver{MonitorId: args.MonitorId},
	}

	// Create heartbeat read query
	cols := []string{"NOW(3)", "ts", "freq", "src_id", "1"}
	var where string
	if r.srcId == "" && r.srcRole == "" {
		blip.Debug("%s: heartbeat from any source (max ts)", r.monitorId)
		where = "ORDER BY ts DESC LIMIT 1"
	} else if r.srcRole != "" {
		blip.Debug("%s: heartbeat from role %s", r.monitorId, r.srcRole)
		where = "WHERE src_role='" + r.srcRole + "' ORDER BY ts DESC LIMIT 1"
	} else {
		blip.Debug("%s: heartbeat from source %s", r.monitorId, r.srcId)
		where = "WHERE src_id='" + r.srcId + "'" // default
	}
	if r.replCheck != "" {
		cols[4] = "@@" + r.replCheck
	}
	r.query = fmt.Sprintf("SELECT %s FROM %s %s", strings.Join(cols, ", "), r.table, where)

	return r
}

func (r *BlipReader) Start() error {
	go r.run()
	return nil
}

func (r *BlipReader) run() {
	defer close(r.doneChan)
	blip.Debug("heartbeat reader: %s", r.query)

	var (
		now    time.Time     // now according to MySQL
		last   sql.NullTime  // last heartbeat
		freq   int           // freq of heartbeats (milliseconds)
		lag    int64         // lag since last
		srcId  string        // source_id, might change if using src_role
		isRepl int           // @@repl-check
		wait   time.Duration // wait time until next check
		err    error
		ctx    context.Context
		cancel context.CancelFunc
	)
	for {
		select {
		case <-r.stopChan:
			return
		default:
		}

		ctx, cancel = context.WithTimeout(context.Background(), ReadTimeout)
		err = r.db.QueryRowContext(ctx, r.query).Scan(&now, &last, &freq, &srcId, &isRepl)
		cancel()
		if err != nil {
			switch {
			case err == sql.ErrNoRows:
				status.Monitor(r.monitorId, "reader-error", "no heartbeat for %s", r.srcId)
			default:
				blip.Debug("%s: %v", r.monitorId, err.Error())
				status.Monitor(r.monitorId, "reader-error", err.Error())
			}
			time.Sleep(ReadErrorWait)
			continue
		}

		if isRepl == 0 {
			r.Lock()
			r.isRepl = false
			r.Unlock()
			status.Monitor(r.monitorId, "reader", "%s (%s) is not a replica (%s=%d), retry in %s",
				srcId, r.srcRole, r.replCheck, isRepl, ReplCheckWait)
			time.Sleep(ReplCheckWait)
			continue
		}

		// Repl source channge?
		if r.srcId != srcId {
			r.srcId = srcId
			r.event.Sendf(event.REPL_SOURCE_CHANGE, "%s to %s", r.srcId, srcId)
		}

		lag, wait = r.waiter.Wait(now, last.Time, freq, srcId)

		r.Lock()
		r.isRepl = true
		r.lag = lag
		r.last = last.Time
		r.Unlock()

		status.Monitor(r.monitorId, "reader", "%d ms lag from %s (%s), next in %s", lag, srcId, r.srcRole, wait)
		time.Sleep(wait)
	}
}

func (r *BlipReader) Stop() {
	r.Lock()
	select {
	case <-r.stopChan:
	case <-r.doneChan:
	default:
		close(r.stopChan)
	}
	r.Unlock()
}

func (r *BlipReader) Lag(_ context.Context) (Lag, error) {
	r.Lock()
	defer r.Unlock()
	if !r.isRepl {
		return Lag{Replica: false}, nil
	}
	//lag := r.lag
	//r.lag = 0
	return Lag{Milliseconds: r.lag, LastTs: r.last, SourceId: r.srcId, SourceRole: r.srcRole, Replica: true}, nil
}

// --------------------------------------------------------------------------

type LagWaiter interface {
	Wait(now, past time.Time, freq int, srcId string) (int64, time.Duration)
}

type SlowFastWaiter struct {
	NetworkLatency time.Duration
}

var _ LagWaiter = &SlowFastWaiter{}

func (w *SlowFastWaiter) Wait(now, last time.Time, freq int, srcId string) (int64, time.Duration) {
	next := last.Add(time.Duration(freq) * time.Millisecond)
	blip.Debug("last=%s  now=%s  next=%s  src=%s", last, now, next, srcId)

	if now.Before(next) {
		lag := now.Sub(last) - w.NetworkLatency
		if lag < 0 {
			lag = 0
		}

		// Wait until next hb
		d := next.Sub(now) + w.NetworkLatency
		blip.Debug("lagged: %d ms; next hb in %d ms", lag.Milliseconds(), next.Sub(now).Milliseconds())
		return lag.Milliseconds(), d
	}

	// Next hb is late (lagging)
	lag := now.Sub(next).Milliseconds()
	var wait time.Duration
	switch {
	case lag < 200:
		wait = time.Duration(50 * time.Millisecond)
		break
	case lag < 600:
		wait = time.Duration(100 * time.Millisecond)
		break
	case lag < 2000:
		wait = time.Duration(500 * time.Millisecond)
		break
	default:
		wait = time.Second
	}

	blip.Debug("lagging: %s; wait %s", now.Sub(next), wait)
	return lag, wait
}
