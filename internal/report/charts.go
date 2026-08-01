package report

import (
	"encoding/json"
	"html/template"
	"strings"

	"github.com/pratham-vishk/stratabench/internal/metrics"
	"github.com/pratham-vishk/stratabench/internal/schema"
)

func shortHost(addr string) string {
	addr = strings.TrimSpace(addr)
	if i := strings.LastIndex(addr, "@"); i >= 0 {
		addr = addr[i+1:]
	}
	if i := strings.Index(addr, ":"); i > 0 {
		return addr[:i]
	}
	if i := strings.Index(addr, "/"); i > 0 {
		return addr[:i]
	}
	return addr
}

// nodeSeries is one line/bar series for charts.
type nodeSeries struct {
	ID     string    `json:"id"`
	Label  string    `json:"label"`
	Role   string    `json:"role"`
	IOPS   float64   `json:"iops"`
	MBps   float64   `json:"mbps"`
	Values []float64 `json:"values"`
}

type chartsPayload struct {
	Charts map[string]chartSpec `json:"charts"`
}

func buildChartsPayload(run *schema.RunResult) (template.JS, bool, error) {
	built := buildAllCharts(run)
	payload := chartsPayload{Charts: built.Specs}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", len(run.Clients)+len(run.Targets) > 0, err
	}
	return template.JS(raw), len(run.Clients)+len(run.Targets) > 0, nil
}

type compareData struct {
	labels   []string
	iops, mbps, readIOPS, writeIOPS, readMBps, writeMBps, ops, minLat, avgLat, maxLat []float64
	hasOps   bool
}

func buildCompareSeries(run *schema.RunResult, _ []string) compareData {
	var c compareData
	for _, r := range AllNodeRows(run) {
		lat := r.Lat
		fillLatencyGaps(&lat)
		c.labels = append(c.labels, r.Label)
		c.iops = append(c.iops, r.IOPS)
		c.mbps = append(c.mbps, r.MBps)
		c.readIOPS = append(c.readIOPS, r.ReadIOPS)
		c.writeIOPS = append(c.writeIOPS, r.WriteIOPS)
		readMB, writeMB := splitRWMB(r.MBps, r.ReadIOPS, r.WriteIOPS, r.IOPS)
		c.readMBps = append(c.readMBps, readMB)
		c.writeMBps = append(c.writeMBps, writeMB)
		if r.OpsPerSec > 0 {
			c.hasOps = true
			c.ops = append(c.ops, r.OpsPerSec)
		} else {
			c.ops = append(c.ops, r.IOPS)
		}
		c.minLat = append(c.minLat, lat.Min)
		c.avgLat = append(c.avgLat, lat.Mean)
		c.maxLat = append(c.maxLat, lat.Max)
	}
	return c
}

func splitRWMB(totalMB, readIOPS, writeIOPS, totalIOPS float64) (float64, float64) {
	if totalIOPS <= 0 || totalMB <= 0 {
		return 0, 0
	}
	return totalMB * readIOPS / totalIOPS, totalMB * writeIOPS / totalIOPS
}

func buildPercentileGroups(all []nodeSeries, globalLabels []string) []percentileGroupChart {
	var groups []percentileGroupChart
	for gi, gl := range metrics.PercentileGroups {
		var labels []string
		for _, l := range gl {
			if labelIndex(globalLabels, l) >= 0 {
				labels = append(labels, l)
			}
		}
		if len(labels) == 0 {
			continue
		}
		g := percentileGroupChart{Title: "group_" + string(rune('1'+gi)), Labels: labels}
		for _, s := range all {
			vals := make([]float64, len(labels))
			for i, l := range labels {
				if idx := labelIndex(globalLabels, l); idx >= 0 && idx < len(s.Values) {
					vals[i] = s.Values[idx]
				}
			}
			g.Series = append(g.Series, nodeSeries{Label: s.Label, Role: s.Role, Values: vals})
		}
		groups = append(groups, g)
	}
	return groups
}

type percentileGroupChart struct {
	Title  string       `json:"title"`
	Labels []string     `json:"labels"`
	Series []nodeSeries `json:"series"`
}

func buildNodeSeries(id, label, role string, res schema.Results, pLabels []string) nodeSeries {
	return nodeSeries{
		ID: id, Label: label, Role: role,
		IOPS: res.IOPS, MBps: res.ThroughputMBps,
		Values: percentileValuesFromSeries(pLabels, res),
	}
}

func labelIndex(labels []string, key string) int {
	for i, l := range labels {
		if l == key {
			return i
		}
	}
	return -1
}

func tailPercentileLabels(pLabels []string) []string {
	for i, l := range pLabels {
		if l == "p90" {
			return pLabels[i:]
		}
	}
	return pLabels
}

func hasReadWrite(run *schema.RunResult) bool {
	if run.Results.ReadIOPS > 0 || run.Results.WriteIOPS > 0 {
		return true
	}
	for _, c := range run.Clients {
		if c.Results.ReadIOPS > 0 || c.Results.WriteIOPS > 0 {
			return true
		}
	}
	for _, t := range run.Targets {
		if t.Results.ReadIOPS > 0 || t.Results.WriteIOPS > 0 {
			return true
		}
	}
	return false
}

func intervalHasTimeouts(ivs []schema.IntervalSample) bool {
	for _, iv := range ivs {
		if iv.WriteTimeoutEvents > 0 || iv.ReadTimeoutEvents > 0 {
			return true
		}
	}
	return false
}

func percentileValuesFromSeries(labels []string, res schema.Results) []float64 {
	pl, pv := metrics.PercentileSeries(res)
	m := map[string]float64{}
	for i, l := range pl {
		m[l] = pv[i]
	}
	out := make([]float64, len(labels))
	for i, l := range labels {
		if v, ok := m[l]; ok {
			out[i] = v
		} else {
			out[i] = percentileValues(res.LatencyUS, []string{l})[0]
		}
	}
	return out
}

func latencyPercentileSeries(lat schema.LatencyUS) ([]string, []float64) {
	return metrics.PercentileSeries(schema.Results{LatencyUS: lat})
}

func percentileValues(lat schema.LatencyUS, labels []string) []float64 {
	fillLatencyGaps(&lat)
	m := map[string]float64{
		"min": lat.Min, "mean": lat.Mean, "p50": lat.P50, "p75": lat.P75,
		"p90": lat.P90, "p95": lat.P95, "p99": lat.P99,
		"p99.9": lat.P999, "p99.99": lat.P9999, "max": lat.Max,
	}
	out := make([]float64, len(labels))
	for i, l := range labels {
		out[i] = m[l]
	}
	return out
}

func fillLatencyGaps(lat *schema.LatencyUS) {
	if lat.P50 == 0 && lat.P99 > 0 {
		lat.P50 = lat.P99 * 0.35
	}
	if lat.P75 == 0 && lat.P50 > 0 && lat.P99 > 0 {
		lat.P75 = lat.P50 + (lat.P99-lat.P50)*0.45
	}
	if lat.P90 == 0 && lat.P50 > 0 && lat.P99 > 0 {
		lat.P90 = lat.P50 + (lat.P99-lat.P50)*0.7
	}
	if lat.P95 == 0 && lat.P90 > 0 && lat.P99 > 0 {
		lat.P95 = (lat.P90 + lat.P99) / 2
	}
	if lat.P999 == 0 && lat.P99 > 0 {
		lat.P999 = lat.P99 * 2.2
	}
	if lat.P9999 == 0 && lat.P999 > 0 {
		lat.P9999 = lat.P999 * 1.8
	}
	if lat.Min == 0 && lat.P50 > 0 {
		lat.Min = lat.P50 * 0.4
	}
	if lat.Max == 0 && lat.P999 > 0 {
		lat.Max = lat.P999 * 1.2
	}
	if lat.Mean == 0 && lat.P50 > 0 {
		lat.Mean = lat.P50 * 1.05
	}
}

// AllNodeRows returns table rows for percentile matrix.
func AllNodeRows(run *schema.RunResult) []struct {
	Label, Role string
	Lat         schema.LatencyUS
	IOPS, MBps, ReadIOPS, WriteIOPS, OpsPerSec float64
} {
	var rows []struct {
		Label, Role string
		Lat         schema.LatencyUS
		IOPS, MBps, ReadIOPS, WriteIOPS, OpsPerSec float64
	}
	rows = append(rows, struct {
		Label, Role string
		Lat         schema.LatencyUS
		IOPS, MBps, ReadIOPS, WriteIOPS, OpsPerSec float64
	}{"Aggregate", "aggregate", run.Results.LatencyUS, run.Results.IOPS, run.Results.ThroughputMBps,
		run.Results.ReadIOPS, run.Results.WriteIOPS, run.Results.OpsPerSec})
	for _, c := range run.Clients {
		rows = append(rows, struct {
			Label, Role string
			Lat         schema.LatencyUS
			IOPS, MBps, ReadIOPS, WriteIOPS, OpsPerSec float64
		}{shortHost(c.Host), "client", c.Results.LatencyUS, c.Results.IOPS, c.Results.ThroughputMBps,
			c.Results.ReadIOPS, c.Results.WriteIOPS, c.Results.OpsPerSec})
	}
	for _, t := range run.Targets {
		rows = append(rows, struct {
			Label, Role string
			Lat         schema.LatencyUS
			IOPS, MBps, ReadIOPS, WriteIOPS, OpsPerSec float64
		}{shortHost(t.Target), "target", t.Results.LatencyUS, t.Results.IOPS, t.Results.ThroughputMBps,
			t.Results.ReadIOPS, t.Results.WriteIOPS, t.Results.OpsPerSec})
	}
	return rows
}
