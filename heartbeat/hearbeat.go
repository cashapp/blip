package heartbeat

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	my "github.com/go-mysql/errors"

	"github.com/square/blip"
	"github.com/square/blip/status"
)

type Args struct {
	Cfg        blip.ConfigHeartbeat
	MonnitorId string
	DB         *sql.DB
	Metronome  *sync.Cond
}

// Writer writes heartbeats to a table.
type Writer interface {
	Write(stopChan, doneChan chan struct{}) error
}

// hb is the main implementation of Monitor.
type hbw struct {
	monitorId string
	db        *sql.DB
}

func NewWriter(monitorId string, db *sql.DB) *hbw {
	return &hbw{
		monitorId: monitorId,
		db:        db,
	}
}

func (w *hbw) Write(stopChan, doneChan chan struct{}) error {
	defer close(doneChan)
	defer status.Monitor(w.monitorId, "writer", "no running")

	status.Monitor(w.monitorId, "writer", "first insert")
	ping := fmt.Sprintf("INSERT INTO blip.heartbeat (monitor_id, ts, freq) VALUES ('%s', NOW(3), 500) ON DUPLICATE KEY UPDATE ts=NOW(3), freq=500", w.monitorId)
	blip.Debug("hb writing: %s", ping)
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 450*time.Millisecond)
		_, err := w.db.ExecContext(ctx, ping)
		cancel()
		if err == nil { // success
			break
		}

		status.Monitor(w.monitorId, "writer", "first insert, failed: %s", err)
		blip.Debug("%s: first insert, failed: %s", w.monitorId, err)
		if ok, myerr := my.Error(err); ok && myerr == my.ErrReadOnly {
			time.Sleep(5 * time.Second)
		} else {
			time.Sleep(2 * time.Second)
		}

		// Was Stop called?
		select {
		case <-stopChan: // yes, return immediately
			return nil
		default: // no
		}
	}

	status.Monitor(w.monitorId, "writer", "running")
	ping = fmt.Sprintf("UPDATE blip.heartbeat SET ts=NOW(3) WHERE monitor_id='%s'", w.monitorId)
	blip.Debug(ping)

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 400*time.Millisecond)
		_, err := w.db.ExecContext(ctx, ping)
		cancel()
		if err != nil {
			// @todo handle read-only
			blip.Debug(err.Error())
			status.Monitor(w.monitorId, "writer-error", err.Error())
		}

		// Was Stop called?
		select {
		case <-stopChan: // yes, return immediately
			return nil
		default: // no
		}
	}

	return nil
}

// --------------------------------------------------------------------------

// Monitor reads and writes heartbeats to a table.
type Reader interface {
	Read(stopChan, doneChan chan struct{}) error
	Report() (int64, time.Time, error)
}

type hbr struct {
	cfg       blip.ConfigMonitor
	monitorId string
	db        *sql.DB
	waiter    LagWaiter
	source    SourceFinder
	// --
	*sync.Mutex
	lag  int64
	last time.Time
}

func NewReader(cfg blip.ConfigMonitor, db *sql.DB, waiter LagWaiter, source SourceFinder) *hbr {
	return &hbr{
		cfg:       cfg,
		monitorId: cfg.MonitorId,
		db:        db,
		waiter:    waiter,
		source:    source,
		// --
		Mutex: &sync.Mutex{},
	}
}

func (r *hbr) Read(stopChan, doneChan chan struct{}) error {
	defer close(doneChan)
	status.Monitor(r.monitorId, "reader", "not running")

	monitorId, err := r.source.Find(r.cfg)
	if err != nil {
		// @todo
	}

	status.Monitor(r.monitorId, "reader", "running")
	q := fmt.Sprintf("SELECT NOW(3), ts, freq FROM blip.heartbeat WHERE monitor_id='%s'", monitorId)
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
		case <-stopChan:
			return nil
		default:
		}

		status.Monitor(r.monitorId, "reader", "running")
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond) // @todo
		err := r.db.QueryRowContext(ctx, q).Scan(&now, &ts, &f)
		cancel()
		if err != nil {
			switch {
			case err == sql.ErrNoRows:
				status.Monitor(r.monitorId, "reader-error", "no heartbeat for %s", monitorId)
			default:
				blip.Debug(err.Error())
				status.Monitor(r.monitorId, "reader-error", err.Error())
			}
			time.Sleep(2 * time.Second)
			continue
		}

		lag, waitTime = r.waiter.Wait(now, ts.Time, f)

		r.Lock()
		r.lag = lag
		r.last = ts.Time
		r.Unlock()

		status.Monitor(r.monitorId, "reader", "running (lag %d ms, sleep %s)", lag, waitTime)
		time.Sleep(waitTime)
	}
}

func (r *hbr) Report() (int64, time.Time, error) {
	r.Lock()
	defer r.Unlock()
	return r.lag, r.last, nil
}
