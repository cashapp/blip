// Copyright 2022 Block, Inc.

package sizetable

import (
	"fmt"
	"strings"
)

func TableSizeQuery(set map[string]string) (string, error) {
	query := "SELECT table_schema AS db, table_name as tbl, COALESCE(data_length + index_length, 0) AS tbl_size_bytes FROM information_schema.TABLES"
	var where string
	if include := set[OPT_INCLUDE]; include != "" {
		where = setWhere(strings.Split(set[OPT_INCLUDE], ","), true)
	} else {
		where = setWhere(strings.Split(set[OPT_EXCLUDE], ","), false)
	}
	return query + where, nil
}

func setWhere(tables []string, isInclude bool) string {
	where := " WHERE "
	if !isInclude {
		where = where + "NOT "
	}
	for i, excludeTable := range tables {
		if strings.Contains(excludeTable, ".") {
			dbAndTable := strings.Split(excludeTable, ".")
			db := dbAndTable[0]
			table := dbAndTable[1]
			if table == "*" {
				where = where + fmt.Sprintf("(table_schema = '%s')", db)
			} else {
				where = where + fmt.Sprintf("(table_schema = '%s' AND table_name = '%s')", db, table)
			}
		} else {
			where = where + fmt.Sprintf("(table_name = '%s')", excludeTable)
		}
		if i != (len(tables) - 1) {
			if isInclude {
				where = where + " OR "
			} else {
				where = where + " AND NOT "
			}
		}
	}
	return where
}

// SELECT table_schema AS db, table_name as tbl, COALESCE(data_length + index_length, 0) AS tbl_size_bytes FROM information_schema.TABLES WHERE NOT (table_schema = 'mysql') AND NOT (table_schema = 'information_schema') AND NOT (table_schema = 'performance_schema') AND NOT (table_schema = 'sys');
// SELECT table_schema AS db, table_name as tbl, COALESCE(data_length + index_length, 0) AS tbl_size_bytes FROM information_schema.TABLES WHERE table_schema NOT IN ('mysql','information_schema','performance_schema','sys');
// mysql.*,information_schema.*,performance.*,sys.*
