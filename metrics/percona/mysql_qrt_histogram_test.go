package percona_test

import (
	"testing"

	"github.com/cashapp/blip/metrics/percona"
)

func TestPercentile(t *testing.T) {
	buckets := []percona.QRTBucket{
		{Time: 0.000001, Count: 0, Total: 0},
		{Time: 0.000003, Count: 0, Total: 0},
		{Time: 0.000007, Count: 21, Total: 0.000005},
		{Time: 0.000015, Count: 12694, Total: 0.153511},
		{Time: 0.00003, Count: 13669, Total: 0.257439},
		{Time: 0.000061, Count: 20159, Total: 0.927535},
		{Time: 0.000122, Count: 20541, Total: 1.660305},
		{Time: 0.000244, Count: 9700, Total: 1.671614},
		{Time: 0.000488, Count: 3938, Total: 1.245692},
		{Time: 0.000976, Count: 769, Total: 0.50613},
		{Time: 0.001953, Count: 245, Total: 0.324439},
		{Time: 0.003906, Count: 126, Total: 0.335418},
		{Time: 0.007812, Count: 52, Total: 0.286619},
		{Time: 0.015625, Count: 38, Total: 0.442354},
		{Time: 0.03125, Count: 20, Total: 0.406278},
		{Time: 0.0625, Count: 7, Total: 0.287812},
		{Time: 0.125, Count: 3, Total: 0.259817},
		{Time: 0.25, Count: 2, Total: 0.287062},
		{Time: 0.5, Count: 0, Total: 0},
		{Time: 1, Count: 0, Total: 0},
		{Time: 2, Count: 0, Total: 0},
	}

	h := percona.NewQRTHistogram(buckets)

	p := [4]float64{0.95, 0.99, 0.999, 1}
	expectedResults := [4]float64{0.00031632605383443373, 0.0006581664499349804, 0.005511903846153846, 0.143531}

	for i, x := range p {
		result := h.Percentile(x)
		if result != expectedResults[i] {
			t.Errorf("For Percentile: %v\tExpected: %v\tGot: %v\n", x, expectedResults[i], result)
		}
	}
}
