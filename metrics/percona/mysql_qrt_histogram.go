package percona

import (
	"sort"
)

// QRTBucket : https://www.percona.com/doc/percona-server/5.6/diagnostics/response_time_distribution.html
// Represents a row from information_schema.Query_Response_Time
type QRTBucket struct {
	Time  float64
	Count uint64
	Total float64
}

// NewQRTBucket Public way to return a QRT bucket to be appended to a Histogram
func NewQRTBucket(time float64, count uint64, total float64) QRTBucket {
	return QRTBucket{
		Time:  time,
		Count: count,
		Total: total,
	}
}

// QRTHistogram represents a histogram containing MySQLQRTBuckets. Where each bucket is a bin.
type QRTHistogram struct {
	buckets []QRTBucket
	total   uint64
}

func NewQRTHistogram(buckets []QRTBucket) QRTHistogram {
	sort.Slice(buckets, func(i, j int) bool {
		return buckets[i].Time < buckets[j].Time
	})

	var total uint64

	for _, v := range buckets {
		total += v.Count
	}

	return QRTHistogram{
		buckets: buckets,
		total:   total,
	}

}

// Percentile for QRTHistogram
// p should be p/100 where p is requested percentile (example: 0.10 for 10th percentile)
// Percentile is defined as the weighted of the percentiles of
// the lowest bin that is greater than the requested percentile rank
func (h QRTHistogram) Percentile(p float64) float64 {
	var pRank float64
	var curRank uint64

	// Rank = N * P
	// N is sample size, which is sum of all counts from all the buckets
	pRank = float64(h.total) * p

	// Find the bucket where our nearest Rank lies, then take the average qrt of that bucket
	for i := range h.buckets {
		// as each of our bucket can have >= 1 data points (queries), we have to move the curRank by v.Count in each iteration
		curRank += h.buckets[i].Count

		if float64(curRank) >= pRank {
			// we have found the bucket where our target pRank lies
			// we take the average qrt of this bucket with (Total Time / Number of Queries) to find target percentile
			return h.buckets[i].Total / float64(h.buckets[i].Count)
		}
	}

	return float64(0)
}
