// Copyright 2022 Block, Inc.

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
// it returns the percentile value and the real percentile used
func (h QRTHistogram) Percentile(p float64) (value float64, actualPercentile float64) {
	var pCount float64
	var curCount uint64

	// Rank = N * P
	// N is sample size, which is sum of all counts from all the buckets
	// as we are using Histogram with buckets these are not actual Ranks
	pCount = float64(h.total) * p

	// Find the bucket where our nearest count lies, then take the average qrt of that bucket
	for i := range h.buckets {
		// as each of our bucket can have >= 1 data points (queries), we have to move the curCount by v.Count in each iteration
		curCount += h.buckets[i].Count

		if float64(curCount) >= pCount {
			// we have found the bucket where our target pCount lies
			// we take the average qrt of this bucket with (Total Time / Number of Queries) to find target percentile
			actualPercentile = float64(curCount) / float64(h.total)
			value = h.buckets[i].Total / float64(h.buckets[i].Count)

			return
		}
	}

	return
}
