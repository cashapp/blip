// Copyright 2024 Block, Inc.

package sizedatabase_test

import (
	"testing"

	"github.com/cashapp/blip/metrics/size.database"
)

func TestDataSizeQuery(t *testing.T) {
	dataSize := sizedatabase.NewDatabase(nil)

	// All defaults
	opts := map[string]string{}
	got, err := sizedatabase.DataSizeQuery(opts, dataSize.Help())
	expect := "SELECT table_schema AS db, SUM(data_length+index_length) AS bytes FROM information_schema.tables WHERE table_schema NOT IN ('mysql','information_schema','performance_schema','sys') GROUP BY 1"
	if err != nil {
		t.Error(err)
	}
	if got != expect {
		t.Errorf("got:\n%s\nexpect:\n%s\n", got, expect)
	}

	// Exclude specific databases
	opts = map[string]string{
		sizedatabase.OPT_EXCLUDE: "a,b,c",
	}
	got, err = sizedatabase.DataSizeQuery(opts, dataSize.Help())
	expect = "SELECT table_schema AS db, SUM(data_length+index_length) AS bytes FROM information_schema.tables WHERE table_schema NOT IN ('a','b','c') GROUP BY 1"
	if err != nil {
		t.Error(err)
	}
	if got != expect {
		t.Errorf("got:\n%s\nexpect:\n%s\n", got, expect)
	}

	// Include specific databases
	opts = map[string]string{
		sizedatabase.OPT_INCLUDE: "foo,bar",
	}
	got, err = sizedatabase.DataSizeQuery(opts, dataSize.Help())
	expect = "SELECT table_schema AS db, SUM(data_length+index_length) AS bytes FROM information_schema.tables WHERE table_schema IN ('foo','bar') GROUP BY 1"
	if err != nil {
		t.Error(err)
	}
	if got != expect {
		t.Errorf("got:\n%s\nexpect:\n%s\n", got, expect)
	}

	// Include overrides exclude
	opts = map[string]string{
		sizedatabase.OPT_INCLUDE: "foo,bar",
		sizedatabase.OPT_EXCLUDE: "a,b,c", // ignored
	}
	got, err = sizedatabase.DataSizeQuery(opts, dataSize.Help())
	expect = "SELECT table_schema AS db, SUM(data_length+index_length) AS bytes FROM information_schema.tables WHERE table_schema IN ('foo','bar') GROUP BY 1"
	if err != nil {
		t.Error(err)
	}
	if got != expect {
		t.Errorf("got:\n%s\nexpect:\n%s\n", got, expect)
	}

	// LIKE include
	opts = map[string]string{
		sizedatabase.OPT_LIKE:    "yes",
		sizedatabase.OPT_INCLUDE: "foo,bar",
	}
	got, err = sizedatabase.DataSizeQuery(opts, dataSize.Help())
	expect = "SELECT table_schema AS db, SUM(data_length+index_length) AS bytes FROM information_schema.tables WHERE table_schema LIKE 'foo' OR table_schema LIKE 'bar' GROUP BY 1"
	if err != nil {
		t.Error(err)
	}
	if got != expect {
		t.Errorf("got:\n%s\nexpect:\n%s\n", got, expect)
	}

	// LIKE exclude
	opts = map[string]string{
		sizedatabase.OPT_LIKE:    "yes",
		sizedatabase.OPT_EXCLUDE: "x,y,z",
	}
	got, err = sizedatabase.DataSizeQuery(opts, dataSize.Help())
	expect = "SELECT table_schema AS db, SUM(data_length+index_length) AS bytes FROM information_schema.tables WHERE table_schema NOT LIKE 'x' AND table_schema NOT LIKE 'y' AND table_schema NOT LIKE 'z' GROUP BY 1"
	if err != nil {
		t.Error(err)
	}
	if got != expect {
		t.Errorf("got:\n%s\nexpect:\n%s\n", got, expect)
	}

	// Total only (with default exclude)
	opts = map[string]string{
		sizedatabase.OPT_TOTAL: "only",
	}
	got, err = sizedatabase.DataSizeQuery(opts, dataSize.Help())
	expect = "SELECT \"\" AS db, SUM(data_length+index_length) AS bytes FROM information_schema.tables WHERE table_schema NOT IN ('mysql','information_schema','performance_schema','sys')"
	if err != nil {
		t.Error(err)
	}
	if got != expect {
		t.Errorf("got:\n%s\nexpect:\n%s\n", got, expect)
	}
}
