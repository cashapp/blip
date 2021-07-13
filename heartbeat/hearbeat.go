package heartbeat

import (
	"database/sql"
	"time"

	"github.com/square/blip"
)

// Monitor reads and writes heartbeats to a table.
type Monitor interface {
	Run(stopChan, doneChan chan struct{}) error
	Status() error
}

// hb is the main implementation of Monitor.
type hb struct {
	cfg blip.ConfigHeartbeat
	db  *sql.DB
}

func NewMonitor(cfg blip.ConfigHeartbeat, db *sql.DB) *hb {
	return &hb{
		cfg: cfg,
		db:  db,
	}
}

func (m *hb) Run(stopChan, doneChan chan struct{}) error {
	defer close(doneChan)

	d, _ := time.ParseDuration(m.cfg.Freq)
	ticker := time.NewTicker(d)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			// read or write heartbeat
		case <-stopChan:
			return nil
		}
	}
	return nil
}

func (m *hb) Status() error {
	return nil
}
