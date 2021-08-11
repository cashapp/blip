package collect

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/square/blip"
	"github.com/square/blip/dbconn"
)

// PlanLooader is a singleton repo for level plans.
type PlanLoader struct {
	defaultPlans map[string]Plan            // keyed on Plan.Name
	monitorPlans map[string]map[string]Plan // keyed on monitorId, Plan.Name
	needToLoad   map[string]string          // keyed on monitorId => Plan.Table
	*sync.RWMutex
}

var planLoader *PlanLoader

func init() {
	planLoader = &PlanLoader{
		defaultPlans: map[string]Plan{},
		monitorPlans: map[string]map[string]Plan{},
		needToLoad:   map[string]string{},
		RWMutex:      &sync.RWMutex{},
	}
}

func DefaultPlanLoader() *PlanLoader {
	return planLoader
}

func (pl *PlanLoader) SetPlans(plans []Plan) error {
	for _, plan := range plans {
		if plan.MonitorId == "" {
			pl.defaultPlans[plan.Name] = plan // default plan
		} else {
			pl.monitorPlans[plan.MonitorId][plan.Name] = plan // monitor plan
		}
	}
	return nil
}

// LoadPlans loads all plans from a Blip config file. It's called by
// Server.Boot() to load all plan when the user doesn't specify a LoadLevelPlans
// plugin. Monitor plans from a table are deferred until the monitor's LPC
// calls Plan() because the monitor might not be online when Blip starts.
func (pl *PlanLoader) LoadPlans(cfg blip.Config, dbMaker dbconn.Factory) error {
	// ----------------------------------------------------------------------
	// Default plans: config.plans
	// ----------------------------------------------------------------------

	defaultPlans := map[string]Plan{}

	if cfg.Plans.Table != "" {
		// Read default plans from table on cfg.plans.monitor
		blip.Debug("loading plans from %s", cfg.Plans.Table)

		// Connect to db specified by config.plans.monitor, which should have
		// been validated already, but double check. It reuses ConfigMonitor
		// for the DSN info, not because it's an actual db to monitor.
		if cfg.Plans.Monitor == nil {
			if blip.Strict {
				return fmt.Errorf("Table set but Monitor is nil")
			} else {
				blip.Debug("ignoring plans.Table because Monitor=nil and not strict")
			}
		} else {
			db, err := dbMaker.Make(*cfg.Plans.Monitor)
			if err != nil {
				return err
			}
			defer db.Close()

			// Last arg "" = no monitorId, read all rows
			plans, err := ReadPlansFromTable(cfg.Plans.Table, db, "")
			if err != nil {
				return err
			}

			// Save all plans from table by name
			for i, plan := range plans {
				if i == 0 {
					plan.firstRow = true
				}
				defaultPlans[plan.Name] = plan
			}
		}
	}

	if len(cfg.Plans.Files) > 0 {
		// Read all plans from all files
		blip.Debug("loading plans from %v", cfg.Plans.Files)
		plans, err := ReadPlansFromFiles(cfg.Plans.Files)
		if err != nil {
			return err
		}

		// Save all plans from table by name
		for i, plan := range plans {
			if i == 0 {
				plan.firstFile = true
			}
			defaultPlans[plan.Name] = plan

		}
	}

	if len(defaultPlans) == 0 && !blip.Strict {
		// Use built-in internal plan becuase neither config.plans.table
		// nor config.plans.file was specififed
		defaultPlans[blip.INTERNAL_PLAN_NAME] = InternalLevelPlan()
	}

	// ----------------------------------------------------------------------
	// Monitor plans: config.monitors.*.plans
	// ----------------------------------------------------------------------

	monitorPlans := map[string]map[string]Plan{}
	needToLoad := map[string]string{}

	for _, mon := range cfg.Monitors {
		if mon.Plans.Table != "" {
			// Monitor plans from table, but defer until monitor's LPC calls Plan()
			blip.Debug("monitor %s plans from %s (deferred)", mon.MonitorId, mon.Plans.Table)
			needToLoad[mon.MonitorId] = mon.Plans.Table
		}

		if len(mon.Plans.Files) > 0 {
			// Monitor plans from files, load all
			blip.Debug("monitor %s plans from %s", mon.MonitorId, mon.Plans.Files)
			plans, err := ReadPlansFromFiles(mon.Plans.Files)
			if err != nil {
				return err
			}
			monitorPlans[mon.MonitorId] = map[string]Plan{}
			for i, plan := range plans {
				if i == 0 {
					plan.firstFile = true
				}
				monitorPlans[mon.MonitorId][plan.Name] = plan
			}
		}

		if len(monitorPlans[mon.MonitorId]) == 0 {
			// A monitor plan must specifiy either .table or .files. They're
			// mututally exclusive and validated in Server.Boot(). If neither
			// specified, then the monitor uses default plans (config.plans).
			blip.Debug("monitor %s plans from default plans", mon.MonitorId)
		}
	}

	pl.Lock()
	pl.defaultPlans = defaultPlans
	pl.monitorPlans = monitorPlans
	pl.needToLoad = needToLoad
	pl.Unlock()

	return nil
}

func (pl *PlanLoader) PlanNames(monitorId string) []string {
	return nil
}

// Plan returns the plan for the given monitor.
func (pl *PlanLoader) Plan(monitorId string, planName string, db *sql.DB) (Plan, error) {
	pl.RLock()
	defer pl.RUnlock()

	if table, ok := pl.needToLoad[monitorId]; ok {
		pl.RUnlock()

		plans, err := ReadPlansFromTable(table, db, monitorId)
		if err != nil {
			return Plan{}, nil
		}

		pl.Lock() // -- X lock
		pl.monitorPlans[monitorId] = map[string]Plan{}
		for i, plan := range plans {
			if i == 0 {
				plan.firstRow = true
			}
			pl.monitorPlans[monitorId][plan.Name] = plan
		}
		delete(pl.needToLoad, monitorId)
		pl.Lock() // -- X unlock

		pl.RLock()
	}

	// Does monitor have its own plans? If yes, then get plan feom the monitor plans.
	if monitorPlans, ok := pl.monitorPlans[monitorId]; ok {
		if planName != "" {
			plan, ok := monitorPlans[planName]
			if !ok {
				return Plan{}, fmt.Errorf("monitor %s has no plan %s", monitorId, planName)
			}
			return plan, nil
		}
		return pl.firstPlan(monitorPlans), nil
	}

	// Get plan from the default plans (config.plans). This is probably the
	// most common case.
	if planName != "" {
		plan, ok := pl.defaultPlans[planName]
		if !ok {
			return Plan{}, fmt.Errorf("no plan %s", planName)
		}
		return plan, nil
	}

	return pl.firstPlan(pl.defaultPlans), nil
}

func (pl *PlanLoader) firstPlan(plans map[string]Plan) Plan {
	var firstFile, internal string
	for planName, plan := range plans {
		if plan.firstRow {
			return plan
		}
		if plan.firstFile {
			firstFile = planName
			continue
		}
		if plan.internal {
			internal = planName
			continue
		}
	}
	if firstFile != "" {
		return plans[firstFile]
	}
	return plans[internal]
}

func (pl *PlanLoader) Print() {
	pl.RLock()
	defer pl.RUnlock()
	var bytes []byte

	for planName, plan := range pl.defaultPlans {
		bytes, _ = yaml.Marshal(plan.Levels)
		fmt.Printf("---\n# %s\n%s\n\n", planName, string(bytes))
	}

	if len(pl.monitorPlans) > 0 {
		bytes, _ = yaml.Marshal(pl.monitorPlans)
		fmt.Printf("---\n%s\n\n", string(bytes))
	} else {
		fmt.Printf("---\n# No monitor plans\n\n")
	}
}

// //////////////////////////////////////////////////////////////////////////

type planFile map[string]Level

func ReadPlansFromFiles(filePaths []string) ([]Plan, error) {
	plans := []Plan{}

PATHS:
	for _, filePattern := range filePaths {

		files, err := filepath.Glob(filePattern)
		if err != nil {
			if blip.Strict {
				return nil, err
			}
			// @todo log bad glob
			continue PATHS
		}

	FILES:
		for _, file := range files {
			fileabs, err := filepath.Abs(file)
			if err != nil {
				return nil, err
			}

			if _, err := os.Stat(fileabs); err != nil {
				if blip.Strict {
					return nil, fmt.Errorf("config file %s does not exist", fileabs)
				}
				// @todod
				continue FILES
			}

			bytes, err := ioutil.ReadFile(file)
			if err != nil {
				if blip.Strict {
					// err includes file name, e.g. "read config file: open <file>: no such file or directory"
					return nil, fmt.Errorf("cannot read config file: %s", err)
				}
				// @todo
				continue FILES
			}

			var pf planFile
			if err := yaml.Unmarshal(bytes, &pf); err != nil {
				return nil, fmt.Errorf("cannot decode YAML in %s: %s", file, err)
			}

			plan := Plan{
				Name:   file,
				Levels: pf,
			}
			plans = append(plans, plan)
		}
	}

	return plans, nil
}

func ReadPlansFromTable(table string, db *sql.DB, monitorId string) ([]Plan, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	q := fmt.Sprintf("SELECT name, plan, COALESCE(monitorId, '') FROM %s", dbconn.SanitizeTable(table))
	if monitorId != "" {
		q += " WHERE monitorId = '" + monitorId + "' ORDER BY name ASC" // @todo sanitize
	}
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	plans := []Plan{}
	for rows.Next() {
		plan := Plan{}
		var levels string
		err := rows.Scan(&plan.Name, &levels, &plan.MonitorId)
		if err != nil {
			return nil, err
		}
		err = yaml.Unmarshal([]byte(levels), &plan.Levels)
		if err != nil {
			return nil, err
		}
		plans = append(plans, plan)
	}

	return plans, nil
}
