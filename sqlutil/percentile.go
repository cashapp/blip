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
	percentileStr = strings.TrimSpace(percentileStr)
	f, err := strconv.ParseFloat(percentileStr, 64)
	if err != nil {
		return 0, fmt.Errorf("percentile value could not be parsed into a number: %s ", percentileStr)
	}

	var percentile float64
	if f < 1 {
		// percentile of the form 0.999 (P99.9)
		percentile = f
	} else if f >= 1 && f <= 100 {
		// percentile of the form 99.9 (P99.9)
		percentile = f / 100.0
	} else {
		// percentile of the form 999 (P99.9)
		// To find the percentage as decimal, we want to convert this number into a float with no significant digits before decimal.
		// we can do this with: f / (10 ^ (number of digits))
		percentile = f / math.Pow10(len(percentileStr))
	}

	return percentile, nil
}

// FormatPercentile formats a percentile into the form pNNN where NNN is the percentile upto 1 decimal point
func FormatPercentile(f float64) string {
	percentile := f * 100
	metaKey := fmt.Sprintf("%.1f", percentile)
	metaKey = strings.Trim(metaKey, "0")
	metaKey = strings.ReplaceAll(metaKey, ".", "")
	metaKey = "p" + metaKey

	return metaKey
}
