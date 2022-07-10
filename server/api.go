// Copyright 2022 Block, Inc.

package server

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strings"
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
	mux.HandleFunc("/monitors/start", api.monitorsStart)

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
	monitorId, ok := monitorId(w, r)
	if !ok {
		return // monitorId() wrote error response
	}
	mon := api.monitorLoader.Monitor(monitorId)
	if mon == nil {
		errMsg := html.EscapeString(fmt.Sprintf("monitorId %s not loaded", monitorId))
		http.Error(w, errMsg, http.StatusNotFound)
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
	monitorId, ok := monitorId(w, r)
	if !ok {
		return // monitorId() wrote error response
	}
	blip.Debug("stop %s", monitorId)
	mon := api.monitorLoader.Monitor(monitorId)
	if mon == nil {
		errMsg := html.EscapeString(fmt.Sprintf("monitorId %s not loaded", monitorId))
		http.Error(w, errMsg, http.StatusNotFound)
		return
	}
	mon.Stop()
	w.WriteHeader(http.StatusOK)
}

func (api *API) monitorsStart(w http.ResponseWriter, r *http.Request) {
	monitorId, ok := monitorId(w, r)
	if !ok {
		return // monitorId() wrote error response
	}
	blip.Debug("start %s", monitorId)
	mon := api.monitorLoader.Monitor(monitorId)
	if mon == nil {
		errMsg := html.EscapeString(fmt.Sprintf("monitorId %s not loaded", monitorId))
		http.Error(w, errMsg, http.StatusNotFound)
		return
	}
	if err := mon.Start(); err != nil {
		errMsg := html.EscapeString(fmt.Sprintf("monitorId %s failed to start: %s", monitorId, err))
		http.Error(w, errMsg, http.StatusConflict)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// --------------------------------------------------------------------------

// monitorId returns the monitor ID from URL query param '?id=monitorId' if set.
// Else it returns an empty string and false. The caller must return without
// doing anything else on false because this func writes the HTTP error response
// on false.
func monitorId(w http.ResponseWriter, r *http.Request) (string, bool) {
	q := r.URL.Query()
	if len(q) == 0 {
		http.Error(w, "missing URL query: ?id=monitorId", http.StatusBadRequest)
		return "", false
	}
	vals, ok := q["id"]
	if !ok {
		http.Error(w, "missing id param in URL query: ?id=monitorId", http.StatusBadRequest)
		return "", false
	}
	if len(vals) == 0 {
		http.Error(w, "id param has no value, expected monitor ID", http.StatusBadRequest)
		return "", false
	}

	// Avoid code scanning alert "Log entries created from user input"
	monitorId := strings.Replace(vals[0], "\n", "", -1)
	monitorId = strings.Replace(monitorId, "\r", "", -1)

	return monitorId, true
}
