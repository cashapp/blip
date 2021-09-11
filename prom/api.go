package prom

import (
	"io"
	"net/http"

	"github.com/square/blip"
)

// API listens on a unique port and responds to GET /metrics for one exporter.
type API struct {
	addr      string
	monitorId string
	exp       *Exporter
	// --
	*http.Server
}

func NewAPI(addr string, monitorId string, exp *Exporter) *API {
	return &API{
		addr: addr,
		exp:  exp,
	}
}

func (api *API) Run() error {
	blip.Debug("%s: prom addr %s", api.monitorId, api.addr)
	http.HandleFunc("/metrics", api.metrics)
	return http.ListenAndServe(api.addr, nil)
}

func (api *API) metrics(w http.ResponseWriter, r *http.Request) {
	expo, err := api.exp.Scrape()
	if err != nil {
		blip.Debug(err.Error())
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, expo)
}
