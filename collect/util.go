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

func CleanObjectName(o string) string {
	o = strings.ReplaceAll(o, ";", "")
	o = strings.ReplaceAll(o, "`", "")
	return strings.TrimSpace(o) // must be last in case Replace make space
}

func ObjectList(csv string, quoteChar string) []string {
	objs := strings.Split(csv, ",")
	for i := range objs {
		objs[i] = quoteChar + CleanObjectName(objs[i]) + quoteChar
	}
	return objs
}

func INList(objs []string, quoteChar string) string {
	if len(objs) == 0 {
		return ""
	}
	in := quoteChar + CleanObjectName(objs[0]) + quoteChar
	for i := range objs[1:] {
		in += "," + quoteChar + CleanObjectName(objs[i]) + quoteChar
	}
	return in
}
