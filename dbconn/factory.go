package dbconn

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"

	"github.com/square/blip"
)

type Factory interface {
	Make(blip.ConfigMonitor) (*sql.DB, error)
}

var _ Factory = connFactory{}

type connFactory struct {
}

func NewConnFactory() connFactory {
	return connFactory{}
}

func (f connFactory) Make(mon blip.ConfigMonitor) (*sql.DB, error) {
	db, err := sql.Open("mysql", "root@tcp(127.0.0.1:3306)/")
	if err != nil {
		return nil, err
	}
	return db, nil
}
