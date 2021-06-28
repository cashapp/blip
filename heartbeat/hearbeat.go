package heartbeat

import (
	"github.com/square/blip/db"
)

// Monitor reads and writes heartbeats to a table.
type Monitor interface {
	Start() error
	Stop() error
	Status() error
}

// dbMonitor is the main implementation of Monitor.
type dbMonitor struct {
	in db.Instance
}

func NewHeartbeat(in db.Instance) Monitor {
	return &dbMonitor{
		in: in,
	}
}

func (m *dbMonitor) Start() error {
	return nil
}

func (m *dbMonitor) Stop() error {
	return nil
}

func (m *dbMonitor) Status() error {
	return nil
}
