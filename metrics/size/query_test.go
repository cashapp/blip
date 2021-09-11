package size_test

import (
	"testing"

	"github.com/square/blip/metrics/size"
)

func TestDataSizeQuery(t *testing.T) {
	dataSize := size.NewData(nil)

	// All defaults
	opts := map[string]string{}
	got, err := size.DataSizeQuery(opts, dataSize.Help())
	expect := "SELECT table_schema AS db, SUM(data_length+index_length) AS bytes FROM information_schema.tables WHERE table_schema NOT IN ('mysql','information_schema','performance_schema','sys') GROUP BY 1"
	if err != nil {
		t.Error(err)
	}
	if got != expect {
		t.Errorf("got:\n%s\nexpect:\n%s\n", got, expect)
	}

	// Exclude specific databases
	opts = map[string]string{
		size.OPT_EXCLUDE: "a,b,c",
	}
	got, err = size.DataSizeQuery(opts, dataSize.Help())
	expect = "SELECT table_schema AS db, SUM(data_length+index_length) AS bytes FROM information_schema.tables WHERE table_schema NOT IN ('a','b','c') GROUP BY 1"
	if err != nil {
		t.Error(err)
	}
	if got != expect {
		t.Errorf("got:\n%s\nexpect:\n%s\n", got, expect)
	}

	// Include specific databases
	opts = map[string]string{
		size.OPT_INCLUDE: "foo,bar",
	}
	got, err = size.DataSizeQuery(opts, dataSize.Help())
	expect = "SELECT table_schema AS db, SUM(data_length+index_length) AS bytes FROM information_schema.tables WHERE table_schema IN ('foo','bar') GROUP BY 1"
	if err != nil {
		t.Error(err)
	}
	if got != expect {
		t.Errorf("got:\n%s\nexpect:\n%s\n", got, expect)
	}

	// Include overrides exclude
	opts = map[string]string{
		size.OPT_INCLUDE: "foo,bar",
		size.OPT_EXCLUDE: "a,b,c", // ignored
	}
	got, err = size.DataSizeQuery(opts, dataSize.Help())
	expect = "SELECT table_schema AS db, SUM(data_length+index_length) AS bytes FROM information_schema.tables WHERE table_schema IN ('foo','bar') GROUP BY 1"
	if err != nil {
		t.Error(err)
	}
	if got != expect {
		t.Errorf("got:\n%s\nexpect:\n%s\n", got, expect)
	}

	// LIKE include
	opts = map[string]string{
		size.OPT_LIKE:    "yes",
		size.OPT_INCLUDE: "foo,bar",
	}
	got, err = size.DataSizeQuery(opts, dataSize.Help())
	expect = "SELECT table_schema AS db, SUM(data_length+index_length) AS bytes FROM information_schema.tables WHERE table_schema LIKE 'foo' OR table_schema LIKE 'bar' GROUP BY 1"
	if err != nil {
		t.Error(err)
	}
	if got != expect {
		t.Errorf("got:\n%s\nexpect:\n%s\n", got, expect)
	}

	// LIKE exclude
	opts = map[string]string{
		size.OPT_LIKE:    "yes",
		size.OPT_EXCLUDE: "x,y,z",
	}
	got, err = size.DataSizeQuery(opts, dataSize.Help())
	expect = "SELECT table_schema AS db, SUM(data_length+index_length) AS bytes FROM information_schema.tables WHERE table_schema NOT LIKE 'x' AND table_schema NOT LIKE 'y' AND table_schema NOT LIKE 'z' GROUP BY 1"
	if err != nil {
		t.Error(err)
	}
	if got != expect {
		t.Errorf("got:\n%s\nexpect:\n%s\n", got, expect)
	}

	// Total only (with default exclude)
	opts = map[string]string{
		size.OPT_TOTAL: "only",
	}
	got, err = size.DataSizeQuery(opts, dataSize.Help())
	expect = "SELECT \"\" AS db, SUM(data_length+index_length) AS bytes FROM information_schema.tables WHERE table_schema NOT IN ('mysql','information_schema','performance_schema','sys')"
	if err != nil {
		t.Error(err)
	}
	if got != expect {
		t.Errorf("got:\n%s\nexpect:\n%s\n", got, expect)
	}
}
