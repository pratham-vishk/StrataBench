package metrics

import (
	"strconv"
	"strings"

	"github.com/pratham-vishk/stratabench/internal/schema"
)

// StandardPercentileLabels matches sbk-charts percentile columns (p5 → p99.99).
var StandardPercentileLabels = []string{
	"p5", "p10", "p15", "p20", "p25", "p30", "p35", "p40", "p45",
	"p50", "p55", "p60", "p65", "p70", "p75", "p80", "p85", "p90",
	"p92.5", "p95", "p97.5", "p99", "p99.25", "p99.5", "p99.75",
	"p99.9", "p99.95", "p99.99",
}

// LatencyBandGroups matches storage-benchmark latency band charts (5 groups).
var LatencyBandGroups = [][]string{
	{"min", "p5"},
	{"p5", "p10", "p15", "p20", "p25", "p30", "p35", "p40", "p45", "p50"},
	{"p50", "mean"},
	{"p50", "p55", "p60", "p65", "p70", "p75", "p80", "p85", "p90"},
	{"p92.5", "p95", "p97.5", "p99", "p99.25", "p99.5", "p99.75", "p99.9", "p99.95", "p99.99"},
}

// PercentileGroups splits labels for sbk-charts Total_Percentiles_1/_2 sheets.
var PercentileGroups = [][]string{
	{"p5", "p10", "p15", "p20", "p25", "p30", "p35", "p40", "p50"},
	{"p50", "p55", "p60", "p65", "p70", "p75", "p80", "p85", "p90",
		"p92.5", "p95", "p97.5", "p99", "p99.25", "p99.5", "p99.75", "p99.9", "p99.95", "p99.99"},
}

// NormalizePercentileKey converts SBK/CSV column names to canonical keys.
func NormalizePercentileKey(name string) string {
	name = strings.TrimSpace(name)
	lower := strings.ToLower(name)
	lower = strings.ReplaceAll(lower, "percentile_", "p")
	lower = strings.ReplaceAll(lower, "percentile", "p")
	lower = strings.TrimPrefix(lower, "p_")
	if strings.HasPrefix(lower, "p") {
		return lower
	}
	if v, err := strconv.ParseFloat(name, 64); err == nil {
		return formatPercentileKey(v)
	}
	return ""
}

func formatPercentileKey(v float64) string {
	s := strconv.FormatFloat(v, 'f', -1, 64)
	return "p" + s
}

// PercentileKeyFromSBKColumn maps "Percentile_99.9" or "percentile_99.9" → "p99.9".
func PercentileKeyFromSBKColumn(col string) string {
	col = strings.ToLower(strings.TrimSpace(col))
	if strings.HasPrefix(col, "percentile_count_") {
		return ""
	}
	if strings.HasPrefix(col, "percentile_") {
		return "p" + strings.TrimPrefix(col, "percentile_")
	}
	return ""
}

// PercentileCountKeyFromSBKColumn maps "Percentile_Count_99.9" or "percentile_count_99.9" → "p99.9".
func PercentileCountKeyFromSBKColumn(col string) string {
	col = strings.ToLower(strings.TrimSpace(col))
	if !strings.HasPrefix(col, "percentile_count_") {
		return ""
	}
	return "p" + strings.TrimPrefix(col, "percentile_count_")
}

// PercentileSeries returns labels and latency values (µs) for charts/Excel.
func PercentileSeries(res schema.Results) ([]string, []float64) {
	if len(res.Percentiles) > 0 {
		var labels []string
		var vals []float64
		for _, l := range StandardPercentileLabels {
			if v, ok := res.Percentiles[l]; ok && v > 0 {
				labels = append(labels, l)
				vals = append(vals, v)
			}
		}
		if len(labels) > 0 {
			return labels, vals
		}
	}
	return legacyLatencySeries(res.LatencyUS)
}

func legacyLatencySeries(lat schema.LatencyUS) ([]string, []float64) {
	type pair struct {
		label string
		val   float64
	}
	pairs := []pair{
		{"min", lat.Min}, {"mean", lat.Mean}, {"p50", lat.P50}, {"p75", lat.P75},
		{"p90", lat.P90}, {"p95", lat.P95}, {"p99", lat.P99},
		{"p99.9", lat.P999}, {"p99.99", lat.P9999}, {"max", lat.Max},
	}
	var labels []string
	var vals []float64
	for _, p := range pairs {
		if p.val <= 0 && p.label != "min" {
			continue
		}
		labels = append(labels, p.label)
		vals = append(vals, p.val)
	}
	return labels, vals
}

// PercentileCountSeries returns histogram data when available.
func PercentileCountSeries(res schema.Results) ([]string, []int64) {
	if len(res.PercentileCounts) == 0 {
		return nil, nil
	}
	var labels []string
	var counts []int64
	for _, l := range StandardPercentileLabels {
		if c, ok := res.PercentileCounts[l]; ok {
			labels = append(labels, l)
			counts = append(counts, c)
		}
	}
	return labels, counts
}

// PopulateLatencyUS copies known percentile map entries into LatencyUS fields.
func PopulateLatencyUS(lat *schema.LatencyUS, p map[string]float64) {
	if lat == nil || len(p) == 0 {
		return
	}
	if v, ok := p["p50"]; ok {
		lat.P50 = v
	}
	if v, ok := p["p75"]; ok {
		lat.P75 = v
	}
	if v, ok := p["p90"]; ok {
		lat.P90 = v
	}
	if v, ok := p["p95"]; ok {
		lat.P95 = v
	}
	if v, ok := p["p99"]; ok {
		lat.P99 = v
	}
	if v, ok := p["p99.9"]; ok {
		lat.P999 = v
	}
	if v, ok := p["p99.99"]; ok {
		lat.P9999 = v
	}
}

// LatencyUnitScale converts SBK latency units to microseconds.
func LatencyUnitScale(unit string) float64 {
	switch strings.ToUpper(strings.TrimSpace(unit)) {
	case "NANOSECONDS", "NS":
		return 0.001
	case "MICROSECONDS", "US", "µS":
		return 1
	case "MILLISECONDS", "MS":
		return 1000
	case "SECONDS", "S":
		return 1_000_000
	default:
		return 1
	}
}
