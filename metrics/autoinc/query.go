// Copyright 2024 Block, Inc.

package autoinc

import (
	"strings"
)

const (
	base = `SELECT C.TABLE_SCHEMA, C.TABLE_NAME, C.COLUMN_NAME, C.DATA_TYPE, (locate('unsigned', C.COLUMN_TYPE) > 0) AS is_unsigned, T.AUTO_INCREMENT
	FROM information_schema.COLUMNS C
	JOIN information_schema.TABLES T
		ON C.TABLE_SCHEMA = T.TABLE_SCHEMA
		AND C.TABLE_NAME = T.TABLE_NAME
	WHERE T.TABLE_TYPE = 'BASE TABLE'
	AND C.EXTRA = 'auto_increment'
	AND T.AUTO_INCREMENT IS NOT NULL`
)

func AutoIncrementQuery(set map[string]string) (string, []interface{}, error) {
	var where string
	var params []interface{}
	if include := set[OPT_INCLUDE]; include != "" {
		where, params = setWhere(strings.Split(set[OPT_INCLUDE], ","), true)
	} else {
		where, params = setWhere(strings.Split(set[OPT_EXCLUDE], ","), false)
	}
	return base + where, params, nil
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
				where = where + "(C.TABLE_SCHEMA = ?)"
				params = append(params, db)
			} else {
				where = where + "(C.TABLE_SCHEMA = ? AND C.TABLE_NAME = ?)"
				params = append(params, db, table)
			}
		} else {
			where = where + "(C.TABLE_NAME = ?)"
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
