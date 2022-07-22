package sizetable

import (
	"github.com/cashapp/blip"
)

func TableSizeQuery(set map[string]string, def blip.CollectorHelp) (string, error) {
	query := `
	SELECT table_schema AS db, table_name as tbl,
		COALESCE(data_length + index_length, 0) AS tbl_size_bytes
		FROM information_schema.TABLES`

	if val := set[OPT_SCHEMA_FILTER]; val == "no" {
		query = query + ` WHERE table_schema NOT IN ('performance_schema', 'information_schema', 'mysql', 'sys')`
	}
	query = query + `;`

	return query, nil
}
