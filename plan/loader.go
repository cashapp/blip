package plan

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
	"github.com/square/blip/sqlutil"
)

// planMeta is a blip.Plan plus metadata.
type planMeta struct {
	plan blip.Plan

	// First is true for the first plan loaded from any source. Loader uses
	// this to return the first plan when there are multiple plans but no LPA to
	// set plans based on state.
	firstRow  bool
	firstFile bool
	internal  bool
}

// PlanLooader is a singleton service and repo for level plans.
type Loader struct {
	cfg          blip.Config
	plugin       func(blip.Config) ([]blip.Plan, error)
	sharedPlans  map[string]planMeta            // keyed on Plan.Name
	monitorPlans map[string]map[string]planMeta // keyed on monitorId, Plan.Name
	needToLoad   map[string]string              // keyed on monitorId => Plan.Table
	*sync.RWMutex
}

func NewLoader(cfg blip.Config, plugin func(blip.Config) ([]blip.Plan, error)) *Loader {
	return &Loader{
		cfg:          cfg,
		plugin:       plugin,
		sharedPlans:  map[string]planMeta{},
		monitorPlans: map[string]map[string]planMeta{},
		needToLoad:   map[string]string{},
		RWMutex:      &sync.RWMutex{},
	}
}

// LoadPlans loads all plans from a Blip config file. It's called by
// Server.Boot() to load all plan when the user doesn't specify a LoadLevelPlans
// plugin. Monitor plans from a table are deferred until the monitor's LPC
// calls Plan() because the monitor might not be online when Blip starts.
func (pl *Loader) LoadPlans(dbMaker blip.DbFactory) error {
	// ----------------------------------------------------------------------
	// Shared plans: config.plans
	// ----------------------------------------------------------------------

	sharedPlans := map[string]planMeta{}

	if pl.plugin != nil {
		plans, err := pl.plugin(pl.cfg)
		if err != nil {
			return err
		}
		for i, plan := range plans {
			pm := planMeta{
				plan:      plan,
				firstRow:  i == 0,
				firstFile: i == 0,
			}
			sharedPlans[plan.Name] = pm
		}
		return nil
	}

	if pl.cfg.Plans.Table != "" {
		// Read default plans from table on pl.cfg.plans.monitor
		blip.Debug("loading plans from %s", pl.cfg.Plans.Table)

		// Connect to db specified by config.plans.monitor, which should have
		// been validated already, but double check. It reuses ConfigMonitor
		// for the DSN info, not because it's an actual db to monitor.
		if pl.cfg.Plans.Monitor == nil {
			if blip.Strict {
				return fmt.Errorf("Table set but Monitor is nil")
			} else {
				blip.Debug("ignoring plans.Table because Monitor=nil and not strict")
			}
		} else {
			db, err := dbMaker.Make(*pl.cfg.Plans.Monitor)
			if err != nil {
				return err
			}
			defer db.Close()

			// Last arg "" = no monitorId, read all rows
			plans, err := ReadPlansFromTable(pl.cfg.Plans.Table, db, "")
			if err != nil {
				return err
			}

			// Save all plans from table by name
			for i, plan := range plans {
				sharedPlans[plan.Name] = planMeta{
					plan:     plan,
					firstRow: i == 0,
				}
			}
		}
	}

	if len(pl.cfg.Plans.Files) > 0 {
		// Read all plans from all files
		blip.Debug("loading plans from %v", pl.cfg.Plans.Files)
		plans, err := ReadPlansFromFiles(pl.cfg.Plans.Files)
		if err != nil {
			return err
		}

		// Save all plans from table by name
		for i, plan := range plans {
			sharedPlans[plan.Name] = planMeta{
				plan:      plan,
				firstFile: i == 0,
			}
		}
	}

	if len(sharedPlans) == 0 && !blip.Strict {
		// Use built-in internal plan becuase neither config.plans.table
		// nor config.plans.file was specififed
		sharedPlans[blip.INTERNAL_PLAN_NAME] = planMeta{
			plan:     blip.InternalLevelPlan(),
			internal: true,
		}
	}

	// ----------------------------------------------------------------------
	// Monitor plans: config.monitors.*.plans
	// ----------------------------------------------------------------------

	monitorPlans := map[string]map[string]planMeta{}
	needToLoad := map[string]string{}

	for _, mon := range pl.cfg.Monitors {
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
			monitorPlans[mon.MonitorId] = map[string]planMeta{}
			for i, plan := range plans {
				monitorPlans[mon.MonitorId][plan.Name] = planMeta{
					plan:      plan,
					firstFile: i == 0,
				}
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
	pl.sharedPlans = sharedPlans
	pl.monitorPlans = monitorPlans
	pl.needToLoad = needToLoad
	pl.Unlock()

	return nil
}

func (pl *Loader) PlanNames(monitorId string) []string {
	return nil
}

// Plan returns the plan for the given monitor.
func (pl *Loader) Plan(monitorId string, planName string, db *sql.DB) (blip.Plan, error) {
	pl.RLock()
	defer pl.RUnlock()

	if table, ok := pl.needToLoad[monitorId]; ok {
		pl.RUnlock()

		plans, err := ReadPlansFromTable(table, db, monitorId)
		if err != nil {
			return blip.Plan{}, nil
		}

		pl.Lock() // -- X lock
		pl.monitorPlans[monitorId] = map[string]planMeta{}
		for i, plan := range plans {
			pl.monitorPlans[monitorId][plan.Name] = planMeta{
				plan:     plan,
				firstRow: i == 0,
			}
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
				return blip.Plan{}, fmt.Errorf("monitor %s has no plan %s", monitorId, planName)
			}
			return plan.plan, nil
		}
		return pl.firstPlan(monitorPlans), nil
	}

	// Get plan from the default plans (config.plans). This is probably the
	// most common case.
	if planName != "" {
		plan, ok := pl.sharedPlans[planName]
		if !ok {
			return blip.Plan{}, fmt.Errorf("no plan %s", planName)
		}
		return plan.plan, nil
	}

	return pl.firstPlan(pl.sharedPlans), nil
}

func (pl *Loader) firstPlan(plans map[string]planMeta) blip.Plan {
	var firstFile, internal string
	for planName, plan := range plans {
		if plan.firstRow {
			return plan.plan
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
		return plans[firstFile].plan
	}
	return plans[internal].plan
}

func (pl *Loader) Print() {
	pl.RLock()
	defer pl.RUnlock()
	var bytes []byte

	for planName, plan := range pl.sharedPlans {
		bytes, _ = yaml.Marshal(plan.plan.Levels)
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

type planFile map[string]*blip.Level

func ReadPlansFromFiles(filePaths []string) ([]blip.Plan, error) {
	plans := []blip.Plan{}

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

			levels := make(map[string]blip.Level, len(pf))
			for k := range pf {
				levels[k] = blip.Level{
					Name:    k, // must have, levels are collected by name
					Freq:    pf[k].Freq,
					Collect: pf[k].Collect,
				}
			}

			plan := blip.Plan{
				Name:   file,
				Levels: levels,
			}
			plans = append(plans, plan)
		}
	}

	return plans, nil
}

func ReadPlansFromTable(table string, db *sql.DB, monitorId string) ([]blip.Plan, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	q := fmt.Sprintf("SELECT name, plan, COALESCE(monitorId, '') FROM %s", sqlutil.SanitizeTable(table, blip.DEFAULT_DATABASE))
	if monitorId != "" {
		q += " WHERE monitorId = '" + monitorId + "' ORDER BY name ASC" // @todo sanitize
	}
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	plans := []blip.Plan{}
	for rows.Next() {
		var plan blip.Plan
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
