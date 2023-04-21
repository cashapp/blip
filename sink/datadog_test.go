package sink

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
	"github.com/cashapp/blip"
	"github.com/cashapp/blip/test/mock"
	"github.com/go-test/deep"
)

func defaultOps() map[string]string {
	return map[string]string{
		"api-key-auth": "testkey",
		"app-key-auth": "testkey",
		"api-compress": "true",
	}
}

func okHttpClient() *http.Client {
	return &http.Client{
		Transport: &mock.Transport{
			RoundTripFunc: func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
				}, nil
			},
		},
	}
}

func getBlipMetrics(valuesCount int) *blip.Metrics {
	values := make([]blip.MetricValue, 0, valuesCount)

	for i := 0; i < valuesCount; i++ {
		values = append(values, blip.MetricValue{
			Name:  fmt.Sprintf("testmetric%d", i+1),
			Value: 1.0,
			Type:  blip.GAUGE,
		})
	}

	return &blip.Metrics{
		Begin:     time.Now().Add(-1 * time.Hour),
		End:       time.Now(),
		MonitorId: "testmonitor",
		Plan:      "testplan",
		Level:     "testlevel",
		State:     "teststate",
		Values: map[string][]blip.MetricValue{
			"testdomain": values,
		},
	}
}

func getPayloadSize(r *http.Request) (int, error) {
	bodyReader, err := r.GetBody()
	if err != nil {
		return 0, err
	}

	defer bodyReader.Close()
	body, err := ioutil.ReadAll(bodyReader)
	if err != nil {
		return 0, err
	}

	return len(body), nil
}

func TestDatadogSink(t *testing.T) {
	ddSink, err := NewDatadog("testmonitor", defaultOps(), map[string]string{}, okHttpClient())

	if err != nil {
		t.Fatalf("Expected no error but got %v", err)
	}

	err = ddSink.Send(context.Background(), getBlipMetrics(10))

	if err != nil {
		t.Fatalf("Expected no error but got %v", err)
	}
}

func TestDatadogMetricsPerRequest(t *testing.T) {
	callCount := 0
	testPayloadSize := 5000
	metricCount := 100

	httpClient := &http.Client{
		Transport: &mock.Transport{
			RoundTripFunc: func(r *http.Request) (*http.Response, error) {
				callCount++

				bodySize, err := getPayloadSize(r)
				if err != nil {
					return nil, err
				}

				if bodySize > testPayloadSize {
					return &http.Response{
						StatusCode: http.StatusRequestEntityTooLarge,
					}, nil
				}

				return &http.Response{
					StatusCode: http.StatusOK,
				}, nil
			},
		},
	}

	ops := defaultOps()
	ops["api-compress"] = "false" // Turn off compression so that we get easier calculations for sizing
	ddSink, err := NewDatadog("testmonitor", ops, map[string]string{}, httpClient)
	ddSink.maxPayloadSize = testPayloadSize // Set the payload size for testing

	if err != nil {
		t.Fatalf("Expected no error but got %v", err)
	}

	err = ddSink.Send(context.Background(), getBlipMetrics(metricCount))

	if err != nil {
		t.Fatalf("Expected no error but got %v", err)
	}

	if ddSink.maxMetricsPerRequest == math.MaxInt {
		t.Error("Expected maxMetricsPerRequest to be adjusted but got MaxInt")
	}

	if callCount != 4 {
		t.Errorf("Expected 4 calls but got %d", callCount)
	}
}

func TestDatadogMetricsPerRequestWithCompression(t *testing.T) {
	callCount := 0
	testPayloadSize := 1500
	metricCount := 600

	httpClient := &http.Client{
		Transport: &mock.Transport{
			RoundTripFunc: func(r *http.Request) (*http.Response, error) {
				callCount++

				bodySize, err := getPayloadSize(r)
				if err != nil {
					return nil, err
				}

				if bodySize > testPayloadSize {
					return &http.Response{
						StatusCode: http.StatusRequestEntityTooLarge,
					}, nil
				}

				return &http.Response{
					StatusCode: http.StatusOK,
				}, nil
			},
		},
	}

	ops := defaultOps()
	ddSink, err := NewDatadog("testmonitor", ops, map[string]string{}, httpClient)
	ddSink.maxPayloadSize = testPayloadSize // Set the payload size for testing

	if err != nil {
		t.Fatalf("Expected no error but got %v", err)
	}

	err = ddSink.Send(context.Background(), getBlipMetrics(metricCount))

	if err != nil {
		t.Fatalf("Expected no error but got %v", err)
	}

	if ddSink.maxMetricsPerRequest == math.MaxInt {
		t.Error("Expected maxMetricsPerRequest to be adjusted but got MaxInt")
	}

	if callCount == 1 {
		t.Error("Expected more than 1 call but got only 1")
	}
}

func TestDatadogMetricsPerRequestMultipleFail(t *testing.T) {
	callCount := 0
	testPayloadSize := 5000
	metricCount := 500
	trCount := 0
	var collectedMetrics []string

	httpClient := &http.Client{
		Transport: &mock.Transport{
			RoundTripFunc: func(r *http.Request) (*http.Response, error) {
				callCount++

				bodySize, err := getPayloadSize(r)
				if err != nil {
					return nil, err
				}

				if bodySize > testPayloadSize {
					return &http.Response{
						StatusCode: http.StatusRequestEntityTooLarge,
					}, nil
				}

				var payload datadogV2.MetricPayload
				body, _ := r.GetBody()
				defer body.Close()
				data, _ := ioutil.ReadAll(body)
				json.Unmarshal(data, &payload)

				for _, metric := range payload.Series {
					collectedMetrics = append(collectedMetrics, metric.Metric)
				}

				return &http.Response{
					StatusCode: http.StatusOK,
				}, nil
			},
		},
	}

	ops := defaultOps()
	ops["api-compress"] = "false" // Turn off compression so that we get easier calculations for sizing
	ddSink, err := NewDatadog("testmonitor", ops, map[string]string{}, httpClient)
	ddSink.maxPayloadSize = testPayloadSize // Set the payload size for testing
	ddSink.tr = &mock.Tr{
		TranslateFunc: func(domain, metric string) string {
			trCount++
			// Once we get past a certain number of messages add a long prefix to force
			// the sink to recalculate the max metrics per request.
			if trCount > 250 {
				return fmt.Sprintf("%s.THEQUICKBROWNFOXJUMPEDOVERTHELAZYDOGMAKETHISVERYLONG%s", domain, metric)
			} else {
				return fmt.Sprintf("%s.%s", domain, metric)
			}
		},
	}

	if err != nil {
		t.Fatalf("Expected no error but got %v", err)
	}

	blipMetrics := getBlipMetrics(metricCount)
	err = ddSink.Send(context.Background(), blipMetrics)

	if err != nil {
		t.Fatalf("Expected no error but got %v", err)
	}

	if ddSink.maxMetricsPerRequest == math.MaxInt {
		t.Error("Expected maxMetricsPerRequest to be adjusted but got MaxInt")
	}

	if callCount == 1 {
		t.Error("Expected more than 1 call but only got 1.")
	}

	expectedMetrics := make([]string, 0, len(blipMetrics.Values))
	expectedCount := 0
	for _, metric := range blipMetrics.Values["testdomain"] {
		expectedCount++
		if expectedCount > 250 {
			expectedMetrics = append(expectedMetrics, fmt.Sprintf("testdomain.THEQUICKBROWNFOXJUMPEDOVERTHELAZYDOGMAKETHISVERYLONG%s", metric.Name))
		} else {
			expectedMetrics = append(expectedMetrics, fmt.Sprintf("testdomain.%s", metric.Name))
		}

	}

	if diff := deep.Equal(expectedMetrics, collectedMetrics); diff != nil {
		t.Fatal(diff)
	}
}

func TestDatadogMetricsErrorResponseFromAPI(t *testing.T) {
	errors := []string{"validation error 1", "validation error 2"}
	resp := map[string][]string{
		"errors": errors,
	}
	respJSON, err := json.Marshal(resp)
	require.NoError(t, err)

	httpClient := &http.Client{
		Transport: &mock.Transport{
			RoundTripFunc: func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusAccepted,
					Body:       ioutil.NopCloser(bytes.NewReader(respJSON)),
				}, nil
			},
		},
	}

	ddSink, err := NewDatadog("testmonitor", defaultOps(), map[string]string{}, httpClient)
	require.NoError(t, err)

	err = ddSink.Send(context.Background(), getBlipMetrics(10))

	require.Errorf(t, err, "error response from Datadog: %s", strings.Join(errors, ","))
}
