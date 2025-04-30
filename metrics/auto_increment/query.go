// Copyright 2024 Block, Inc.

package autoincrement

import (
	"strings"
)

func AutoIncrementQuery(set map[string]string) (string, []interface{}, error) {
	query := "SELECT table_schema, table_name, column_name, data_type, auto_increment_ratio, is_unsigned FROM sys.schema_auto_increment_columns WHERE auto_increment_ratio IS NOT NULL"
	var where string
	var params []interface{}
	if include := set[OPT_INCLUDE]; include != "" {
		where, params = setWhere(strings.Split(set[OPT_INCLUDE], ","), true)
	} else {
		where, params = setWhere(strings.Split(set[OPT_EXCLUDE], ","), false)
	}
	return query + where, params, nil
}

func setWhere(tables []string, isInclude bool) (string, []interface{}) {
	where := " AND ("
	if !isInclude {
		where = where + "NOT "
	}
	var params []interface{} = make([]interface{}, 0)
	for i, excludeTable := range tables {
		if strings.Contains(excludeTable, ".") {
			dbAndTable := strings.Split(excludeTable, ".")
			db := dbAndTable[0]
			table := dbAndTable[1]
			if table == "*" {
				where = where + "(table_schema = ?)"
				params = append(params, db)
			} else {
				where = where + "(table_schema = ? AND table_name = ?)"
				params = append(params, db, table)
			}
		} else {
			where = where + "(table_name = ?)"
			params = append(params, excludeTable)
		}
		if i != (len(tables) - 1) {
			if isInclude {
				where = where + " OR "
			} else {
				where = where + " AND NOT "
			}
		}
	}
	where = where + ")"
	return where, params
}
