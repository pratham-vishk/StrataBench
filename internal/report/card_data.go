package report

import (
	"encoding/json"
	"fmt"
	"html/template"
	"math"
	"strings"
	"time"

	"github.com/pratham-vishk/stratabench/internal/analyst"
	"github.com/pratham-vishk/stratabench/internal/baseline"
	"github.com/pratham-vishk/stratabench/internal/metrics"
	"github.com/pratham-vishk/stratabench/internal/schema"
	"github.com/pratham-vishk/stratabench/internal/version"
)

// Options configures report card generation.
type Options struct {
	Insights []analyst.Insight
	Summary  string
	Alerts   []baseline.Alert
}

// CardData is the template model for the HTML report card (Excel parity).
type CardData struct {
	Run            *schema.RunResult
	Insights       []analyst.Insight
	Summary        string
	GeneratedAt    string
	Version        string
	HonestPass     bool
	ValidationOK   bool
	IOPS           string
	Throughput     string
	P99            string
	P50            string
	P95            string
	TotalMB        string
	TotalRecords   string
	IOPSDelta      string
	IOPSDeltaClass string
	P99Delta       string
	P99DeltaClass  string
	ChartsJS       template.JS
	HasNodeCharts  bool
	HasIntervals   bool
	HasHistogram   bool
	HasReadWrite   bool
	HasTimeouts    bool
	HasTotals      bool
	BenchmarkLabel string
	EngineLabel    string
	Labels         WorkloadLabels
	OperationBadge string
	ChartGroups    []ChartGroup
	ChartCount     int
	NavSections    []NavSection
	KPIs           []KPI
	SummaryRows    []KVRow
	DurationRows   []DurationRow
	MetricRows     []KVRow
	TotalRows      []KVRow
	NodeRows       []NodeRow
	IntervalRows   []IntervalRow
	PercentileRows []PercentileRow
	PercLabels     []string
}

type NavSection struct {
	ID, Title string
}

type KPI struct {
	Label, Value, Unit, Hint string
}

type PercentileRow struct {
	Label, Latency, Count string
}

type KVRow struct{ Key, Value string }

type DurationRow struct {
	Node, Role, Target, Profile, Start, End, Duration string
}

type IntervalRow struct {
	Seq, Timestamp, IOPS, ReadIOPS, WriteIOPS, MBps, Avg, Min, Max, WTimeout, RTimeout string
}

// NodeRow is one row in the all-nodes percentile table.
type NodeRow struct {
	Label, Role, IOPS, ReadIOPS, WriteIOPS, MBps string
	Min, Mean, P50, P75, P90, P95, P99, P999, P9999, Max string
}

func buildCardData(run *schema.RunResult, opts Options) (CardData, error) {
	cd := CardData{
		Run:          run,
		Insights:     opts.Insights,
		Summary:      opts.Summary,
		GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
		Version:      version.Version,
		ValidationOK: run.Validation.Passed,
		HonestPass:   run.Validation.Passed && !run.Mock,
		IOPS:         fmt.Sprintf("%.0f", run.Results.IOPS),
		Throughput:   fmt.Sprintf("%.2f", run.Results.ThroughputMBps),
		P99:          fmt.Sprintf("%.0f", run.Results.LatencyUS.P99),
		P50:          fmt.Sprintf("%.0f", run.Results.LatencyUS.P50),
		P95:          fmt.Sprintf("%.0f", run.Results.LatencyUS.P95),
		HasIntervals: len(run.Results.Intervals) > 0,
		HasHistogram: len(run.Results.PercentileCounts) > 0,
		HasReadWrite: hasReadWrite(run),
		HasTimeouts:    intervalHasTimeouts(run.Results.Intervals),
		BenchmarkLabel: benchmarkLabel(run),
		EngineLabel:    displayEngine(run.Engine),
	}
	cd.Labels = workloadLabels(run)
	if cd.Labels.IsObject && cd.Labels.Operation != "" {
		cd.OperationBadge = strings.ToUpper(cd.Labels.Operation)
	}

	for _, a := range opts.Alerts {
		if a.Metric == "iops" {
			cd.IOPSDelta = formatDelta(a.DeltaPct, true)
			cd.IOPSDeltaClass = deltaClass(a.DeltaPct, true)
		}
		if a.Metric == "latency_p99" {
			cd.P99Delta = formatDelta(a.DeltaPct, false)
			cd.P99DeltaClass = deltaClass(a.DeltaPct, false)
		}
	}
	if cd.IOPSDelta == "" {
		cd.IOPSDelta = "—"
		cd.IOPSDeltaClass = "neutral"
	}
	if cd.P99Delta == "" {
		cd.P99Delta = "—"
		cd.P99DeltaClass = "neutral"
	}

	labels, _ := latencyPercentileSeries(run.Results.LatencyUS)
	cd.PercLabels = labels

	built := buildAllCharts(run)
	cd.ChartGroups = built.Groups
	cd.ChartCount = built.Count
	cd.HasNodeCharts = len(run.Clients)+len(run.Targets) > 0

	raw, err := json.Marshal(chartsPayload{Charts: built.Specs})
	if err != nil {
		return cd, err
	}
	cd.ChartsJS = template.JS(raw)

	for _, r := range AllNodeRows(run) {
		lat := r.Lat
		fillLatencyGaps(&lat)
		cd.NodeRows = append(cd.NodeRows, NodeRow{
			Label: r.Label, Role: r.Role,
			IOPS: fmt.Sprintf("%.0f", r.IOPS), ReadIOPS: fmt.Sprintf("%.0f", r.ReadIOPS),
			WriteIOPS: fmt.Sprintf("%.0f", r.WriteIOPS), MBps: fmt.Sprintf("%.2f", r.MBps),
			Min: fµs(lat.Min), Mean: fµs(lat.Mean), P50: fµs(lat.P50),
			P75: fµs(lat.P75), P90: fµs(lat.P90), P95: fµs(lat.P95),
			P99: fµs(lat.P99), P999: fµs(lat.P999), P9999: fµs(lat.P9999), Max: fµs(lat.Max),
		})
	}

	cd.SummaryRows = buildSummaryRows(run, cd)
	cd.DurationRows = buildDurationRows(run)
	cd.MetricRows = buildMetricRows(run)
	cd.TotalRows = buildTotalRows(run)
	cd.HasTotals = len(cd.TotalRows) > 0 && !isObjectLayer(run)
	cd.PercentileRows = buildPercentileRows(run)
	cd.IntervalRows = buildIntervalRows(run)
	cd.KPIs = buildKPIs(run, cd)
	cd.NavSections = buildNavSections(cd)

	return cd, nil
}

func buildSummaryRows(run *schema.RunResult, cd CardData) []KVRow {
	nodes := []string{}
	for _, r := range cd.NodeRows {
		nodes = append(nodes, r.Label)
	}
	val := "FAILED"
	if run.Validation.Passed {
		val = "PASSED"
	}
	rows := []KVRow{
		{"Version", version.Version},
		{"Run ID", run.RunID},
		{"Profile", run.Profile},
		{"Benchmark", benchmarkLabel(run)},
		{"Topology", run.Topology},
		{"Target", runTarget(run)},
		{"Workload", fmt.Sprintf("%s %s qd=%d threads=%d", run.Workload.Pattern, run.Workload.BlockSize, run.Workload.QueueDepth, run.Workload.Threads)},
		{"Duration", fmt.Sprintf("%ds (ramp %ds)", run.Workload.DurationSec, run.Workload.RampTimeSec)},
		{"Time unit", "microseconds (µs)"},
		{"Nodes", stringsJoin(nodes)},
		{"Validation", val},
		{"Mock", fmt.Sprintf("%v", run.Mock)},
	}
	if run.Provenance.GitBranch != "" {
		rows = append(rows,
			KVRow{"Git branch", run.Provenance.GitBranch},
			KVRow{"Git commit", run.Provenance.GitSHA},
		)
		if run.Provenance.BuildCmd != "" {
			rows = append(rows, KVRow{"Build", run.Provenance.BuildCmd})
		}
	}
	return rows
}

func stringsJoin(ss []string) string {
	if len(ss) == 0 {
		return "—"
	}
	out := ss[0]
	for i := 1; i < len(ss); i++ {
		out += ", " + ss[i]
	}
	return out
}

func runTarget(run *schema.RunResult) string {
	if run.Target.Device != "" {
		return run.Target.Device
	}
	return run.Target.Endpoint
}

func runDurationSec(run *schema.RunResult) int {
	if !run.Timestamps.StartedAt.IsZero() && !run.Timestamps.CompletedAt.IsZero() {
		return int(run.Timestamps.CompletedAt.Sub(run.Timestamps.StartedAt).Seconds())
	}
	return run.Workload.DurationSec
}

func buildDurationRows(run *schema.RunResult) []DurationRow {
	start := run.Timestamps.StartedAt.UTC().Format(time.RFC3339)
	end := run.Timestamps.CompletedAt.UTC().Format(time.RFC3339)
	dur := fmt.Sprintf("%ds", runDurationSec(run))
	tgt := runTarget(run)
	var rows []DurationRow
	rows = append(rows, DurationRow{"Aggregate", "aggregate", tgt, run.Profile, start, end, dur})
	for _, c := range run.Clients {
		t := c.Target
		if t == "" {
			t = tgt
		}
		rows = append(rows, DurationRow{shortHost(c.Host), "client", t, run.Profile, start, end, dur})
	}
	for _, t := range run.Targets {
		rows = append(rows, DurationRow{shortHost(t.Target), "target", t.Target, run.Profile, start, end, dur})
	}
	return rows
}

func buildMetricRows(run *schema.RunResult) []KVRow {
	r := run.Results
	l := r.LatencyUS
	lbl := workloadLabels(run)
	opsLabel := lbl.OpsRate
	if lbl.IsObject {
		opsLabel = lbl.OpsRate + " (" + lbl.OpsUnit + ")"
	}
	rows := []KVRow{
		{opsLabel, fmt.Sprintf("%.0f", r.IOPS)},
		{lbl.ReadOp, fmt.Sprintf("%.0f", r.ReadIOPS)},
		{lbl.WriteOp, fmt.Sprintf("%.0f", r.WriteIOPS)},
		{"Throughput (MB/s)", fmt.Sprintf("%.2f", r.ThroughputMBps)},
		{"Ops/sec", fmt.Sprintf("%.0f", r.OpsPerSec)},
		{"Latency min (µs)", fµs(l.Min)},
		{"Latency mean (µs)", fµs(l.Mean)},
		{"Latency p50 (µs)", fµs(l.P50)},
		{"Latency p75 (µs)", fµs(l.P75)},
		{"Latency p90 (µs)", fµs(l.P90)},
		{"Latency p95 (µs)", fµs(l.P95)},
		{"Latency p99 (µs)", fµs(l.P99)},
		{"Latency p99.9 (µs)", fµs(l.P999)},
		{"Latency p99.99 (µs)", fµs(l.P9999)},
		{"Latency max (µs)", fµs(l.Max)},
		{"CPU %", fmt.Sprintf("%.1f", r.CPUPercent)},
	}
	return rows
}

func buildPercentileRows(run *schema.RunResult) []PercentileRow {
	labels, values := metrics.PercentileSeries(run.Results)
	counts := run.Results.PercentileCounts
	var rows []PercentileRow
	for i, label := range labels {
		lat := "—"
		if i < len(values) && values[i] > 0 {
			lat = fµs(values[i])
		}
		cnt := "—"
		if counts != nil {
			if c, ok := counts[label]; ok {
				cnt = fmt.Sprintf("%d", c)
			}
		}
		rows = append(rows, PercentileRow{Label: label, Latency: lat, Count: cnt})
	}
	return rows
}

func buildKPIs(run *schema.RunResult, cd CardData) []KPI {
	dur := run.Workload.DurationSec
	if dur <= 0 {
		dur = runDurationSec(run)
	}
	t := effectiveTotals(run.Results, dur)
	lbl := cd.Labels
	totalMB := "—"
	totalRec := "—"
	if t.TotalMB > 0 {
		totalMB = fmt.Sprintf("%.1f", t.TotalMB)
		cd.TotalMB = totalMB
	}
	if t.TotalRecords > 0 {
		totalRec = formatCompactInt(t.TotalRecords)
		cd.TotalRecords = totalRec
	}
	opsKPI := KPI{Label: lbl.OpsRate, Value: cd.IOPS, Unit: lbl.OpsUnit, Hint: "Operation throughput"}
	if lbl.IsObject && lbl.Operation != "" {
		opsKPI.Hint = strings.ToUpper(lbl.Operation) + " workload"
	}
	return []KPI{
		opsKPI,
		{Label: "Throughput", Value: cd.Throughput, Unit: "MB/s", Hint: "Data rate"},
		{Label: "Total data", Value: totalMB, Unit: "MB", Hint: "Volume transferred"},
		{Label: lbl.TotalVolume, Value: totalRec, Unit: "", Hint: "Completed operations"},
		{Label: "p50 latency", Value: cd.P50, Unit: "µs", Hint: "Median"},
		{Label: "p95 latency", Value: cd.P95, Unit: "µs", Hint: "Tail start"},
		{Label: "p99 latency", Value: cd.P99, Unit: "µs", Hint: "SLA critical"},
		{Label: "Duration", Value: fmt.Sprintf("%d", dur), Unit: "sec", Hint: "Test window"},
	}
}

func buildNavSections(cd CardData) []NavSection {
	sections := []NavSection{
		{ID: "overview", Title: "Overview"},
	}
	if cd.HasTotals {
		sections = append(sections, NavSection{ID: "totals", Title: "Totals"})
	}
	if len(cd.PercentileRows) > 0 {
		sections = append(sections, NavSection{ID: "percentiles", Title: "Percentiles"})
	}
	seen := map[string]bool{}
	for _, g := range cd.ChartGroups {
		id := g.ID
		if id == "" {
			id = sectionSlug(g.Title)
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		title := g.Title
		if g.Collapsed {
			title = title + " (" + fmt.Sprintf("%d", len(g.Panels)) + ")"
		}
		sections = append(sections, NavSection{ID: id, Title: title})
	}
	sections = append(sections, NavSection{ID: "nodes", Title: "Node matrix"})
	if cd.HasIntervals {
		sections = append(sections, NavSection{ID: "intervals", Title: "Interval data"})
	}
	return sections
}

func formatCompactInt(n int64) string {
	if n >= 1_000_000_000 {
		return fmt.Sprintf("%.2fB", float64(n)/1_000_000_000)
	}
	if n >= 1_000_000 {
		return fmt.Sprintf("%.2fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	}
	return fmt.Sprintf("%d", n)
}

func buildIntervalRows(run *schema.RunResult) []IntervalRow {
	var rows []IntervalRow
	for _, iv := range run.Results.Intervals {
		ts := "—"
		if !iv.Timestamp.IsZero() {
			ts = iv.Timestamp.Format(time.RFC3339)
		}
		rows = append(rows, IntervalRow{
			Seq: fmt.Sprintf("%d", iv.Seq), Timestamp: ts,
			IOPS: fmt.Sprintf("%.0f", iv.IOPS), ReadIOPS: fmt.Sprintf("%.0f", iv.ReadIOPS),
			WriteIOPS: fmt.Sprintf("%.0f", iv.WriteIOPS), MBps: fmt.Sprintf("%.2f", iv.ThroughputMBps),
			Avg: fµs(iv.AvgLatencyUS), Min: fµs(iv.MinLatencyUS), Max: fµs(iv.MaxLatencyUS),
			WTimeout: fmt.Sprintf("%d", iv.WriteTimeoutEvents), RTimeout: fmt.Sprintf("%d", iv.ReadTimeoutEvents),
		})
	}
	return rows
}

func fµs(v float64) string {
	if v <= 0 {
		return "—"
	}
	if v >= 1000 {
		return fmt.Sprintf("%.1f", v)
	}
	return fmt.Sprintf("%.2f", v)
}

func formatDelta(pct float64, higherIsBetter bool) string {
	if math.Abs(pct) < 0.05 {
		return "±0%"
	}
	arrow := "▲"
	if pct < 0 {
		arrow = "▼"
	}
	if !higherIsBetter {
		if pct > 0 {
			arrow = "▲"
		} else {
			arrow = "▼"
		}
	}
	return fmt.Sprintf("%s %.1f%%", arrow, math.Abs(pct))
}

func deltaClass(pct float64, higherIsBetter bool) string {
	if math.Abs(pct) < 1 {
		return "neutral"
	}
	improved := pct > 0
	if !higherIsBetter {
		improved = pct < 0
	}
	if improved {
		return "good"
	}
	return "bad"
}
