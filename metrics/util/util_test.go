package util

import (
	"math"
	"testing"
)

func TestParsePercentileStr(t *testing.T) {
	tolerance := 0.000001
	testCases := map[string]float64{
		"0.999": 0.999,
		"99.9":  0.999,
		"999":   0.999,
	}

	for percentileStr, expectedResult := range testCases {
		actualResult, err := ParsePercentileStr(percentileStr)
		if err != nil {
			t.Fatal(err)
		}
		if math.Abs(actualResult-expectedResult) > tolerance {
			t.Errorf("Expected result %f for ParsePercentileStr(%s) but got %f", expectedResult, percentileStr, actualResult-expectedResult)
		}
	}
}

func TestFormatPercentile(t *testing.T) {
	testCases := map[float64]string{
		0.99:  "p99",
		0.999: "p999",
		0.90:  "p90",
	}

	for percentile, expectedFormat := range testCases {
		actualFormat := FormatPercentile(percentile)
		if actualFormat != expectedFormat {
			t.Errorf("Expected format %s for FormatPercentile(%f) but got %s", expectedFormat, percentile, actualFormat)
		}
	}
}
