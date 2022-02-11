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
	"github.com/cashapp/blip/status"
)

// Reader reads heartbeats from a writer. It runs in a separate goroutine and
// reports replication lag for the repl.lag metric collector, where it's also
// created in Prepare. Currently, there's only one implementation: BlipReader,
// but an implementation for pt-table-heartbeat is an idea.
type Reader interface {
	Start() error
	Stop()
	Lag(context.Context) (int64, time.Time, error)
}

var ReadTimeout = 2 * time.Second
var ReadErrorWait = 1 * time.Second
var ReplCheckWait = 2 * time.Second

const NOT_A_REPLICA = -1

// BlipReader reads heartbeats from BlipWriter.
type BlipReader struct {
	monitorId  string
	db         *sql.DB
	table      string
	sourceId   string
	sourceRole string
	replCheck  string
	// --
	waiter LagWaiter
	*sync.Mutex
	lag      int64
	last     time.Time
	stopChan chan struct{}
	doneChan chan struct{}
	isRepl   bool
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
	srcId := args.SourceId
	if srcId == "" {
		srcId = args.MonitorId
	}
	return &BlipReader{
		monitorId:  args.MonitorId,
		db:         args.DB,
		table:      args.Table,
		sourceId:   srcId,
		sourceRole: args.SourceRole,
		replCheck:  args.ReplCheck,
		// --
		waiter:   args.Waiter,
		Mutex:    &sync.Mutex{},
		stopChan: make(chan struct{}),
		doneChan: make(chan struct{}),
		isRepl:   true,
	}
}

func (r *BlipReader) Start() error {
	go r.run()
	return nil
}

func (r *BlipReader) run() {
	defer close(r.doneChan)

	cols := []string{"NOW(3)", "ts", "freq", "''", "1"}
	var where string
	if r.sourceRole == "" {
		where = "src_id='" + r.sourceId + "'" // default
	} else {
		cols[3] = "src_id"
		where = "src_role='" + r.sourceRole + "' ORDER BY ts DESC LIMIT 1"
	}
	if r.replCheck != "" {
		cols[4] = "@@" + r.replCheck
	}
	q := fmt.Sprintf("SELECT %s FROM %s WHERE %s", strings.Join(cols, ", "), r.table, where)
	blip.Debug("heartbeat reader: %s", q)

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
		err = r.db.QueryRowContext(ctx, q).Scan(&now, &last, &freq, &srcId, &isRepl)
		cancel()
		if err != nil {
			switch {
			case err == sql.ErrNoRows:
				status.Monitor(r.monitorId, "reader-error", "no heartbeat for %s", r.sourceId)
			default:
				blip.Debug(err.Error())
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
				srcId, r.sourceRole, r.replCheck, isRepl, ReplCheckWait)
			time.Sleep(ReplCheckWait)
			continue
		}

		lag, wait = r.waiter.Wait(now, last.Time, freq, srcId)

		r.Lock()
		r.isRepl = true
		if lag > r.lag {
			r.lag = lag
		}
		r.last = last.Time
		r.Unlock()

		status.Monitor(r.monitorId, "reader", "%d ms lag from %s (%s), next in %s", lag, srcId, r.sourceRole, wait)
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

func (r *BlipReader) Lag(_ context.Context) (int64, time.Time, error) {
	r.Lock()
	defer r.Unlock()
	if !r.isRepl {
		return NOT_A_REPLICA, time.Time{}, nil
	}
	lag := r.lag
	r.lag = 0
	return lag, r.last, nil
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
