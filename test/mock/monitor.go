package mock

import (
	"database/sql"
)

type Monitor struct {
	MonitorIdFunc func() string
	DBFunc        func() *sql.DB
}

func (m Monitor) MonitorId() string {
	if m.MonitorIdFunc != nil {
		return m.MonitorIdFunc()
	}
	return ""
}

func (m Monitor) DB() *sql.DB {
	if m.DBFunc != nil {
		return m.DBFunc()
	}
	return nil
}
