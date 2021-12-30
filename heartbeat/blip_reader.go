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
		lag      int64
		waitTime time.Duration
		now      time.Time
		ts       sql.NullTime
		f        int
	)
	for {
		select {
		case <-r.stopChan:
			return
		default:
		}

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond) // @todo
		err := r.db.QueryRowContext(ctx, q).Scan(&now, &ts, &f)
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

		lag, waitTime = r.waiter.Wait(now, ts.Time, f)

		r.Lock()
		r.lag = lag
		r.last = ts.Time
		r.Unlock()

		status.Monitor(r.source, "reader", "running (lag %d ms, sleep %s)", lag, waitTime)
		time.Sleep(waitTime)
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
	return r.lag, r.last, nil
}

// --------------------------------------------------------------------------

type LagWaiter interface {
	Wait(now, then time.Time, f int) (int64, time.Duration)
}

type SlowFastWaiter struct {
	NetworkLatency time.Duration
	waits          int
	lag            int64
}

var _ LagWaiter = &SlowFastWaiter{}

func (w *SlowFastWaiter) Wait(now, then time.Time, freq int) (int64, time.Duration) {
	next := then.Add(time.Duration(freq) * time.Millisecond)
	blip.Debug("then=%s  now=%s  next=%s", then, now, next)

	if now.Before(next) {
		blip.Debug("next hb in %d ms", next.Sub(now).Milliseconds())
		w.waits = 0

		// Wait until next hb
		d := next.Sub(now) + w.NetworkLatency
		if d < 0 {
			d = w.NetworkLatency
		}
		return w.lag, d
	}

	// Next hb is late (lagging)
	w.lag = now.Sub(next).Milliseconds()

	w.waits += 1
	var waitTime time.Duration
	switch {
	case w.waits <= 4: // first 200 ms
		waitTime = time.Duration(50 * time.Millisecond)
		break
	case w.waits <= 4: // next 400 ms (600 ms total)
		waitTime = time.Duration(100 * time.Millisecond)
		break
	default: // remaining >600ms lag
		waitTime = time.Duration(500 * time.Millisecond)
	}

	blip.Debug("lagging: %s past ETA, wait %s (%d)", now.Sub(next), waitTime, w.waits)
	return w.lag, waitTime
}
