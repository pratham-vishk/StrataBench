package compare

import (
	"fmt"
	"math"

	"github.com/pratham-vishk/stratabench/internal/metrics"
	"github.com/pratham-vishk/stratabench/internal/provenance"
	"github.com/pratham-vishk/stratabench/internal/schema"
)

// MetricRow is one compared scalar metric.
type MetricRow struct {
	Name     string  `json:"name"`
	Base     float64 `json:"base"`
	Head     float64 `json:"head"`
	DeltaPct float64 `json:"delta_pct"`
	Better   string  `json:"better"` // improved | regressed | neutral
}

// DiffResult is a structured A vs B comparison (base → head).
type DiffResult struct {
	Profile    string      `json:"profile"`
	Target     string      `json:"target"`
	BaseRunID  string      `json:"base_run_id"`
	HeadRunID  string      `json:"head_run_id"`
	BaseLabel  string      `json:"base_label"`
	HeadLabel  string      `json:"head_label"`
	Metrics    []MetricRow `json:"metrics"`
	Verdict    string      `json:"verdict"`
	Summary    string      `json:"summary"`
	Regressed  bool        `json:"regressed"`
	Improved   bool        `json:"improved"`
}

// Diff compares base (A) and head (B) runs. Positive delta on IOPS means head is faster.
func Diff(base, head *schema.RunResult) DiffResult {
	d := DiffResult{
		Profile:   base.Profile,
		Target:    base.Target.Device,
		BaseRunID: base.RunID,
		HeadRunID: head.RunID,
		BaseLabel: runLabel(base),
		HeadLabel: runLabel(head),
	}
	d.Metrics = append(d.Metrics,
		row("IOPS", base.Results.IOPS, head.Results.IOPS, true),
		row("Throughput (MB/s)", base.Results.ThroughputMBps, head.Results.ThroughputMBps, true),
		row("Read IOPS", base.Results.ReadIOPS, head.Results.ReadIOPS, true),
		row("Write IOPS", base.Results.WriteIOPS, head.Results.WriteIOPS, true),
		row("Latency mean (µs)", base.Results.LatencyUS.Mean, head.Results.LatencyUS.Mean, false),
		row("Latency p50 (µs)", base.Results.LatencyUS.P50, head.Results.LatencyUS.P50, false),
		row("Latency p99 (µs)", base.Results.LatencyUS.P99, head.Results.LatencyUS.P99, false),
		row("Latency p99.9 (µs)", base.Results.LatencyUS.P999, head.Results.LatencyUS.P999, false),
		row("Latency max (µs)", base.Results.LatencyUS.Max, head.Results.LatencyUS.Max, false),
	)
	for _, pl := range metrics.StandardPercentileLabels {
		bv, bok := base.Results.Percentiles[pl]
		hv, hok := head.Results.Percentiles[pl]
		if !bok && !hok {
			continue
		}
		if !bok {
			bv = lookupLatency(base.Results.LatencyUS, pl)
		}
		if !hok {
			hv = lookupLatency(head.Results.LatencyUS, pl)
		}
		d.Metrics = append(d.Metrics, row(pl+" (µs)", bv, hv, false))
	}
	d.Verdict, d.Summary, d.Improved, d.Regressed = verdict(d.Metrics)
	return d
}

func runLabel(r *schema.RunResult) string {
	if r.Provenance.GitBranch != "" {
		return provenance.Label(r.Provenance)
	}
	if len(r.RunID) >= 8 {
		return r.RunID[:8]
	}
	return r.RunID
}

func row(name string, base, head float64, higherIsBetter bool) MetricRow {
	delta := pctDelta(base, head)
	better := "neutral"
	if math.Abs(delta) >= 0.5 {
		improved := delta > 0
		if !higherIsBetter {
			improved = delta < 0
		}
		if improved {
			better = "improved"
		} else {
			better = "regressed"
		}
	}
	return MetricRow{Name: name, Base: base, Head: head, DeltaPct: delta, Better: better}
}

func pctDelta(a, b float64) float64 {
	if a == 0 {
		return 0
	}
	return ((b - a) / a) * 100
}

func lookupLatency(lat schema.LatencyUS, pl string) float64 {
	switch pl {
	case "p50":
		return lat.P50
	case "p75":
		return lat.P75
	case "p90":
		return lat.P90
	case "p95":
		return lat.P95
	case "p99":
		return lat.P99
	case "p99.9":
		return lat.P999
	case "p99.99":
		return lat.P9999
	default:
		return 0
	}
}

func verdict(metrics []MetricRow) (verdict, summary string, improved, regressed bool) {
	var iopsDelta, p99Delta float64
	improveCount, regressCount := 0, 0
	for _, m := range metrics {
		switch m.Better {
		case "improved":
			improveCount++
		case "regressed":
			regressCount++
		}
		if m.Name == "IOPS" {
			iopsDelta = m.DeltaPct
		}
		if m.Name == "Latency p99 (µs)" {
			p99Delta = m.DeltaPct
		}
	}
	improved = iopsDelta > 1 && p99Delta <= 5
	regressed = iopsDelta < -5 || p99Delta > 10
	switch {
	case improved && !regressed:
		verdict = "improved"
		summary = fmt.Sprintf("Head branch looks faster: IOPS %+.1f%%, p99 %+.1f%%", iopsDelta, p99Delta)
	case regressed:
		verdict = "regressed"
		summary = fmt.Sprintf("Head branch regressed: IOPS %+.1f%%, p99 %+.1f%%", iopsDelta, p99Delta)
	default:
		verdict = "neutral"
		summary = fmt.Sprintf("Mixed impact: IOPS %+.1f%%, p99 %+.1f%% (%d improved, %d regressed metrics)",
			iopsDelta, p99Delta, improveCount, regressCount)
	}
	return verdict, summary, improved, regressed
}

// Print writes a human-readable diff to stdout.
func Print(base, head *schema.RunResult) {
	d := Diff(base, head)
	fmt.Printf("Compare runs\n")
	fmt.Printf("  Base: %s  %s  profile=%s\n", d.BaseRunID, d.BaseLabel, d.Profile)
	fmt.Printf("  Head: %s  %s  profile=%s\n", d.HeadRunID, d.HeadLabel, d.Profile)
	fmt.Printf("\nVerdict: %s — %s\n\n", d.Verdict, d.Summary)
	fmt.Printf("%-22s %12s %12s %12s %10s\n", "Metric", "Base", "Head", "Delta%", "Change")
	for _, m := range d.Metrics {
		if m.Base == 0 && m.Head == 0 {
			continue
		}
		fmt.Printf("%-22s %12.2f %12.2f %11.1f%% %10s\n", m.Name, m.Base, m.Head, m.DeltaPct, m.Better)
	}
}
