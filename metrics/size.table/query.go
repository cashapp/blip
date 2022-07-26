package sizetable

import (
	"fmt"
	"strings"
)

func TableSizeQuery(set map[string]string) (string, error) {
	query := "SELECT table_schema AS db, table_name as tbl, COALESCE(data_length + index_length, 0) AS tbl_size_bytes FROM information_schema.TABLES"
	where := " WHERE "
	if include := set[OPT_INCLUDE]; include != "" {
		var include_tables []string
		if strings.Contains(set[OPT_INCLUDE], ",") {
			include_tables = strings.Split(set[OPT_INCLUDE], ",")
		} else {
			include_tables = []string{set[OPT_INCLUDE]}
		}
		where = setWhere(include_tables, true, where)
	} else {
		where = where + "NOT "
		var exclude_tables []string
		if strings.Contains(set[OPT_EXCLUDE], ",") {
			exclude_tables = strings.Split(set[OPT_EXCLUDE], ",")
		} else {
			exclude_tables = []string{set[OPT_EXCLUDE]}
		}
		where = setWhere(exclude_tables, false, where)
	}
	return query + where, nil
}

func setWhere(tables []string, isInclude bool, where string) string {
	for i, exclude_table := range tables {
		if strings.Contains(exclude_table, ".") {
			dbAndTable := strings.Split(exclude_table, ".")
			db := dbAndTable[0]
			table := dbAndTable[1]
			if table == "*" {
				where = where + fmt.Sprintf("(table_schema = '%s')", db)
			} else {
				where = where + fmt.Sprintf("(table_schema = '%s' AND table_name = '%s')", db, table)
			}
		} else {
			where = where + fmt.Sprintf("(table_name = '%s')", exclude_table)
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
