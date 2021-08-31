package heartbeat

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

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
	metronome *sync.Cond
}

func NewWriter(monitorId string, db *sql.DB, metronome *sync.Cond) *hbw {
	return &hbw{
		monitorId: monitorId,
		db:        db,
		metronome: metronome,
	}
}

func (w *hbw) Write(stopChan, doneChan chan struct{}) error {
	defer close(doneChan)
	blip.Debug("hb writing")
	status.Monitor(w.monitorId, "writer", "writing")

	w.metronome.L.Lock()
	w.metronome.Wait() // for tick every 500ms
	ping := fmt.Sprintf("INSERT INTO blip.heartbeat (monitor_id, ts, freq) VALUES ('%s', NOW(3), 500) ON DUPLICATE KEY UPDATE ts=NOW(3), freq=500", w.monitorId)
	ctx, cancel := context.WithTimeout(context.Background(), 450*time.Millisecond)
	_, err := w.db.ExecContext(ctx, ping)
	cancel()
	if err != nil {
		// @todo
	}

	ping = fmt.Sprintf("UPDATE blip.heartbeat SET ts=NOW(3) WHERE monitor_id='%s'", w.monitorId)
	blip.Debug(ping)
	for {
		w.metronome.Wait() // for tick every 500ms

		// Was Stop called?
		select {
		case <-stopChan: // yes, return immediately
			return nil
		default: // no
		}

		ctx, cancel := context.WithTimeout(context.Background(), 450*time.Millisecond)
		_, err := w.db.ExecContext(ctx, ping)
		cancel()
		if err != nil {
			blip.Debug(err.Error())
		}
	}
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

	monitorId, err := r.source.Find(r.cfg)
	if err != nil {
		// @todo
	}

	q := fmt.Sprintf("SELECT NOW(3), ts, freq FROM blip.heartbeat WHERE monitor_id='%s'", monitorId)
	var lag int64
	var waitTime time.Duration
	for {
		select {
		case <-stopChan:
			return nil
		default:
		}

		status.Monitor(r.monitorId, "reader", "reading")
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond) // @todo
		var now time.Time
		var ts sql.NullTime
		var f int
		err := r.db.QueryRowContext(ctx, q).Scan(&now, &ts, &f)
		cancel()
		if err != nil {
			switch {
			case err == sql.ErrNoRows:
				// wait for row
				blip.Debug("no row for monitor")
			default:
				blip.Debug(err.Error())
			}
			time.Sleep(1 * time.Second)
			continue
		}

		lag, waitTime = r.waiter.Wait(now, ts.Time, f)
		r.Lock()
		r.lag = lag
		r.last = ts.Time
		r.Unlock()
		status.Monitor(r.monitorId, "reader", "sleeping %s", waitTime)
		time.Sleep(waitTime)
	}
}

func (r *hbr) Report() (int64, time.Time, error) {
	r.Lock()
	defer r.Unlock()
	return r.lag, r.last, nil
}
