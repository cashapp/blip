package collect

import (
	"strconv"
	"strings"
	"time"
)

// ToFloat64 converts string to float64, if possible.
func Float64(s string) (float64, bool) {
	f, err := strconv.ParseFloat(s, 64)
	if err == nil {
		return f, true
	}

	switch s {
	case "ON", "YES", "Yes":
		return 1, true
	case "OFF", "NO", "No", "DISABLED":
		return 0, true
	case "Connecting":
		return 0, true
	}

	if ts, err := time.Parse("Jan 02 15:04:05 2006 MST", s); err == nil {
		return float64(ts.Unix()), true
	}
	if ts, err := time.Parse("2006-01-02 15:04:05", s); err == nil {
		return float64(ts.Unix()), true
	}

	return 0, false // failed
}

func ObjectList(csv string, quoteChar string) []string {
	objs := strings.Split(csv, ",")
	for i := range objs {
		o := strings.ReplaceAll(objs[i], ";", "")
		o = strings.ReplaceAll(o, "`", "")
		o = strings.TrimSpace(o) // must be last in case Replace make space
		objs[i] = quoteChar + o + quoteChar
	}
	return objs
}
