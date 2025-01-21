// Copyright 2024 Block, Inc.

package sqlutil

import (
	"testing"
)

func TestFloat64(t *testing.T) {
	type test struct {
		s  string
		f  float64
		ok bool
	}
	floatTests := []test{
		{s: "0.0", f: 0, ok: true},
		{s: "1.0", f: 1.0, ok: true},
		{s: "1", f: 1.0, ok: true},
	}
	for _, ft := range floatTests {
		f, ok := Float64(ft.s)
		if f != ft.f {
			t.Errorf("Float64(\"%s\"): got %f, expected 1.0", ft.s, ft.f)
		}
		if ok != ft.ok {
			t.Errorf("Float64(\"%s\"): conversion ok=%t, expected ok=%t", ft.s, ok, ft.ok)
		}
	}
}

func TestInList(t *testing.T) {
	values := []string{"bleh;", "`something`"}
	result := INList(values, "'")
	expected := "'bleh','something'"

	if expected != result {
		t.Errorf("INList(%v, \"'\"): got %s, expected %s", values, result, expected)
	}
}
