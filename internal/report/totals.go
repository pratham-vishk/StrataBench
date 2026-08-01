package report

import (
	"fmt"

	"github.com/pratham-vishk/stratabench/internal/schema"
)

func effectiveTotals(res schema.Results, durationSec int) schema.TotalStats {
	t := res.Totals
	if t.TotalRecords == 0 && res.TotalOperations > 0 {
		t.TotalRecords = res.TotalOperations
	}
	if t.TotalRecords == 0 && res.IOPS > 0 && durationSec > 0 {
		t.TotalRecords = int64(res.IOPS * float64(durationSec))
	}
	if t.TotalMB == 0 && res.ThroughputMBps > 0 && durationSec > 0 {
		t.TotalMB = res.ThroughputMBps * float64(durationSec)
	}
	if t.WriteRequestRecords == 0 && res.WriteIOPS > 0 && durationSec > 0 {
		t.WriteRequestRecords = int64(res.WriteIOPS * float64(durationSec))
	}
	if t.ReadRequestRecords == 0 && res.ReadIOPS > 0 && durationSec > 0 {
		t.ReadRequestRecords = int64(res.ReadIOPS * float64(durationSec))
	}
	if t.WriteRequestMB == 0 && res.WriteIOPS > 0 && res.ThroughputMBps > 0 && res.IOPS > 0 && durationSec > 0 {
		t.WriteRequestMB = res.ThroughputMBps * (res.WriteIOPS / res.IOPS) * float64(durationSec)
	}
	if t.ReadRequestMB == 0 && res.ReadIOPS > 0 && res.ThroughputMBps > 0 && res.IOPS > 0 && durationSec > 0 {
		t.ReadRequestMB = res.ThroughputMBps * (res.ReadIOPS / res.IOPS) * float64(durationSec)
	}
	if t.WriteTimeoutEvents == 0 || t.ReadTimeoutEvents == 0 {
		for _, iv := range res.Intervals {
			t.WriteTimeoutEvents += iv.WriteTimeoutEvents
			t.ReadTimeoutEvents += iv.ReadTimeoutEvents
		}
	}
	return t
}

func hasTotalVolume(t schema.TotalStats) bool {
	return t.TotalMB > 0 || t.TotalRecords > 0 || t.WriteRequestMB > 0 || t.ReadRequestMB > 0
}

func hasTotalTimeouts(t schema.TotalStats) bool {
	return t.WriteTimeoutEvents > 0 || t.ReadTimeoutEvents > 0
}

type totalsRow struct {
	label string
	stats schema.TotalStats
}

func collectTotalsRows(run *schema.RunResult) []totalsRow {
	dur := run.Workload.DurationSec
	if dur <= 0 {
		dur = runDurationSec(run)
	}
	var rows []totalsRow
	agg := effectiveTotals(run.Results, dur)
	if hasTotalVolume(agg) || hasTotalTimeouts(agg) {
		rows = append(rows, totalsRow{"Aggregate", agg})
	}
	for _, c := range run.Clients {
		t := effectiveTotals(c.Results, dur)
		if hasTotalVolume(t) {
			rows = append(rows, totalsRow{shortHost(c.Host), t})
		}
	}
	for _, tgt := range run.Targets {
		t := effectiveTotals(tgt.Results, dur)
		if hasTotalVolume(t) {
			rows = append(rows, totalsRow{shortHost(tgt.Target), t})
		}
	}
	return rows
}

func addTotalsCharts(b *chartBuilder, run *schema.RunResult, compare compareData) {
	// SBK T-sheet parity: Total_Throughput_*, Total_Min/Avg/Max, Total_MB, Total_RW_TimeoutEvents
	b.add("Totals — throughput", false, ChartPanel{ID: "totalThroughputMbps", Title: "Total throughput (MB/s)"},
		chartSpec{Kind: "bar", Labels: compare.labels, Datasets: []chartDataset{{Label: "MB/s", Data: compare.mbps}}})
	b.add("Totals — throughput", false, ChartPanel{ID: "totalThroughputRecords", Title: "Total throughput (records/s)"},
		chartSpec{Kind: "bar", Labels: compare.labels, Datasets: []chartDataset{{Label: "Records/s", Data: compare.iops}}})

	b.add("Totals — latency", false, ChartPanel{ID: "totalMinLatency", Title: "Total min latency"},
		chartSpec{Kind: "bar", Labels: compare.labels, Datasets: []chartDataset{{Label: "Min µs", Data: compare.minLat}}})
	b.add("Totals — latency", false, ChartPanel{ID: "totalAvgLatency", Title: "Total avg latency"},
		chartSpec{Kind: "bar", Labels: compare.labels, Datasets: []chartDataset{{Label: "Mean µs", Data: compare.avgLat}}})
	b.add("Totals — latency", false, ChartPanel{ID: "totalMaxLatency", Title: "Total max latency"},
		chartSpec{Kind: "bar", Labels: compare.labels, Datasets: []chartDataset{{Label: "Max µs", Data: compare.maxLat}}})

	rows := collectTotalsRows(run)
	if len(rows) == 0 {
		return
	}
	labels := make([]string, len(rows))
	wrMB := make([]float64, len(rows))
	rdMB := make([]float64, len(rows))
	wrPending := make([]float64, len(rows))
	rdPending := make([]float64, len(rows))
	totalMB := make([]float64, len(rows))
	wrRec := make([]float64, len(rows))
	rdRec := make([]float64, len(rows))
	totalRec := make([]float64, len(rows))
	wrTO := make([]float64, len(rows))
	rdTO := make([]float64, len(rows))
	for i, r := range rows {
		labels[i] = r.label
		wrMB[i] = r.stats.WriteRequestMB
		rdMB[i] = r.stats.ReadRequestMB
		wrPending[i] = r.stats.WritePendingMB
		rdPending[i] = r.stats.ReadPendingMB
		totalMB[i] = r.stats.TotalMB
		wrRec[i] = float64(r.stats.WriteRequestRecords)
		rdRec[i] = float64(r.stats.ReadRequestRecords)
		totalRec[i] = float64(r.stats.TotalRecords)
		wrTO[i] = float64(r.stats.WriteTimeoutEvents)
		rdTO[i] = float64(r.stats.ReadTimeoutEvents)
	}

	if hasAnyPositive(append(wrMB, append(rdMB, totalMB...)...)) {
		b.add("Totals — data volume", false, ChartPanel{ID: "totalMbBreakdown", Title: "Total data volume (MB)"},
			chartSpec{Kind: "bar", Labels: labels, Stacked: true, Datasets: []chartDataset{
				{Label: "Write request MB", Data: wrMB},
				{Label: "Read request MB", Data: rdMB},
				{Label: "Write pending MB", Data: wrPending},
				{Label: "Read pending MB", Data: rdPending},
			}})
		b.add("Totals — data volume", false, ChartPanel{ID: "totalMbTotal", Title: "Total MB transferred"},
			chartSpec{Kind: "bar", Labels: labels, Datasets: []chartDataset{{Label: "Total MB", Data: totalMB}}})
	}
	if hasAnyPositive(append(wrRec, append(rdRec, totalRec...)...)) {
		b.add("Totals — data volume", false, ChartPanel{ID: "totalRecordsBreakdown", Title: "Total records"},
			chartSpec{Kind: "bar", Labels: labels, Stacked: true, Datasets: []chartDataset{
				{Label: "Write records", Data: wrRec},
				{Label: "Read records", Data: rdRec},
			}})
		b.add("Totals — data volume", false, ChartPanel{ID: "totalRecordsTotal", Title: "Total record count"},
			chartSpec{Kind: "bar", Labels: labels, Datasets: []chartDataset{{Label: "Records", Data: totalRec}}})
	}
	if hasAnyPositive(append(wrTO, rdTO...)) {
		b.add("Totals — reliability", false, ChartPanel{ID: "totalTimeoutEvents", Title: "Total timeout events"},
			chartSpec{Kind: "bar", Labels: labels, Datasets: []chartDataset{
				{Label: "Write timeouts", Data: wrTO},
				{Label: "Read timeouts", Data: rdTO},
			}})
	}
}

func buildTotalRows(run *schema.RunResult) []KVRow {
	dur := run.Workload.DurationSec
	if dur <= 0 {
		dur = runDurationSec(run)
	}
	t := effectiveTotals(run.Results, dur)
	if !hasTotalVolume(t) && !hasTotalTimeouts(t) {
		return nil
	}
	rows := []KVRow{
		{"Total MB", fmt.Sprintf("%.2f", t.TotalMB)},
		{"Total records", fmt.Sprintf("%d", t.TotalRecords)},
		{"Write request MB", fmt.Sprintf("%.2f", t.WriteRequestMB)},
		{"Write request records", fmt.Sprintf("%d", t.WriteRequestRecords)},
		{"Read request MB", fmt.Sprintf("%.2f", t.ReadRequestMB)},
		{"Read request records", fmt.Sprintf("%d", t.ReadRequestRecords)},
	}
	if t.WritePendingMB > 0 || t.ReadPendingMB > 0 {
		rows = append(rows,
			KVRow{"Write pending MB", fmt.Sprintf("%.2f", t.WritePendingMB)},
			KVRow{"Read pending MB", fmt.Sprintf("%.2f", t.ReadPendingMB)},
		)
	}
	if t.WriteTimeoutEvents > 0 || t.ReadTimeoutEvents > 0 {
		rows = append(rows,
			KVRow{"Write timeout events", fmt.Sprintf("%d", t.WriteTimeoutEvents)},
			KVRow{"Read timeout events", fmt.Sprintf("%d", t.ReadTimeoutEvents)},
		)
	}
	if t.InvalidLatencies > 0 {
		rows = append(rows, KVRow{"Invalid latencies", fmt.Sprintf("%d", t.InvalidLatencies)})
	}
	if t.LowerDiscard > 0 || t.HigherDiscard > 0 {
		rows = append(rows,
			KVRow{"Lower discard", fmt.Sprintf("%d", t.LowerDiscard)},
			KVRow{"Higher discard", fmt.Sprintf("%d", t.HigherDiscard)},
		)
	}
	if t.SLC1 > 0 || t.SLC2 > 0 {
		rows = append(rows,
			KVRow{"SLC1", fmt.Sprintf("%d", t.SLC1)},
			KVRow{"SLC2", fmt.Sprintf("%d", t.SLC2)},
		)
	}
	return rows
}
