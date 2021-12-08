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

	"github.com/cashapp/blip/test"
)

var (
	db *sql.DB
)

// First Method that gets run before all tests.
func TestMain(m *testing.M) {
	var err error

	// Read plans from files

	// Connect to MySQL
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/?parseTime=true",
		"root",
		"test",
		"localhost",
		"33570",
	)
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	code := m.Run() // run tests
	os.Exit(code)
}

func TestPrepareForSingleLevelAndNoSource(t *testing.T) {
	// Given a LevelPlan for a collector with single level and no supplied source,
	// test that Prepare correctly constructs the query for that level (ie: select query)
	c := NewGlobal(db)

	defaultPlan := test.ReadPlan(t, "")
	err := c.Prepare(context.Background(), defaultPlan)
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t,
		`SELECT CONCAT_WS(',', @@GLOBAL.read_only) v`,
		c.queryIn["kpi"], // level
	)
}

func TestPrepareWithAllSources(t *testing.T) {
	// Given a LevelPlan for a collector with single level and a custom source,
	// test that Prepare correctly constructs the query for that level
	// using that source query.

	// Test this for all source types ie; auto, select, pfs and show
	queries := map[string]string{
		"auto":   "SELECT CONCAT_WS(',', @@GLOBAL.read_only) v",
		"select": "SELECT CONCAT_WS(',', @@GLOBAL.read_only) v",
		"pfs":    "SELECT variable_name, variable_value from performance_schema.global_variables WHERE variable_name in ('read_only')",
		"show":   "SHOW GLOBAL VARIABLES WHERE variable_name in ('read_only')",
	}

	for src, query := range queries {
		c := NewGlobal(db)
		plan := test.ReadPlan(t, "")
		plan.Levels["kpi"].Collect[blip_domain].Options[OPT_SOURCE] = src
		err := c.Prepare(context.Background(), plan)
		if err != nil {
			t.Error(err)
		}
		assert.Equal(t,
			query,
			c.queryIn["kpi"],
		)
	}
}

func TestPrepareWithCustomInvalidSource(t *testing.T) {

	// Given a LevelPlan for a collector with single level and a custom source,
	// which is invalid, test that Prepare returns an error

	c := NewGlobal(db)
	plan := test.ReadPlan(t, "")
	plan.Levels["kpi"].Collect[blip_domain].Options[OPT_SOURCE] = "this_causes_error"
	err := c.Prepare(context.Background(), plan)
	assert.Error(t, err)
}

func TestPrepareWithInvalidMetricName(t *testing.T) {

	// Given a LevelPlan for a collector with single level and
	// invalid metricname, test that Prepare returns an error

	c := NewGlobal(db)

	plan := test.ReadPlan(t, "")
	dom := plan.Levels["kpi"].Collect[blip_domain]
	dom.Metrics = []string{"max_connections'); DROP TABLE students;--,", "max_prepared_stmt_count"}
	plan.Levels["kpi"].Collect[blip_domain] = dom

	err := c.Prepare(context.Background(), plan)
	assert.Error(t, err)
}

// Metrics collected in test/plans/var_global.yaml
var fourMetrics = []string{"max_connections", "max_prepared_stmt_count", "innodb_log_file_size", "innodb_max_dirty_pages_pct"}

func TestCollectWithSingleLevelPlanAndNoSource(t *testing.T) {

	// Given a plan with single level containing a list of metrics for domain
	// and no custom source, verify that those metrics are retrieved correctly.

	c := NewGlobal(db)
	plan := test.ReadPlan(t, "../../test/plans/var_global.yaml")
	err := c.Prepare(context.Background(), plan)
	if err != nil {
		t.Error(err)
	}
	metrics, _ := c.Collect(context.TODO(), "kpi")
	metricKeys := make([]string, 0, len(metrics))
	for _, m := range metrics {
		metricKeys = append(metricKeys, m.Name)
	}
	assert.ElementsMatch(t, metricKeys, fourMetrics)
}

func TestCollectWithAllSources(t *testing.T) {

	// Given a level plan with custom source option
	// collect should return the list of metrics correctly.

	// Test this for all source types ie; auto, select, pfs and show

	srcs := [4]string{"auto", "select", "pfs", "show"}

	for _, src := range srcs {
		c := NewGlobal(db)
		plan := test.ReadPlan(t, "../../test/plans/var_global.yaml")
		plan.Levels["kpi"].Collect[blip_domain].Options[OPT_SOURCE] = src

		err := c.Prepare(context.Background(), plan)
		if err != nil {
			t.Error(err)
		}
		metrics, _ := c.Collect(context.TODO(), "kpi")
		metricKeys := make([]string, 0, len(metrics))
		for _, m := range metrics {
			metricKeys = append(metricKeys, m.Name)
		}
		assert.ElementsMatch(t, metricKeys, fourMetrics)
	}
}

func TestCollectWithMultipleLevels(t *testing.T) {

	// Given a plan with multiple levels containing a list of metrics for each level
	// verify that those metrics are retrieved correctly only at their respective levels.

	c := NewGlobal(db)
	plan := test.ReadPlan(t, "../../test/plans/var_global_2_levels.yaml")
	err := c.Prepare(context.Background(), plan)
	if err != nil {
		t.Error(err)
	}
	metrics, _ := c.Collect(context.TODO(), "level_1")
	metricKeys := make([]string, 0, len(metrics))
	for _, m := range metrics {
		metricKeys = append(metricKeys, m.Name)
	}
	assert.ElementsMatch(t, metricKeys, []string{"max_connections", "max_prepared_stmt_count"})

	metrics, _ = c.Collect(context.TODO(), "level_2")
	metricKeys = make([]string, 0, len(metrics))
	for _, m := range metrics {
		metricKeys = append(metricKeys, m.Name)
	}
	assert.ElementsMatch(t, metricKeys, []string{"innodb_log_file_size", "innodb_max_dirty_pages_pct"})
}

func TestCollectWithOneNonExistentMetric(t *testing.T) {

	// Given a level plan with single level which contains 3 valid metrics and
	// 1 non existent metric, it should successfully return 3 metrics

	c := NewGlobal(db)
	plan := test.ReadPlan(t, "../../test/plans/var_global_bad_metric.yaml")
	err := c.Prepare(context.Background(), plan)
	if err != nil {
		t.Error(err)
	}
	metrics, _ := c.Collect(context.TODO(), "kpi")
	metricKeys := make([]string, 0, len(metrics))
	for _, m := range metrics {
		metricKeys = append(metricKeys, m.Name)
	}
	assert.ElementsMatch(t, metricKeys, []string{"max_connections", "max_prepared_stmt_count", "innodb_max_dirty_pages_pct"})
}
