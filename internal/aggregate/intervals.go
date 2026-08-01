package aggregate

import (
	"math"

	"github.com/pratham-vishk/stratabench/internal/schema"
)

// Intervals merges time-bucket samples across nodes (sum throughput, max tail latency).
func Intervals(runs []schema.Results) []schema.IntervalSample {
	if len(runs) == 0 {
		return nil
	}
	if len(runs) == 1 {
		return runs[0].Intervals
	}

	maxBuckets := 0
	for _, r := range runs {
		if len(r.Intervals) > maxBuckets {
			maxBuckets = len(r.Intervals)
		}
	}
	if maxBuckets == 0 {
		return nil
	}

	out := make([]schema.IntervalSample, maxBuckets)
	for i := 0; i < maxBuckets; i++ {
		var merged schema.IntervalSample
		merged.Seq = i + 1
		merged.Percentiles = map[string]float64{}
		nAvg := 0

		for _, r := range runs {
			if i >= len(r.Intervals) {
				continue
			}
			iv := r.Intervals[i]
			if merged.Seq == i+1 && iv.Seq > 0 {
				merged.Seq = iv.Seq
			}
			if merged.Timestamp.IsZero() && !iv.Timestamp.IsZero() {
				merged.Timestamp = iv.Timestamp
			}
			if merged.ElapsedSec == 0 && iv.ElapsedSec > 0 {
				merged.ElapsedSec = iv.ElapsedSec
			}

			merged.IOPS += iv.IOPS
			merged.ReadIOPS += iv.ReadIOPS
			merged.WriteIOPS += iv.WriteIOPS
			merged.ThroughputMBps += iv.ThroughputMBps
			merged.ReadMBps += iv.ReadMBps
			merged.WriteMBps += iv.WriteMBps
			merged.WriteTimeoutEvents += iv.WriteTimeoutEvents
			merged.ReadTimeoutEvents += iv.ReadTimeoutEvents
			merged.WriteTimeoutPerSec += iv.WriteTimeoutPerSec
			merged.ReadTimeoutPerSec += iv.ReadTimeoutPerSec

			if iv.MinLatencyUS > 0 && (merged.MinLatencyUS == 0 || iv.MinLatencyUS < merged.MinLatencyUS) {
				merged.MinLatencyUS = iv.MinLatencyUS
			}
			merged.MaxLatencyUS = math.Max(merged.MaxLatencyUS, iv.MaxLatencyUS)
			if iv.AvgLatencyUS > 0 {
				merged.AvgLatencyUS += iv.AvgLatencyUS
				nAvg++
			}
			for k, v := range iv.Percentiles {
				merged.Percentiles[k] = math.Max(merged.Percentiles[k], v)
			}
		}
		if nAvg > 0 {
			merged.AvgLatencyUS /= float64(nAvg)
		}
		if len(merged.Percentiles) == 0 {
			merged.Percentiles = nil
		}
		out[i] = merged
	}
	return out
}
