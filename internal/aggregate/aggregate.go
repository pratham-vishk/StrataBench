package aggregate

import (
	"math"

	"github.com/pratham-vishk/stratabench/internal/schema"
)

// Results combines per-client measurements into cluster-wide totals.
func Results(runs []schema.Results) schema.Results {
	if len(runs) == 0 {
		return schema.Results{}
	}
	if len(runs) == 1 {
		return runs[0]
	}

	out := schema.Results{}
	for _, r := range runs {
		out.IOPS += r.IOPS
		out.ReadIOPS += r.ReadIOPS
		out.WriteIOPS += r.WriteIOPS
		out.ThroughputMBps += r.ThroughputMBps
		out.OpsPerSec += r.OpsPerSec
		out.TotalBytesRead += r.TotalBytesRead
		out.TotalBytesWritten += r.TotalBytesWritten
		out.TotalOperations += r.TotalOperations
		out.CPUPercent += r.CPUPercent

		out.LatencyUS.P50 = math.Max(out.LatencyUS.P50, r.LatencyUS.P50)
		out.LatencyUS.P75 = math.Max(out.LatencyUS.P75, r.LatencyUS.P75)
		out.LatencyUS.P90 = math.Max(out.LatencyUS.P90, r.LatencyUS.P90)
		out.LatencyUS.P95 = math.Max(out.LatencyUS.P95, r.LatencyUS.P95)
		out.LatencyUS.P99 = math.Max(out.LatencyUS.P99, r.LatencyUS.P99)
		out.LatencyUS.P999 = math.Max(out.LatencyUS.P999, r.LatencyUS.P999)
		if out.LatencyUS.Min == 0 || (r.LatencyUS.Min > 0 && r.LatencyUS.Min < out.LatencyUS.Min) {
			out.LatencyUS.Min = r.LatencyUS.Min
		}
		out.LatencyUS.Max = math.Max(out.LatencyUS.Max, r.LatencyUS.Max)
		out.LatencyUS.Mean += r.LatencyUS.Mean
	}
	n := float64(len(runs))
	out.LatencyUS.Mean /= n
	out.CPUPercent /= n
	out.Intervals = Intervals(runs)
	out.Totals = mergeTotals(runs)
	return out
}

func mergeTotals(runs []schema.Results) schema.TotalStats {
	var t schema.TotalStats
	for _, r := range runs {
		t.TotalMB += r.Totals.TotalMB
		t.TotalRecords += r.Totals.TotalRecords
		t.WriteRequestMB += r.Totals.WriteRequestMB
		t.WriteRequestRecords += r.Totals.WriteRequestRecords
		t.ReadRequestMB += r.Totals.ReadRequestMB
		t.ReadRequestRecords += r.Totals.ReadRequestRecords
		t.WriteTimeoutEvents += r.Totals.WriteTimeoutEvents
		t.ReadTimeoutEvents += r.Totals.ReadTimeoutEvents
	}
	return t
}
