package heartbeat

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/cashapp/blip"
	"github.com/cashapp/blip/status"
)

var BlipReaderBackoff = 2 * time.Second

type BlipReader struct {
	db     *sql.DB
	table  string
	source string
	// --
	waiter LagWaiter
	*sync.Mutex
	lag      int64
	last     time.Time
	stopChan chan struct{}
	doneChan chan struct{}
}

func NewBlipReader(db *sql.DB, table, source string, waiter LagWaiter) *BlipReader {
	return &BlipReader{
		db:     db,
		table:  table,
		source: source,
		// --
		waiter:   waiter,
		Mutex:    &sync.Mutex{},
		stopChan: make(chan struct{}),
		doneChan: make(chan struct{}),
	}
}

func (r *BlipReader) Start() error {
	go r.run()
	return nil
}

func (r *BlipReader) run() {
	defer close(r.doneChan)

	q := fmt.Sprintf("SELECT NOW(3), ts, freq FROM %s WHERE monitor_id='%s'", r.table, r.source)
	blip.Debug("heartbeat reader: %s", q)

	var (
		now  time.Time     // now according to MySQL
		last sql.NullTime  // last heartbeat
		freq int           // freq of heartbeats (milliseconds)
		lag  int64         // lag since last
		wait time.Duration // wait time until next check
	)
	for {
		select {
		case <-r.stopChan:
			return
		default:
		}

		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond) // @todo
		err := r.db.QueryRowContext(ctx, q).Scan(&now, &last, &freq)
		cancel()
		if err != nil {
			switch {
			case err == sql.ErrNoRows:
				status.Monitor(r.source, "reader-error", "no heartbeat for %s", r.source)
			default:
				blip.Debug(err.Error())
				status.Monitor(r.source, "reader-error", err.Error())
			}
			time.Sleep(BlipReaderBackoff)
			continue
		}

		lag, wait = r.waiter.Wait(now, last.Time, freq)

		r.Lock()
		if lag > r.lag {
			r.lag = lag
		}
		r.last = last.Time
		r.Unlock()

		status.Monitor(r.source, "reader", "running (lag %d ms, sleep %s)", lag, wait)
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
	lag := r.lag
	r.lag = 0
	return lag, r.last, nil
}

// --------------------------------------------------------------------------

type LagWaiter interface {
	Wait(now, past time.Time, freq int) (int64, time.Duration)
}

type SlowFastWaiter struct {
	NetworkLatency time.Duration
}

var _ LagWaiter = &SlowFastWaiter{}

func (w *SlowFastWaiter) Wait(now, last time.Time, freq int) (int64, time.Duration) {
	next := last.Add(time.Duration(freq) * time.Millisecond)
	blip.Debug("last=%s  now=%s  next=%s", last, now, next)

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
