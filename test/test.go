// Package test provides helper functions for tests.
package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/cashapp/blip"
	"github.com/cashapp/blip/plan"
)

// blip.Confg* bool are *bool, so test.True is a convenience var
var t = true
var True = &t

var Headers = map[string]string{}

// MakeHTTPRequest is a helper function for making an http request. The
// response body of the http request is unmarshalled into the struct pointed to
// by the respStruct argument (if it's not nil). The status code of the
// response is returned.
func MakeHTTPRequest(httpVerb, url string, payload []byte, respStruct interface{}) (int, error) {
	var statusCode int

	// Make the http request.
	req, err := http.NewRequest(httpVerb, url, bytes.NewReader(payload))
	if err != nil {
		return statusCode, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range Headers {
		req.Header.Set(k, v)
	}
	res, err := (http.DefaultClient).Do(req)
	if err != nil {
		return statusCode, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)

	// Decode response into respSruct
	if respStruct != nil && len(body) > 0 {
		if err := json.Unmarshal(body, &respStruct); err != nil {
			return statusCode, fmt.Errorf("Can't decode response body: %s: %s", err, string(body))
		}
	}

	statusCode = res.StatusCode

	return statusCode, nil
}

// ReadPlan reads a plan file and returns its blip.Plan data struct. If a
// file name is not given, it reads the default plan: test/plans/default.yaml.
func ReadPlan(t *testing.T, file string) blip.Plan {
	if file == "" {
		file = "../../test/plans/default.yaml"
	}
	plan, err := plan.ReadPlanFile(file)
	if err != nil {
		t.Fatal(err)
	}
	return plan
}
