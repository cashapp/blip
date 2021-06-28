package db

import (
	"database/sql"
)

const (
	DB_STATE_DISCONNECTED = "disconnected"
	DB_STATE_STANDBY      = "standby"
	DB_STATE_READ_ONLY    = "read-only"
	DB_STATE_ACTIVE       = "active"
)

// Instance represents a single MySQL instance.
type Instance struct {
	db *sql.DB
	id string
}

func (in *Instance) Id() string {
	return in.id
}

func (in *Instance) DB() *sql.DB {
	return in.db
}
