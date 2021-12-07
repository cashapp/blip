package heartbeat

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	my "github.com/go-mysql/errors"

	"github.com/square/blip"
	"github.com/square/blip/sqlutil"
	"github.com/square/blip/status"
)

type BlipWriter struct {
	monitorId string
	db        *sql.DB
	cfg       blip.ConfigHeartbeat
}

func NewBlipWriter(monitorId string, db *sql.DB, cfg blip.ConfigHeartbeat) *BlipWriter {
	if cfg.Freq == "" {
		panic("heartbeat.NewWriter called but config.heartbeat.freq not set")
	}
	if cfg.Table == "" {
		panic("heartbeat.NewWriter called but config.heartbeat.table not set")
	}

	return &BlipWriter{
		monitorId: monitorId,
		db:        db,
		cfg:       cfg,
	}
}

func (w *BlipWriter) Write(stopChan, doneChan chan struct{}) error {
	defer close(doneChan)
	defer status.Monitor(w.monitorId, "BlipWriter", "no running")

	freq, _ := time.ParseDuration(w.cfg.Freq)
	freqMs := freq.Milliseconds()

	table := sqlutil.SanitizeTable(w.cfg.Table, blip.DEFAULT_DATABASE)

	var (
		err    error
		ctx    context.Context
		cancel context.CancelFunc
	)

	status.Monitor(w.monitorId, "BlipWriter", "first insert")
	ping := fmt.Sprintf("INSERT INTO %s (monitor_id, ts, freq) VALUES ('%s', NOW(3), %d) ON DUPLICATE KEY UPDATE ts=NOW(3), freq=%d",
		table, w.monitorId, freqMs, freqMs)
	blip.Debug("hb writing: %s", ping)
	for {
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		_, err = w.db.ExecContext(ctx, ping)
		cancel()
		if err == nil { // success
			break
		}

		// Error --

		status.Monitor(w.monitorId, "BlipWriter", "first insert, failed: %s", err)
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

	status.Monitor(w.monitorId, "BlipWriter", "running")
	ping = fmt.Sprintf("UPDATE %s SET ts=NOW(3) WHERE monitor_id='%s'", table, w.monitorId)
	blip.Debug(ping)
	for {
		time.Sleep(freq)

		ctx, cancel = context.WithTimeout(context.Background(), 400*time.Millisecond)
		_, err = w.db.ExecContext(ctx, ping)
		cancel()
		if err != nil {
			// @todo handle read-only
			blip.Debug(err.Error())
			status.Monitor(w.monitorId, "BlipWriter-error", err.Error())
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
