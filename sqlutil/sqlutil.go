// Copyright 2022 Block, Inc.

package sqlutil

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"

	my "github.com/go-mysql/errors"
	ver "github.com/hashicorp/go-version"
)

// Float64 converts string to float64, if possible.
func Float64(s string) (float64, bool) {
	f, err := strconv.ParseFloat(s, 64)
	if err == nil {
		return f, true
	}

	switch s {
	case "ON", "YES", "Yes":
		return 1, true
	case "OFF", "NO", "No", "DISABLED":
		return 0, true
	case "Connecting":
		return 0, true
	}

	if ts, err := time.Parse("Jan 02 15:04:05 2006 MST", s); err == nil {
		return float64(ts.Unix()), true
	}
	if ts, err := time.Parse("2006-01-02 15:04:05", s); err == nil {
		return float64(ts.Unix()), true
	}

	return 0, false // failed
}

func CleanObjectName(o string) string {
	o = strings.ReplaceAll(o, ";", "")
	o = strings.ReplaceAll(o, "`", "")
	return strings.TrimSpace(o) // must be last in case Replace make space
}

func ObjectList(csv string, quoteChar string) []string {
	objs := strings.Split(csv, ",")
	for i := range objs {
		objs[i] = quoteChar + CleanObjectName(objs[i]) + quoteChar
	}
	return objs
}

func INList(objs []string, quoteChar string) string {
	if len(objs) == 0 {
		return ""
	}
	in := quoteChar + CleanObjectName(objs[0]) + quoteChar
	for i := range objs[1:] {
		in += "," + quoteChar + CleanObjectName(objs[i]) + quoteChar
	}
	return in
}

func SanitizeTable(table, db string) string {
	v := strings.SplitN(table, ".", 2)
	if len(v) == 1 {
		return "`" + db + "`.`" + v[0] + "`"
	}
	return "`" + v[0] + "`.`" + v[1] + "`"
}

// MySQLVersion returns the MySQL version as integers: major, minor, patch.
func MySQLVersion(ctx context.Context, db *sql.DB) (int, int, int) {
	var val string
	err := db.QueryRowContext(ctx, "SELECT @@version").Scan(&val)
	if err != nil {
		return -1, -1, -1
	}
	cuurentVersion, _ := ver.NewVersion(val)
	v := cuurentVersion.Segments()
	if len(v) != 3 {
		return -1, -1, -1
	}
	return v[0], v[1], v[2]
}

// MySQLVersionGTE returns true if the current MySQL version is >= version.
// It returns false on any error.
func MySQLVersionGTE(version string, db *sql.DB, ctx context.Context) (bool, error) {
	var val string
	err := db.QueryRowContext(ctx, "SELECT @@version").Scan(&val)
	if err != nil {
		return false, err
	}
	cuurentVersion, _ := ver.NewVersion(val)

	targetVersion, err := ver.NewVersion(version)
	if err != nil {
		return false, err
	}

	return cuurentVersion.GreaterThanOrEqual(targetVersion), nil
}

func ReadOnly(err error) bool {
	mysqlError, myerr := my.Error(err)
	if !mysqlError {
		return false
	}
	return myerr == my.ErrReadOnly
}

// RowToMap converts a single row from query (or the last row) to a map of
// strings keyed on column name. All row values a converted to strings.
// This is used for one-row command outputs like SHOW SLAVE|REPLICA STATUS
// that have a mix of values and variaible columns (based on MySQL version)
// but the caller only needs specific cols/vals, so it uses this generic map
// rather than a specific struct.
func RowToMap(ctx context.Context, db *sql.DB, query string) (map[string]string, error) {
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Get list of columns returned by query
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	// Scan() takes pointers, so scanArgs is a list of pointers to values
	scanArgs := make([]interface{}, len(columns))
	values := make([]sql.RawBytes, len(columns))
	for i := range values {
		scanArgs[i] = &values[i]
	}

	for rows.Next() {
		err = rows.Scan(scanArgs...)
		if err != nil {
			return nil, err
		}
	}

	// Map column => value
	m := map[string]string{}
	for i, col := range columns {
		m[col] = fmt.Sprintf("%s", string(values[i]))
	}

	return m, nil
}
