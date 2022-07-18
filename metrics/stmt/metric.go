package stmt

import (
	"context"
	"database/sql"
	"fmt"
)

const (
	OLDEST_QUERY = `SELECT TIMER_WAIT FROM performance_schema.events_statements_current 
		WHERE END_EVENT_ID IS NULL 
		AND EVENT_NAME NOT IN ('statement/com/Sleep','statement/com/Connect','statement/com/Binlog Dump','statement/com/Binlog Dump GTID') 
		ORDER BY TIMER_WAIT DESC LIMIT 1;`
	ACTIVE_LONG_QUERY_COUNT = `SELECT COUNT(*) FROM performance_schema.events_statements_current 
		WHERE END_EVENT_ID IS NULL 
		AND EVENT_NAME NOT IN ('statement/com/Sleep','statement/com/Connect','statement/com/Binlog Dump','statement/com/Binlog Dump GTID') 
		AND TIMER_WAIT > 30000000000000;`
)

// stmtMetric is a metric in the stmt domain
type stmtMetric interface {
	// Gets the name of the metric
	Name() string
	// Collects the metric
	CollectMetric(ctx context.Context, db *sql.DB) (float64, error)
}

// Parent metric
type metric struct {
	name string
}

func (m metric) Name() string {
	return m.name
}

type oldestQuery struct {
	metric
}

func (q oldestQuery) CollectMetric(ctx context.Context, db *sql.DB) (float64, error) {
	var t float64

	err := db.QueryRowContext(ctx, OLDEST_QUERY).Scan(&t)
	if err != nil {
		return t, fmt.Errorf("%s failed: %s", OLDEST_QUERY, err)
	}

	// Convert unit from picoseconds to seconds.
	t /= 1e12

	return t, nil
}

type activeLongQueryCount struct {
	metric
}

func (q activeLongQueryCount) CollectMetric(ctx context.Context, db *sql.DB) (float64, error) {
	var t float64

	err := db.QueryRowContext(ctx, ACTIVE_LONG_QUERY_COUNT).Scan(&t)
	if err != nil {
		return t, fmt.Errorf("%s failed: %s", ACTIVE_LONG_QUERY_COUNT, err)
	}

	return t, nil
}
