package percona_test

import (
	"testing"

	"github.com/cashapp/blip/metrics/percona"
)

func TestPercentile(t *testing.T) {
	p := [13]float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.75, 0.8, 0.9, 0.95, 0.99, 0.999}
	expectedResults := [13]float64{10, 20, 20, 20, 20, 30, 30, 30, 40, 40, 40, 40, 40}

	h := percona.QRTHistogram{
		percona.QRTBucket{},
		percona.QRTBucket{Time: 20, Count: 5, Total: 100},
		percona.QRTBucket{Time: 10, Count: 2, Total: 20},
		percona.QRTBucket{Time: 30, Count: 4, Total: 120},
		percona.QRTBucket{Time: 40, Count: 3, Total: 120},
	}

	h.Sort()

	for i, x := range p {
		result := h.Percentile(x)
		if result != expectedResults[i] {
			t.Errorf("For Percentile: %v\tExpected: %v\tGot: %v\n", x, expectedResults[i], result)
		}
	}
}
