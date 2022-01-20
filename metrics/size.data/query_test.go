package sizedata_test

import (
	"testing"

	"github.com/cashapp/blip/metrics/size.data"
)

func TestDataSizeQuery(t *testing.T) {
	dataSize := sizedata.NewData(nil)

	// All defaults
	opts := map[string]string{}
	got, err := sizedata.DataSizeQuery(opts, dataSize.Help())
	expect := "SELECT table_schema AS db, SUM(data_length+index_length) AS bytes FROM information_schema.tables WHERE table_schema NOT IN ('mysql','information_schema','performance_schema','sys') GROUP BY 1"
	if err != nil {
		t.Error(err)
	}
	if got != expect {
		t.Errorf("got:\n%s\nexpect:\n%s\n", got, expect)
	}

	// Exclude specific databases
	opts = map[string]string{
		sizedata.OPT_EXCLUDE: "a,b,c",
	}
	got, err = sizedata.DataSizeQuery(opts, dataSize.Help())
	expect = "SELECT table_schema AS db, SUM(data_length+index_length) AS bytes FROM information_schema.tables WHERE table_schema NOT IN ('a','b','c') GROUP BY 1"
	if err != nil {
		t.Error(err)
	}
	if got != expect {
		t.Errorf("got:\n%s\nexpect:\n%s\n", got, expect)
	}

	// Include specific databases
	opts = map[string]string{
		sizedata.OPT_INCLUDE: "foo,bar",
	}
	got, err = sizedata.DataSizeQuery(opts, dataSize.Help())
	expect = "SELECT table_schema AS db, SUM(data_length+index_length) AS bytes FROM information_schema.tables WHERE table_schema IN ('foo','bar') GROUP BY 1"
	if err != nil {
		t.Error(err)
	}
	if got != expect {
		t.Errorf("got:\n%s\nexpect:\n%s\n", got, expect)
	}

	// Include overrides exclude
	opts = map[string]string{
		sizedata.OPT_INCLUDE: "foo,bar",
		sizedata.OPT_EXCLUDE: "a,b,c", // ignored
	}
	got, err = sizedata.DataSizeQuery(opts, dataSize.Help())
	expect = "SELECT table_schema AS db, SUM(data_length+index_length) AS bytes FROM information_schema.tables WHERE table_schema IN ('foo','bar') GROUP BY 1"
	if err != nil {
		t.Error(err)
	}
	if got != expect {
		t.Errorf("got:\n%s\nexpect:\n%s\n", got, expect)
	}

	// LIKE include
	opts = map[string]string{
		sizedata.OPT_LIKE:    "yes",
		sizedata.OPT_INCLUDE: "foo,bar",
	}
	got, err = sizedata.DataSizeQuery(opts, dataSize.Help())
	expect = "SELECT table_schema AS db, SUM(data_length+index_length) AS bytes FROM information_schema.tables WHERE table_schema LIKE 'foo' OR table_schema LIKE 'bar' GROUP BY 1"
	if err != nil {
		t.Error(err)
	}
	if got != expect {
		t.Errorf("got:\n%s\nexpect:\n%s\n", got, expect)
	}

	// LIKE exclude
	opts = map[string]string{
		sizedata.OPT_LIKE:    "yes",
		sizedata.OPT_EXCLUDE: "x,y,z",
	}
	got, err = sizedata.DataSizeQuery(opts, dataSize.Help())
	expect = "SELECT table_schema AS db, SUM(data_length+index_length) AS bytes FROM information_schema.tables WHERE table_schema NOT LIKE 'x' AND table_schema NOT LIKE 'y' AND table_schema NOT LIKE 'z' GROUP BY 1"
	if err != nil {
		t.Error(err)
	}
	if got != expect {
		t.Errorf("got:\n%s\nexpect:\n%s\n", got, expect)
	}

	// Total only (with default exclude)
	opts = map[string]string{
		sizedata.OPT_TOTAL: "only",
	}
	got, err = sizedata.DataSizeQuery(opts, dataSize.Help())
	expect = "SELECT \"\" AS db, SUM(data_length+index_length) AS bytes FROM information_schema.tables WHERE table_schema NOT IN ('mysql','information_schema','performance_schema','sys')"
	if err != nil {
		t.Error(err)
	}
	if got != expect {
		t.Errorf("got:\n%s\nexpect:\n%s\n", got, expect)
	}
}
