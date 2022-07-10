// Copyright 2022 Block, Inc.

package server_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	//"github.com/stretchr/testify/assert"

	"github.com/cashapp/blip"
	"github.com/cashapp/blip/aws"
	"github.com/cashapp/blip/dbconn"
	"github.com/cashapp/blip/monitor"
	"github.com/cashapp/blip/plan"
	"github.com/cashapp/blip/proto"
	"github.com/cashapp/blip/server"
	"github.com/cashapp/blip/test"
	"github.com/cashapp/blip/test/mock"
)

type testAPI struct {
	cfg blip.Config
	api *server.API
	ts  *httptest.Server
	url string
}

func setup(t *testing.T) testAPI {
	cfg := blip.DefaultConfig()
	ml := monitor.NewLoader(monitor.LoaderArgs{
		Config: cfg,
		Factories: blip.Factories{
			DbConn: dbconn.NewConnFactory(nil, nil),
		},
		PlanLoader: plan.NewLoader(nil),
		RDSLoader:  aws.RDSLoader{ClientFactory: mock.RDSClientFactory{}},
	})
	api := server.NewAPI(cfg.API, ml)

	ts := httptest.NewServer(api)

	u, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatal(err)
	}

	return testAPI{
		cfg: cfg,
		api: api,
		ts:  ts,
		url: fmt.Sprintf("http://%s", u.Host),
	}
}

func TestAPIStatusGet(t *testing.T) {
	server := setup(t)
	defer server.ts.Close()

	var gotStatus proto.Status
	url := server.url + "/status"
	statusCode, err := test.MakeHTTPRequest("GET", url, nil, &gotStatus)
	if err != nil {
		t.Fatal(err)
	}
	if statusCode != http.StatusOK {
		t.Errorf("got HTTP status = %d, expected %d", statusCode, http.StatusOK)
	}

	if gotStatus.Version != blip.VERSION {
		t.Errorf("got Status.Version %s, expected %s", gotStatus.Version, blip.VERSION)
	}
}
