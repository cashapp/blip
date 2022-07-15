package stmt

import (
	"context"
	"database/sql"
	"fmt"
)

type stmtMetric interface {
	Name() string
	CollectMetric(ctx context.Context, db *sql.DB) (float64, error)
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

func (q oldestQuery) CollectMetric(ctx context.Context, db *sql.DB) (float64, error) {
	var t float64

	query := `SELECT TIMER_WAIT FROM performance_schema.events_statements_current 
		WHERE EVENT_NAME NOT IN ('statement/com/Sleep','statement/com/Connect','statement/com/Binlog Dump','statement/com/Binlog Dump GTID') 
		ORDER BY TIMER_WAIT DESC LIMIT 1;`

	err := db.QueryRowContext(ctx, query).Scan(&t)
	if err != nil {
		return t, fmt.Errorf("%s failed: %s", query, err)
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

	query := `SELECT COUNT(*) FROM performance_schema.events_statements_current 
		WHERE EVENT_NAME NOT IN ('statement/com/Sleep','statement/com/Connect','statement/com/Binlog Dump','statement/com/Binlog Dump GTID') 
		AND TIMER_WAIT > 30000000000000;`

	err := db.QueryRowContext(ctx, query).Scan(&t)
	if err != nil {
		return t, fmt.Errorf("%s failed: %s", query, err)
	}

	return t, nil
}
