// Copyright 2022 Block, Inc.

package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/cashapp/blip"
	"github.com/cashapp/blip/metrics"
	"github.com/cashapp/blip/monitor"
	"github.com/cashapp/blip/proto"
	"github.com/cashapp/blip/sink"
	"github.com/cashapp/blip/status"
)

type API struct {
	monitorLoader *monitor.Loader
	// --
	addr       string
	httpServer *http.Server
	startTime  time.Time
}

func NewAPI(cfg blip.ConfigAPI, ml *monitor.Loader) *API {
	api := &API{
		monitorLoader: ml,
		addr:          cfg.Bind,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/version", api.version)
	mux.HandleFunc("/status", api.status)
	mux.HandleFunc("/status/monitors", api.statusMonitors)
	mux.HandleFunc("/status/monitor/internal", api.statusMonitorInternal)
	mux.HandleFunc("/registered", api.registered)
	mux.HandleFunc("/monitors/stop", api.monitorsStop)
	mux.HandleFunc("/monitors/restart", api.monitorsRestart)

	api.httpServer = &http.Server{
		Addr:    cfg.Bind,
		Handler: mux,
	}

	return api
}

// ServeHTTP allows the API to statisfy the http.HandlerFunc interface.
func (api *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	api.httpServer.Handler.ServeHTTP(w, r)
}

func (api *API) Run() error {
	api.startTime = time.Now()
	return api.httpServer.ListenAndServe()
}

func (api *API) version(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(blip.VERSION))
}

func (api *API) status(w http.ResponseWriter, r *http.Request) {
	status := proto.Status{
		Started:      api.startTime.UTC().Format(time.RFC3339),
		Uptime:       int64(time.Now().Sub(api.startTime).Seconds()),
		Version:      blip.VERSION,
		MonitorCount: api.monitorLoader.Count(),
		Internal:     status.ReportBlip(),
	}
	json.NewEncoder(w).Encode(status)
}

func (api *API) statusMonitors(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(status.ReportMonitors("*"))
}

func (api *API) statusMonitorInternal(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if len(q) == 0 {
		http.Error(w, "missing URL query: ?id=monitorId", http.StatusBadRequest)
		return
	}
	vals, ok := q["id"]
	if !ok {
		http.Error(w, "missing id param in URL query: ?id=monitorId", http.StatusBadRequest)
		return
	}
	if len(vals) == 0 {
		http.Error(w, "id param has no value", http.StatusBadRequest)
		return
	}
	blip.Debug("%v", vals)
	mon := api.monitorLoader.Monitor(vals[0])
	if mon == nil {
		http.Error(w, fmt.Sprintf("monitorId %s not loaded", vals[0]), http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(mon.Status())
}

func (api *API) registered(w http.ResponseWriter, r *http.Request) {
	reg := proto.Registered{
		Collectors: metrics.List(),
		Sinks:      sink.List(),
	}
	json.NewEncoder(w).Encode(reg)
}

func (api *API) monitorsStop(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if len(q) == 0 {
		http.Error(w, "missing URL query: ?id=monitorId", http.StatusBadRequest)
		return
	}
	vals, ok := q["id"]
	if !ok {
		http.Error(w, "missing id param in URL query: ?id=monitorId", http.StatusBadRequest)
		return
	}
	if len(vals) == 0 {
		http.Error(w, "id param has no value", http.StatusBadRequest)
		return
	}
	monitorId := vals[0]
	blip.Debug("unload %s", monitorId)
	api.monitorLoader.Unload(monitorId)
	w.WriteHeader(http.StatusOK)
}

func (api *API) monitorsRestart(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if len(q) == 0 {
		http.Error(w, "missing URL query: ?id=monitorId", http.StatusBadRequest)
		return
	}
	vals, ok := q["id"]
	if !ok {
		http.Error(w, "missing id param in URL query: ?id=monitorId", http.StatusBadRequest)
		return
	}
	if len(vals) == 0 {
		http.Error(w, "id param has no value", http.StatusBadRequest)
		return
	}
	monitorId := vals[0]
	blip.Debug("restart %s", monitorId)
	mon := api.monitorLoader.Monitor(monitorId)
	if mon == nil {
		http.Error(w, fmt.Sprintf("monitorId %s not loaded", monitorId), http.StatusNotFound)
		return
	}
	mon.Restart()
	w.WriteHeader(http.StatusOK)
}
