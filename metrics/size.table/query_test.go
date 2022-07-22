// Copyright 2022 Block Inc.

package sizetable_test

import (
	"testing"

	sizetable "github.com/cashapp/blip/metrics/size.table"
)

func TestTableSizeQuery(t *testing.T) {
	tableSize := sizetable.NewTable(nil)

	// All defaults
	opts := map[string]string{}
	got, err := sizetable.TableSizeQuery(opts, tableSize.Help())
	expect := `
	SELECT table_schema AS db, table_name as tbl,
		COALESCE(data_length + index_length, 0) AS tbl_size_bytes
		FROM information_schema.TABLES;`
	if err != nil {
		t.Error(err)
	}
	if got != expect {
		t.Errorf("got:\n%s\nexpect:\n%s\n", got, expect)
	}

	// Exclude schemas, mysql, and sys
	opts = map[string]string{
		sizetable.OPT_SCHEMA_FILTER: "no",
	}
	got, err = sizetable.TableSizeQuery(opts, tableSize.Help())
	expect = `
	SELECT table_schema AS db, table_name as tbl,
		COALESCE(data_length + index_length, 0) AS tbl_size_bytes
		FROM information_schema.TABLES  WHERE table_schema NOT IN ('performance_schema', 'information_schema', 'mysql', 'sys');`
	if err != nil {
		t.Error(err)
	}
	if got != expect {
		t.Errorf("got:\n%s\nexpect:\n%s\n", got, expect)
	}
}
