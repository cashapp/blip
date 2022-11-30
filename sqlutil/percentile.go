// Copyright 2022 Block, Inc.

package sqlutil

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// ParsePercentileStr coverts a string of percentile in different form to the percentile as decimal.
// There are 3 kind of forms, like P99.9 has form 0.999, form 99.9 or form 999. All should be parsed as 0.999
func ParsePercentileStr(percentileStr string) (float64, error) {
	s := strings.TrimLeft(strings.TrimSpace(percentileStr), "pP")
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("error parsing '%s' ('%s') as float64: %s ", percentileStr, s, err)
	}

	var percentile float64
	if f < 1 {
		// percentile of the form 0.999 (P99.9)
		percentile = f
	} else if f >= 1 && f < 100 {
		// percentile of the form 99.9 (P99.9)
		percentile = f / 100.0
	} else if f == 100.0 {
		return 100, nil
	} else {
		// percentile of the form 999 (P99.9)
		// To find the percentage as decimal, we want to convert this number into a float with no significant digits before decimal.
		// we can do this with: f / (10 ^ (number of digits))
		percentile = f / math.Pow10(len(s))
	}

	return percentile, nil
}

// FormatPercentile formats a percentile into the form pNNN where NNN is
// the percentile upt o 1 decimal point. For example, 0.99 returns "p99".
func FormatPercentile(f float64) string {
	percentile := f * 100
	metaKey := fmt.Sprintf("%.1f", percentile)
	metaKey = strings.Trim(metaKey, "0")
	metaKey = strings.ReplaceAll(metaKey, ".", "")
	metaKey = "p" + metaKey
	return metaKey
}

type P struct {
	Name  string
	Value float64
}

func PercentileMetrics(metrics []string) ([]P, error) {
	if len(metrics) == 0 {
		return nil, fmt.Errorf("no percentile metrics specified, expected at least 1 like 'p99'")
	}
	p := []P{}
	for _, s := range metrics {
		val, err := ParsePercentileStr(s)
		if err != nil {
			return nil, err
		}
		p = append(p, P{
			Name:  FormatPercentile(val),
			Value: val,
		})
	}
	return p, nil
}
