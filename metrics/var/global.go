package sysvar

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"strings"

	_ "github.com/go-sql-driver/mysql"

	"github.com/square/blip"
	"github.com/square/blip/collect"
)

const (
	OPT_SOURCE = "source"
	OPT_ALL    = "all"

	SOURCE_SELECT = "select"
	SOURCE_PFS    = "pfs"
	SOURCE_SHOW   = "show"
)

var validMetricRegex = regexp.MustCompile("^[a-zA-Z0-9_-]*$")

// Global collects global system variables for the var.global domain.
type Global struct {
	db       *sql.DB
	plans    collect.Plan
	domain   string
	metrics  map[string][]string // keyed on level
	queryIn  map[string]string
	sourceIn map[string]string
}

const (
	blip_domain = "var.global"
)

func NewGlobal(db *sql.DB) *Global {
	return &Global{
		db:       db,
		metrics:  map[string][]string{},
		queryIn:  make(map[string]string),
		sourceIn: make(map[string]string),
	}
}

func (c *Global) Domain() string {
	return blip_domain
}

func (c *Global) Help() collect.Help {
	return collect.Help{
		Domain:      blip_domain,
		Description: "Collect global status variables (sysvars)",
		Options: map[string]collect.HelpOption{
			OPT_SOURCE: {
				Name:    OPT_SOURCE,
				Desc:    "Where to collect sysvars from",
				Default: "auto",
				Values: map[string]string{
					"auto":   "Auto-determine best source",
					"select": "@@GLOBAL.metric_name",
					"pfs":    "performance_schema.global_variables",
					"show":   "SHOW GLOBAL STATUS",
				},
			},
			OPT_ALL: {
				Name: OPT_ALL,
				Desc: "Collect all sysvars",
				Values: map[string]string{
					"yes": "Enable",
					"no":  "Disable",
				},
			},
		},
	}
}

// Prepares queries for all levels in the plan that contain the "var.global" domain
func (c *Global) Prepare(plan collect.Plan) error {
LEVEL:
	for levelName, level := range plan.Levels {
		dom, ok := level.Collect[blip_domain]
		if !ok {
			continue LEVEL // not collected in this level
		}
		err := c.prepareLevel(levelName, dom.Metrics, dom.Options)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Global) Collect(ctx context.Context, levelName string) ([]blip.MetricValue, error) {
	switch c.sourceIn[levelName] {
	case SOURCE_SELECT:
		return c.collectSELECT(ctx, levelName)
	case SOURCE_PFS:
		return c.collectRows(ctx, levelName)
	case SOURCE_SHOW:
		return c.collectRows(ctx, levelName)
	}

	errorStr := fmt.Sprintf("invalid source in Collect %s", c.sourceIn[levelName])
	panic(errorStr)
}

// //////////////////////////////////////////////////////////////////////////
// Internal methods
// //////////////////////////////////////////////////////////////////////////

// Prepares the query for given level based on it's metrics and source option
func (c *Global) prepareLevel(levelName string, metrics []string, options map[string]string) error {

	// Reset in case because prepareLevel can be called multiple times
	// if the LPA changes the plan
	c.sourceIn[levelName] = ""
	c.queryIn[levelName] = ""
	c.metrics[levelName] = []string{}

	// Validate the metricnames for the level
	err := validateMetricNames(metrics)
	if err != nil {
		return err
	}

	// Save metrics to collect for this level
	c.metrics[levelName] = append(c.metrics[levelName], metrics...)

	// -------------------------------------------------------------------------
	// Manual source
	// -------------------------------------------------------------------------

	// If user specified a method, use only that method, whether it works or not
	if src, ok := options[OPT_SOURCE]; ok {
		if len(src) > 0 && src != "auto" {
			switch src {
			case SOURCE_SELECT:
				return c.prepareSELECT(levelName)
			case SOURCE_PFS:
				return c.preparePFS(levelName)
			case SOURCE_SHOW:
				return c.prepareSHOW(levelName, false)
			default:
				return fmt.Errorf("invalid source: %s; valid values: auto, select, pfs, show", src)
			}
		}
	}

	if all, ok := options[OPT_ALL]; ok && strings.ToLower(all) == "yes" {
		return c.prepareSHOW(levelName, true)
	}

	// -------------------------------------------------------------------------
	// Auto source (default)
	// -------------------------------------------------------------------------

	if err = c.prepareSELECT(levelName); err == nil {
		return nil
	}

	if err = c.preparePFS(levelName); err == nil {
		return nil
	}

	if err = c.prepareSHOW(levelName, false); err == nil {
		return nil
	}

	return fmt.Errorf("auto source failed, last error: %s", err)
}

// Validate input metric names to make sure there won't be any
// sql injection attacks.
func validateMetricNames(metricNames []string) error {
	for _, name := range metricNames {
		if !validMetricRegex.MatchString(name) {
			return fmt.Errorf("%s metric isn't a valid metric name", name)
		}
	}
	return nil
}

func (c *Global) prepareSELECT(levelName string) error {
	var globalMetrics = make([]string, len(c.metrics[levelName]))

	for i, str := range c.metrics[levelName] {
		globalMetrics[i] = fmt.Sprintf("@@GLOBAL.%s", str)
	}
	globalMetricString := strings.Join(globalMetrics, ", ")

	c.queryIn[levelName] = fmt.Sprintf("SELECT CONCAT_WS(',', %s) v;", globalMetricString)
	c.sourceIn[levelName] = SOURCE_SELECT

	// Try collecting, discard metrics
	_, err := c.collectSELECT(context.TODO(), levelName)
	return err
}

func (c *Global) preparePFS(levelName string) error {
	var metricString string
	metricString = strings.Join(c.metrics[levelName], "', '")

	query := fmt.Sprintf("SELECT variable_name, variable_value from performance_schema.global_variables WHERE variable_name in ('%s');",
		metricString,
	)
	c.queryIn[levelName] = query
	c.sourceIn[levelName] = SOURCE_PFS

	// Try collecting, discard metrics
	_, err := c.collectRows(context.TODO(), levelName)
	return err
}

func (c *Global) prepareSHOW(levelName string, all bool) error {
	var query string
	if all {
		query = "SHOW GLOBAL VARIABLES"
	} else {
		metricString := strings.Join(c.metrics[levelName], "', '")
		query = fmt.Sprintf("SHOW GLOBAL VARIABLES WHERE variable_name in ('%s');", metricString)
	}

	c.queryIn[levelName] = query
	c.sourceIn[levelName] = SOURCE_SHOW

	// Try collecting, discard metrics
	_, err := c.collectRows(context.TODO(), levelName)
	return err
}

// --------------------------------------------------------------------------

func (c *Global) collectSELECT(ctx context.Context, levelName string) ([]blip.MetricValue, error) {
	rows, err := c.db.QueryContext(ctx, c.queryIn[levelName])
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	metrics := make([]blip.MetricValue, len(c.metrics[levelName]))
	for rows.Next() {
		var val string
		if err := rows.Scan(&val); err != nil {
			log.Println(err)
			// Log error and continue to next row to retrieve next metric
			continue
		}

		values := strings.Split(val, ",")
		for i, metric := range c.metrics[levelName] {
			f, ok := collect.Float64(values[i])
			if !ok {
				continue
			}
			metrics[i] = blip.MetricValue{
				Name:  metric,
				Value: f,
				Type:  blip.GAUGE,
			}
		}
	}

	return metrics, nil
}

// Since both `show` and `pfs` queries return results in same format (ie; 2 columns, name and value)
// use the same logic for querying and retrieving metrics from the results
func (c *Global) collectRows(ctx context.Context, levelName string) ([]blip.MetricValue, error) {
	rows, err := c.db.QueryContext(ctx, c.queryIn[levelName])
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	metrics := []blip.MetricValue{}

	var val string
	var ok bool
	for rows.Next() {
		m := blip.MetricValue{Type: blip.GAUGE}

		if err = rows.Scan(&m.Name, &val); err != nil {
			log.Printf("Error scanning row %s", err)
			// Log error and continue to next row to retrieve next metric
			continue
		}

		m.Value, ok = collect.Float64(val)
		if !ok {
			// log.Printf("Error parsing the metric: %s value: %s as float %s", m.Name, val, err)
			// Log error and continue to next row to retrieve next metric
			continue
		}

		metrics = append(metrics, m)
	}

	return metrics, err
}
