// Copyright 2022 Block Inc.

package sizetable_test

import (
	"fmt"
	"testing"

	sizetable "github.com/cashapp/blip/metrics/size.table"
)

func TestTableSizeQuery(t *testing.T) {
	// All defaults
	opts := map[string]string{
		sizetable.OPT_EXCLUDE: "mysql.*,information_schema.*,performance_schema.*,sys.*",
	}
	got, err := sizetable.TableSizeQuery(opts)
	expect := "SELECT table_schema AS db, table_name as tbl, COALESCE(data_length + index_length, 0) AS tbl_size_bytes FROM information_schema.TABLES WHERE NOT (table_schema = 'mysql') AND NOT (table_schema = 'information_schema') AND NOT (table_schema = 'performance_schema') AND NOT (table_schema = 'sys')"
	if err != nil {
		t.Error(err)
	}
	if got != expect {
		t.Errorf("got:\n%s\nexpect:\n%s\n", got, expect)
	}
	fmt.Printf("\nexpected: %v\ngot: %v \n", expect, got)
	// Exclude schemas, mysql, and sys
	opts = map[string]string{
		sizetable.OPT_INCLUDE: "test_table,sys.*,information_schema.XTRADB_ZIP_DICT",
	}
	got, err = sizetable.TableSizeQuery(opts)
	expect = "SELECT table_schema AS db, table_name as tbl, COALESCE(data_length + index_length, 0) AS tbl_size_bytes FROM information_schema.TABLES WHERE (table_name = 'test_table') OR (table_schema = 'sys') OR (table_schema = 'information_schema' AND table_name = 'XTRADB_ZIP_DICT')"
	fmt.Printf("\nexpected: %v\ngot: %v \n", expect, got)
	if err != nil {
		t.Error(err)
	}
	if got != expect {
		t.Errorf("got:\n%s\nexpect:\n%s\n", got, expect)
	}
}
