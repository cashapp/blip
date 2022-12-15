// Copyright 2022 Block, Inc.

package waitiotable_test

import (
	"testing"

	waitiotable "github.com/cashapp/blip/metrics/wait.io.table"
)

func TestTableIoQuery(t *testing.T) {
	// All defaults
	opts := map[string]string{
		waitiotable.OPT_EXCLUDE: "mysql.*,information_schema.*,performance_schema.*,sys.*",
		waitiotable.OPT_ALL:     "no",
	}

	metrics := []string{
		"count_fetch",
		"count_insert",
	}

	got := waitiotable.TableIoWaitQuery(opts, metrics)
	expect := "SELECT OBJECT_SCHEMA, OBJECT_NAME, count_fetch, count_insert FROM performance_schema.table_io_waits_summary_by_table WHERE NOT (OBJECT_SCHEMA = 'mysql') AND NOT (OBJECT_SCHEMA = 'information_schema') AND NOT (OBJECT_SCHEMA = 'performance_schema') AND NOT (OBJECT_SCHEMA = 'sys')"
	if got != expect {
		t.Errorf("got:\n%s\nexpect:\n%s\n", got, expect)
	}
	// Exclude schemas, mysql, and sys
	opts = map[string]string{
		waitiotable.OPT_INCLUDE: "test_table,sys.*,information_schema.XTRADB_ZIP_DICT",
		waitiotable.OPT_ALL:     "no",
	}
	got = waitiotable.TableIoWaitQuery(opts, metrics)
	expect = "SELECT OBJECT_SCHEMA, OBJECT_NAME, count_fetch, count_insert FROM performance_schema.table_io_waits_summary_by_table WHERE (OBJECT_NAME = 'test_table') OR (OBJECT_SCHEMA = 'sys') OR (OBJECT_SCHEMA = 'information_schema' AND OBJECT_NAME = 'XTRADB_ZIP_DICT')"
	if got != expect {
		t.Errorf("got:\n%s\nexpect:\n%s\n", got, expect)
	}

	// Use the default columns
	opts = map[string]string{
		waitiotable.OPT_INCLUDE: "test_table,sys.*,information_schema.XTRADB_ZIP_DICT",
		waitiotable.OPT_ALL:     "yes",
	}
	got = waitiotable.TableIoWaitQuery(opts, []string{})
	expect = "SELECT OBJECT_SCHEMA, OBJECT_NAME, sum_timer_wait, min_timer_wait, avg_timer_wait, max_timer_wait, count_read, sum_timer_read, min_timer_read, avg_timer_read, max_timer_read, count_write, sum_timer_write, min_timer_write, avg_timer_write, max_timer_write, count_fetch, sum_timer_fetch, min_timer_fetch, avg_timer_fetch, max_timer_fetch, count_insert, sum_timer_insert, min_timer_insert, avg_timer_insert, max_timer_insert, count_update, sum_timer_update, min_timer_update, avg_timer_update, max_timer_update, count_delete, sum_timer_delete, min_timer_delete, avg_timer_delete, max_timer_delete FROM performance_schema.table_io_waits_summary_by_table WHERE (OBJECT_NAME = 'test_table') OR (OBJECT_SCHEMA = 'sys') OR (OBJECT_SCHEMA = 'information_schema' AND OBJECT_NAME = 'XTRADB_ZIP_DICT')"
	if got != expect {
		t.Errorf("got:\n%s\nexpect:\n%s\n", got, expect)
	}
}
