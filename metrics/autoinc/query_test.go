// Copyright 2024 Block, Inc.

package autoinc

import (
	"testing"

	"github.com/go-test/deep"
)

func TestAutoIncrementQuery(t *testing.T) {
	// All defaults
	opts := map[string]string{
		OPT_EXCLUDE: "mysql.*,information_schema.*,performance_schema.*,sys.*",
	}
	got, params, err := AutoIncrementQuery(opts)
	expect := base + " AND (NOT (C.TABLE_SCHEMA = ?) AND NOT (C.TABLE_SCHEMA = ?) AND NOT (C.TABLE_SCHEMA = ?) AND NOT (C.TABLE_SCHEMA = ?))"
	if err != nil {
		t.Error(err)
	}
	if got != expect {
		t.Errorf("got:\n%s\nexpect:\n%s\n", got, expect)
	}

	expectedParams := []interface{}{"mysql", "information_schema", "performance_schema", "sys"}
	if diff := deep.Equal(params, expectedParams); diff != nil {
		t.Error(diff)
	}

	// Exclude schemas, mysql, and sys
	opts = map[string]string{
		OPT_INCLUDE: "test_table,sys.*,information_schema.XTRADB_ZIP_DICT",
	}
	got, params, err = AutoIncrementQuery(opts)
	expect = base + " AND ((C.TABLE_NAME = ?) OR (C.TABLE_SCHEMA = ?) OR (C.TABLE_SCHEMA = ? AND C.TABLE_NAME = ?))"
	if err != nil {
		t.Error(err)
	}
	if got != expect {
		t.Errorf("got:\n%s\nexpect:\n%s\n", got, expect)
	}

	expectedParams = []interface{}{"test_table", "sys", "information_schema", "XTRADB_ZIP_DICT"}
	if diff := deep.Equal(params, expectedParams); diff != nil {
		t.Error(diff)
	}
}
