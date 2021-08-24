package server

import (
	"encoding/json"
	"net/http"

	"github.com/square/blip"
	"github.com/square/blip/status"
)

type API struct {
	addr string
	*http.Server
}

func NewAPI(cfg blip.ConfigAPI) *API {
	return &API{
		addr: cfg.Bind,
	}
}

func (api *API) Run() error {
	http.HandleFunc("/status", api.status)
	return http.ListenAndServe(api.addr, nil)
}

func (api *API) status(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(status.Report(true, "*"))
}
