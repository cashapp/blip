package sysvar

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"

	"github.com/square/blip/collect"
	blipdb "github.com/square/blip/db"
)

func createDomain(domainName string, metrics []string, sourceOption ...string) map[string]collect.Domain {
	domainMap := make(map[string]collect.Domain)
	if len(sourceOption) >= 1 {
		domainMap[domainName] = collect.Domain{domainName, make(map[string]string), metrics}
		domainMap[domainName].Options["source"] = sourceOption[0]
	} else {
		domainMap[domainName] = collect.Domain{domainName, nil, metrics}
	}
	return domainMap
}

func createLevel(levelName string, frequency string, domain map[string]collect.Domain) *collect.Level {
	return &collect.Level{
		levelName,
		frequency,
		domain,
	}
}

func createPlan(planName string, levelNames []string, levels []collect.Level) *collect.Plan {
	lvls := make(map[string]collect.Level)
	for i, name := range levelNames {
		lvls[name] = levels[i]
	}
	return &collect.Plan{planName, lvls}
}

// First Method that gets run before all tests.
func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	os.Exit(code)
}

var globalCollector *Global

func setup() {
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/?parseTime=true",
		"root",
		"test",
		"localhost",
		"33570",
	)
	dbIns, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}
	err = dbIns.Ping()
	if err != nil {
		log.Fatalf("Unable to ping the database dsn: %s", dsn)
	}

	globalCollector = NewGlobal(blipdb.NewInstance(dbIns, "test"))
}

func TestPrepareForSingleLevelAndNoSource(t *testing.T) {

	// Given a LevelPlan for a collector with single level and no supplied source,
	// test that Prepare correctly constructs the query for that level (ie: select query)

	domain := createDomain("var.global", []string{"max_connections", "max_prepared_stmt_count", "innodb_log_file_size", "innodb_max_dirty_pages_pct"})
	level := createLevel("vars", "10s", domain)
	plan := createPlan("testPlan", []string{"vars"}, []collect.Level{*level})

	err := globalCollector.Prepare(*plan)
	if err != nil {
		assert.Fail(t, fmt.Sprint("Unable to prepare the collector for metric collection", globalCollector, err))
	}

	assert.Equal(t,
		"SELECT "+
			"CONCAT_WS(',', @@GLOBAL.max_connections, @@GLOBAL.max_prepared_stmt_count,"+
			" @@GLOBAL.innodb_log_file_size, @@GLOBAL.innodb_max_dirty_pages_pct) AS globalvalue;",
		globalCollector.queryIn["vars"],
	)
}

func TestPreparewithAllSources(t *testing.T) {
	// Given a LevelPlan for a collector with single level and a custom source,
	// test that Prepare correctly constructs the query for that level
	// using that source query.

	// Test this for all source types ie; auto, select, pfs and show

	queries := map[string]string{
		"auto": "SELECT " +
			"CONCAT_WS(',', @@GLOBAL.max_connections, @@GLOBAL.max_prepared_stmt_count, " +
			"@@GLOBAL.innodb_log_file_size, @@GLOBAL.innodb_max_dirty_pages_pct) AS globalvalue;",

		"select": "SELECT " +
			"CONCAT_WS(',', @@GLOBAL.max_connections, @@GLOBAL.max_prepared_stmt_count, " +
			"@@GLOBAL.innodb_log_file_size, @@GLOBAL.innodb_max_dirty_pages_pct) AS globalvalue;",

		"pfs": "SELECT variable_name, variable_value " +
			"from performance_schema.global_variables " +
			"WHERE variable_name in ('max_connections', 'max_prepared_stmt_count', " +
			"'innodb_log_file_size', 'innodb_max_dirty_pages_pct');",

		"show": "SHOW GLOBAL VARIABLES " +
			"WHERE variable_name " +
			"in ('max_connections', 'max_prepared_stmt_count', " +
			"'innodb_log_file_size', 'innodb_max_dirty_pages_pct');",
	}

	for src, query := range queries {
		domain := createDomain("var.global", []string{"max_connections", "max_prepared_stmt_count", "innodb_log_file_size", "innodb_max_dirty_pages_pct"}, src)
		level := createLevel("vars", "10s", domain)
		plan := createPlan("testPlan", []string{"vars"}, []collect.Level{*level})
		err := globalCollector.Prepare(*plan)
		if err != nil {
			assert.Fail(t, fmt.Sprint("Unable to prepare the collector for metric collection", globalCollector, err))
		}

		assert.Equal(t,
			query,
			globalCollector.queryIn["vars"],
		)
	}
}

func TestPreparewithCustomInvalidSource(t *testing.T) {

	// Given a LevelPlan for a collector with single level and a custom source,
	// which is invalid, test that Prepare returns an error

	domain := createDomain("var.global", []string{"max_connections", "max_prepared_stmt_count", "innodb_log_file_size", "innodb_max_dirty_pages_pct"}, "shadow")
	level := createLevel("vars", "10s", domain)
	plan := createPlan("testPlan", []string{"vars"}, []collect.Level{*level})
	err := globalCollector.Prepare(*plan)
	assert.Error(t, err)
}

func TestPreparewithInvalidMetricName(t *testing.T) {

	// Given a LevelPlan for a collector with single level and
	// invalid metricname, test that Prepare returns an error

	domain := createDomain("var.global", []string{"max_connections'); DROP TABLE students;--,", "max_prepared_stmt_count"})
	level := createLevel("vars", "10s", domain)
	plan := createPlan("testPlan", []string{"vars"}, []collect.Level{*level})
	err := globalCollector.Prepare(*plan)
	assert.Error(t, err)
}

func TestCollectWithSingleLevelPlanAndNoSource(t *testing.T) {

	// Given a plan with single level containing a list of metrics for domain
	// and no custom source, verify that those metrics are retrieved correctly.

	domain := createDomain("var.global", []string{"max_connections", "max_prepared_stmt_count", "innodb_log_file_size", "innodb_max_dirty_pages_pct"})
	level := createLevel("vars", "10s", domain)
	plan := createPlan("testPlan", []string{"vars"}, []collect.Level{*level})

	err := globalCollector.Prepare(*plan)

	if err != nil {
		assert.Fail(t, fmt.Sprint("Unable to prepare the collector for metric collection", globalCollector, err))
	}
	metrics, _ := globalCollector.Collect(context.TODO(), "vars")
	metricKeys := make([]string, 0, len(metrics.Values))
	for k, _ := range metrics.Values {
		metricKeys = append(metricKeys, k)
	}
	assert.ElementsMatch(t, metricKeys, []string{"max_connections", "max_prepared_stmt_count", "innodb_log_file_size", "innodb_max_dirty_pages_pct"})
}

func TestCollectWithAllSources(t *testing.T) {

	// Given a level plan with custom source option
	// collect should return the list of metrics correctly.

	// Test this for all source types ie; auto, select, pfs and show

	srcs := [4]string{"auto", "select", "pfs", "show"}

	for _, src := range srcs {
		domain := createDomain("var.global", []string{"max_connections", "max_prepared_stmt_count", "innodb_log_file_size", "innodb_max_dirty_pages_pct"}, src)
		level := createLevel("vars", "10s", domain)
		plan := createPlan("testPlan", []string{"vars"}, []collect.Level{*level})

		err := globalCollector.Prepare(*plan)
		if err != nil {
			assert.Fail(t, fmt.Sprintf("Unable to prepare the collector for metric collection for source: %s", src), err)
		}
		metrics, _ := globalCollector.Collect(context.TODO(), "vars")
		metricKeys := make([]string, 0, len(metrics.Values))
		for k, _ := range metrics.Values {
			metricKeys = append(metricKeys, k)
		}
		assert.ElementsMatch(t, metricKeys, []string{"max_connections", "max_prepared_stmt_count", "innodb_log_file_size", "innodb_max_dirty_pages_pct"})
	}
}

func TestCollectWithMultipleLevels(t *testing.T) {

	// Given a plan with multiple levels containing a list of metrics for each level
	// verify that those metrics are retrieved correctly only at their respective levels.

	domain1 := createDomain("var.global", []string{"max_connections", "max_prepared_stmt_count"})
	domain2 := createDomain("var.global", []string{"innodb_log_file_size", "innodb_max_dirty_pages_pct"})
	level1 := createLevel("vars1", "10s", domain1)
	level2 := createLevel("vars2", "10s", domain2)
	plan := createPlan("testPlan", []string{"vars1", "vars2"}, []collect.Level{*level1, *level2})

	err := globalCollector.Prepare(*plan)
	if err != nil {
		assert.Fail(t, fmt.Sprint("Unable to prepare the collector for metric collection", globalCollector, err))
	}
	metrics, _ := globalCollector.Collect(context.TODO(), "vars1")
	metricKeys := make([]string, 0, len(metrics.Values))
	for k, _ := range metrics.Values {
		metricKeys = append(metricKeys, k)
	}
	assert.ElementsMatch(t, metricKeys, []string{"max_connections", "max_prepared_stmt_count"})

	metrics, _ = globalCollector.Collect(context.TODO(), "vars2")
	metricKeys = make([]string, 0, len(metrics.Values))
	for k, _ := range metrics.Values {
		metricKeys = append(metricKeys, k)
	}
	assert.ElementsMatch(t, metricKeys, []string{"innodb_log_file_size", "innodb_max_dirty_pages_pct"})
}

func TestCollectWithOneNonExistentMetric(t *testing.T) {

	// Given a level plan with single level which contains 3 valid metrics and
	// 1 non existent metric, it should successfully return 3 metrics

	domain := createDomain("var.global", []string{"max_connections", "max_prepared_stmt_count", "non_existent_metric", "innodb_max_dirty_pages_pct"})
	level := createLevel("vars", "10s", domain)
	plan := createPlan("testPlan", []string{"vars"}, []collect.Level{*level})

	err := globalCollector.Prepare(*plan)
	if err != nil {
		assert.Fail(t, fmt.Sprintf("Unable to prepare the collector for metric collection"))
	}
	metrics, _ := globalCollector.Collect(context.TODO(), "vars")
	metricKeys := make([]string, 0, len(metrics.Values))
	for k, _ := range metrics.Values {
		metricKeys = append(metricKeys, k)
	}
	assert.ElementsMatch(t, metricKeys, []string{"max_connections", "max_prepared_stmt_count", "innodb_max_dirty_pages_pct"})
}
