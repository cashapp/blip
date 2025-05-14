// Copyright 2024 Block, Inc.

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

// Float64 converts string to float64. If successful, it returns the float64
// value and true, else it returns 0, false.
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

// DEPRECATED: Use interpolated queries instead
func CleanObjectName(o string) string {
	o = strings.ReplaceAll(o, ";", "")
	o = strings.ReplaceAll(o, "`", "")
	return strings.TrimSpace(o) // must be last in case Replace make space
}

// DEPRECATED: Use strings.Split instead in conjunction with
// interpolated queries.
func ObjectList(csv string, quoteChar string) []string {
	objs := strings.Split(csv, ",")
	for i := range objs {
		objs[i] = quoteChar + CleanObjectName(objs[i]) + quoteChar
	}
	return objs
}

// PlaceholderList returns a string of ? placeholders separated by commas.
func PlaceholderList(count int) string {
	if count <= 0 {
		return ""
	}

	return fmt.Sprintf("?%s", strings.Repeat(", ?", count-1))
}

func MultiPlaceholderList(count int, tupleLength int) string {
	if tupleLength == 1 {
		return PlaceholderList(count)
	}

	if count <= 0 {
		return ""
	} else if tupleLength <= 0 {
		return ""
	}

	tuple := fmt.Sprintf("(?%s)", strings.Repeat(", ?", tupleLength-1))
	return fmt.Sprintf("%s%s", tuple, strings.Repeat(", "+tuple, count-1))
}

// ToInterfaceArray converts a list of any type to a list of interface{}.
func ToInterfaceArray[T any](list []T) []interface{} {
	if len(list) == 0 {
		return []interface{}{}
	}

	result := make([]interface{}, 0, len(list))
	for _, value := range list {
		result = append(result, value)
	}

	return result
}

// DEPRECATED: Use interpolated queries instead
func INList(objs []string, quoteChar string) string {
	if len(objs) == 0 {
		return ""
	}
	in := quoteChar + CleanObjectName(objs[0]) + quoteChar
	for i := range objs[1:] {
		in += "," + quoteChar + CleanObjectName(objs[i+1]) + quoteChar
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

// ReadOnly returns true if the err is a MySQL read-only error caused by writing
// to a read-only instance.
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

	// Count rows while scanning, not because there should be only 1, but because
	// MySQL sends the columns ^ even if 0 rows, which makes "for i, col := range columns"
	// below always run, creating a map with cols but empty values. To prevent that,
	// we return a truly empty map if MySQL returns zero rows.
	n := 0
	for rows.Next() {
		err = rows.Scan(scanArgs...)
		if err != nil {
			return nil, err
		}
		n++
	}
	if n == 0 {
		return nil, nil
	}

	// Map column => value
	m := map[string]string{}
	for i, col := range columns {
		m[col] = fmt.Sprintf("%s", string(values[i]))
	}

	return m, nil
}

// RowToTypedMap converts a single row from query (or the last row) to a map of
// type T values keyed on column name. All row values are converted to type T.
// This is used for one-row command outputs which return values of the same type.
func RowToTypedMap[T comparable](ctx context.Context, db *sql.DB, query string) (map[string]T, error) {
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
	values := make([]T, len(columns))
	for i := range values {
		scanArgs[i] = &values[i]
	}

	// Count rows while scanning, not because there should be only 1, but because
	// MySQL sends the columns ^ even if 0 rows, which makes "for i, col := range columns"
	// below always run, creating a map with cols but empty values. To prevent that,
	// we return a truly empty map if MySQL returns zero rows.
	n := 0
	for rows.Next() {
		err = rows.Scan(scanArgs...)
		if err != nil {
			return nil, err
		}
		n++
	}
	if n == 0 {
		return nil, nil
	}

	// Map column => value
	m := map[string]T{}
	for i, col := range columns {
		m[col] = values[i]
	}

	return m, nil
}
