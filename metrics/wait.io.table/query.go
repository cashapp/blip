// Copyright 2024 Block, Inc.

package waitiotable

import (
	"fmt"
	"strings"
)

func TableIoWaitQuery(set map[string]string, metrics []string) (string, []interface{}) {
	columns := setColumns(set, metrics)

	query := fmt.Sprintf("SELECT %s FROM performance_schema.table_io_waits_summary_by_table", strings.Join(columns, ", "))
	var where string
	var params []interface{}
	if include := set[OPT_INCLUDE]; include != "" {
		where, params = setWhere(strings.Split(set[OPT_INCLUDE], ","), true)
	} else {
		where, params = setWhere(strings.Split(set[OPT_EXCLUDE], ","), false)
	}
	return query + where, params
}

func setColumns(set map[string]string, metrics []string) []string {
	columns := []string{"OBJECT_SCHEMA", "OBJECT_NAME"}

	if all, ok := set[OPT_ALL]; ok && strings.ToLower(all) == "yes" {
		for _, name := range columnNames {
			columns = append(columns, name)
		}
	} else {
		// Default
		for _, metric := range metrics {
			metric = strings.ToLower(metric)

			if _, ok := columnExists[metric]; ok {
				columns = append(columns, metric)
			}
		}
	}

	return columns
}

func setWhere(tables []string, isInclude bool) (string, []interface{}) {
	where := " WHERE "
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
				params = append(params, db)
				where = where + "(OBJECT_SCHEMA = ?)"
			} else {
				params = append(params, db, table)
				where = where + "(OBJECT_SCHEMA = ? AND OBJECT_NAME = ?)"
			}
		} else {
			params = append(params, excludeTable)
			where = where + "(OBJECT_NAME = ?)"
		}
		if i != (len(tables) - 1) {
			if isInclude {
				where = where + " OR "
			} else {
				where = where + " AND NOT "
			}
		}
	}
	return where, params
}
