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

// Monitor reads and writes heartbeats to a table.
type Monitor interface {
	Read(stopChan, doneChan chan struct{}) error
	Write(stopChan, doneChan chan struct{}) error
}

// hb is the main implementation of Monitor.
type hb struct {
	cfg       blip.ConfigHeartbeat
	monitorId string
	db        *sql.DB

	res       time.Duration
	metronome *sync.Cond
	waiter    LagWaiter
}

var _ Monitor = &hb{}

func NewMonitor(monitorId string, cfg blip.ConfigHeartbeat, db *sql.DB, metronome *sync.Cond) *hb {
	return &hb{
		monitorId: monitorId,
		cfg:       cfg,
		db:        db,
		metronome: metronome,
		waiter:    NewSlowFastWaiter(),
	}
}

func (hb *hb) Write(stopChan, doneChan chan struct{}) error {
	defer close(doneChan)
	blip.Debug("hb writing")
	status.Monitor(hb.monitorId, "writer", "writing")

	hb.metronome.L.Lock()
	hb.metronome.Wait() // for tick every 500ms
	ping := fmt.Sprintf("INSERT INTO blip.heartbeat (monitor_id, ts, freq) VALUES ('%s', NOW(3), 500) ON DUPLICATE KEY UPDATE ts=NOW(3), freq=500", hb.monitorId)
	ctx, cancel := context.WithTimeout(context.Background(), 450*time.Millisecond)
	_, err := hb.db.ExecContext(ctx, ping)
	cancel()
	if err != nil {
		// @todo
	}

	ping = fmt.Sprintf("UPDATE blip.heartbeat SET ts=NOW(3) WHERE monitor_id='%s'", hb.monitorId)
	blip.Debug(ping)
	for {
		hb.metronome.Wait() // for tick every 500ms

		// Was Stop called?
		select {
		case <-stopChan: // yes, return immediately
			return nil
		default: // no
		}

		ctx, cancel := context.WithTimeout(context.Background(), 450*time.Millisecond)
		_, err := hb.db.ExecContext(ctx, ping)
		cancel()
		if err != nil {
			blip.Debug(err.Error())
		}
	}
}

func (hb *hb) Read(stopChan, doneChan chan struct{}) error {
	defer close(doneChan)
	q := fmt.Sprintf("SELECT NOW(3), ts, freq FROM blip.heartbeat WHERE monitor_id='%s'", hb.monitorId)
	var waitTime time.Duration
	for {
		select {
		case <-stopChan:
			return nil
		default:
		}

		status.Monitor(hb.monitorId, "reader", "reading")
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond) // @todo
		var now time.Time
		var ts sql.NullTime
		var f int
		err := hb.db.QueryRowContext(ctx, q).Scan(&now, &ts, &f)
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

		waitTime = hb.waiter.Wait(now, ts.Time, f)
		status.Monitor(hb.monitorId, "reader", "sleeping %s", waitTime)
		time.Sleep(waitTime)
	}
}
