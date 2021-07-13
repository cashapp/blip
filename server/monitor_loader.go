package server

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sync"

	"github.com/square/blip"
	"github.com/square/blip/event"
)

type MonitorChanges struct {
	Added   []*DbMon
	Removed []*DbMon
	Changed []*DbMon
}

type MonitorLoader struct {
	cfg     blip.Config
	plugin  func(blip.Config) ([]blip.ConfigMonitor, error)
	factory Factories
	dbmon   map[string]*DbMon // keyed on monitorId
	source  string
	*sync.Mutex
}

func NewMonitorLoader(cfg blip.Config, plugins Plugins, factory Factories) *MonitorLoader {
	return &MonitorLoader{
		cfg:     cfg,
		plugin:  plugins.LoadMonitors,
		factory: factory,
		// --
		dbmon: map[string]*DbMon{},
		Mutex: &sync.Mutex{},
	}
}

func (ml *MonitorLoader) Monitors() []*DbMon {
	ml.Lock()
	defer ml.Unlock()
	monitors := make([]*DbMon, len(ml.dbmon))
	i := 0
	for _, dbmon := range ml.dbmon {
		monitors[i] = dbmon
		i++
	}
	return monitors
}

func (ml *MonitorLoader) Load(ctx context.Context) (MonitorChanges, error) {
	event.Send(event.MONITOR_LOADER_LOADING)

	ch := MonitorChanges{
		Added:   []*DbMon{},
		Removed: []*DbMon{},
		Changed: []*DbMon{},
	}
	defer func() {
		event.Sendf(event.BOOT_MONITORS_LOADED, "added: %d removed: %d changed: %d",
			len(ch.Added), len(ch.Removed), len(ch.Changed))
	}()

	dbmon := map[string]*DbMon{}

	if ml.plugin != nil {
		blip.Debug("call plugin.LoadMonitors")
		monitors, err := ml.plugin(ml.cfg)
		if err != nil {
			return ch, err
		}
		ml.merge(monitors, dbmon)
	} else {
		// First, monitors from the config file
		if len(ml.cfg.Monitors) != 0 {
			ml.merge(ml.cfg.Monitors, dbmon)
		}

		// Second, monitors from the monitor files
		monitors, err := ml.loadFiles(ctx)
		if err != nil {
			return ch, err
		}
		ml.merge(monitors, dbmon)

		// Third, monitors from the AWS RDS API
		monitors, err = ml.loadAWS(ctx)
		if err != nil {
			return ch, err
		}
		ml.merge(monitors, dbmon)

		// Last, local monitors auto-detected
		monitors, err = ml.loadLocal(ctx)
		if err != nil {
			return ch, err
		}
		ml.merge(monitors, dbmon)
	}

	ml.Lock()
	defer ml.Unlock()

	// Don't change if >= StopLoss% of monitors are lost/don't load
	if ml.cfg.MonitorLoader.StopLoss > 0 {
		nBefore := float64(len(ml.dbmon))
		nNow := float64(len(ml.dbmon))
		if nNow < nBefore && (nBefore-nNow)/nBefore >= ml.cfg.MonitorLoader.StopLoss {
			return ch, fmt.Errorf("stop-loss") // @tody
		}
	}

	for monitorId, newDbmon := range dbmon {
		oldDbmon := ml.dbmon[monitorId]
		if oldDbmon == nil {
			ch.Added = append(ch.Added, newDbmon)
			ml.dbmon[monitorId] = newDbmon
		} else if hash(newDbmon) != hash(oldDbmon) {
			go oldDbmon.Stop()
			ch.Changed = append(ch.Changed, oldDbmon)
			ml.dbmon[monitorId] = newDbmon
		} else {
			// existing dbmon, no change
		}
	}
	for monitorId, oldDbmon := range ml.dbmon {
		if _, ok := dbmon[monitorId]; !ok {
			go oldDbmon.Stop()
			ch.Removed = append(ch.Removed, oldDbmon)
			delete(ml.dbmon, monitorId)
		}
	}

	return ch, nil
}

func (ml *MonitorLoader) merge(monitors []blip.ConfigMonitor, dbmon map[string]*DbMon) {
	for _, mon := range monitors {
		dbmon[mon.Id] = ml.factory.MakeDbMon.Make(mon)
	}
}

func hash(v interface{}) [sha256.Size]byte {
	return sha256.Sum256([]byte(fmt.Sprintf("%v", v)))
}

func (ml *MonitorLoader) loadFiles(ctx context.Context) ([]blip.ConfigMonitor, error) {
	if len(ml.cfg.MonitorLoader.Files) == 0 {
		return nil, nil
	}
	return nil, nil
}

func (ml *MonitorLoader) loadAWS(ctx context.Context) ([]blip.ConfigMonitor, error) {
	if !ml.cfg.MonitorLoader.AWS.Auto {
		return nil, nil
	}
	// @todo auto-detect AWS stuff
	return nil, nil
}

func (ml *MonitorLoader) loadLocal(ctx context.Context) ([]blip.ConfigMonitor, error) {
	if !ml.cfg.MonitorLoader.Local.Auto {
		return nil, nil
	}
	// @todo auto-detect local instances
	monitors := []blip.ConfigMonitor{
		blip.DefaultConfigMonitor(),
	}
	return monitors, nil
}
