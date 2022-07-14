package stmt

import "database/sql"

const (
	OldestQuery = `SELECT TIMER_WAIT FROM performance_schema.events_statements_current 
		WHERE EVENT_NAME NOT IN ('statement/com/Sleep','statement/com/Connect','statement/com/Binlog Dump','statement/com/Binlog Dump GTID') 
		ORDER BY TIMER_WAIT DESC LIMIT 1;`
	ActiveLongRunningQueries = `SELECT * FROM performance_schema.events_statements_current 
		WHERE EVENT_NAME NOT IN ('statement/com/Sleep','statement/com/Connect','statement/com/Binlog Dump','statement/com/Binlog Dump GTID') 
		AND TIMER_WAIT > 30000000000000;`
)

type stmtMetric interface {
	Name() string
	CollectMetric(db *sql.DB) float64
}

type metric struct {
	name string
}

func (m metric) Name() string {
	return m.name
}

type oldestQuery struct {
	metric
}

func (q oldestQuery) CollectMetric(db *sql.DB) float64 {
	return 0.0
}

type activeLongQueryCount struct {
	metric
}

func (q activeLongQueryCount) CollectMetric(db *sql.DB) float64 {
	return 0.0
}
