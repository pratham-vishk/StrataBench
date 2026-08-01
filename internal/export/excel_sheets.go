package export

import (
	"fmt"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/pratham-vishk/stratabench/internal/schema"
	"github.com/pratham-vishk/stratabench/internal/version"
)

type excelStyles struct {
	header int
	title  int
	label  int
	pass   int
	fail   int
}

func newExcelStyles(f *excelize.File) excelStyles {
	header, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"334155"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	title, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 16, Color: "1E3A5F"},
		Alignment: &excelize.Alignment{Vertical: "center"},
	})
	label, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "334155"},
	})
	pass, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "166534"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"DCFCE7"}, Pattern: 1},
	})
	fail, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "991B1B"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"FEE2E2"}, Pattern: 1},
	})
	return excelStyles{header: header, title: title, label: label, pass: pass, fail: fail}
}

type nodeRow struct {
	label string
	role  string
	res   schema.Results
}

func collectNodeRows(run *schema.RunResult) []nodeRow {
	rows := []nodeRow{{"Aggregate", "aggregate", run.Results}}
	for _, c := range run.Clients {
		rows = append(rows, nodeRow{nodeLabel(c.Host), "client", c.Results})
	}
	for _, t := range run.Targets {
		rows = append(rows, nodeRow{nodeLabel(t.Target), "target", t.Results})
	}
	return rows
}

func nodeLabel(addr string) string {
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

func runTarget(run *schema.RunResult) string {
	if run.Target.Device != "" {
		return run.Target.Device
	}
	return run.Target.Endpoint
}

func hasReadWriteData(rows []nodeRow) bool {
	for _, r := range rows {
		if r.res.ReadIOPS > 0 || r.res.WriteIOPS > 0 {
			return true
		}
	}
	return false
}

func writeSummarySheet(f *excelize.File, run *schema.RunResult, st excelStyles) (int, error) {
	sheet := "Summary"
	idx, _ := f.NewSheet(sheet)
	f.SetCellValue(sheet, "A1", version.Name+" Report")
	f.SetCellStyle(sheet, "A1", "A1", st.title)

	rows := collectNodeRows(run)
	nodeNames := make([]string, 0, len(rows))
	for _, r := range rows {
		nodeNames = append(nodeNames, r.label)
	}

	kv := []struct{ k, v string }{
		{"Version", version.Version},
		{"Generated", time.Now().UTC().Format(time.RFC3339)},
		{"Run ID", run.RunID},
		{"Profile", run.Profile},
		{"Engine / Layer", fmt.Sprintf("%s / %s", run.Engine, run.Layer)},
		{"Topology", run.Topology},
		{"Target", runTarget(run)},
		{"Workload", fmt.Sprintf("%s %s qd=%d threads=%d", run.Workload.Pattern, run.Workload.BlockSize, run.Workload.QueueDepth, run.Workload.Threads)},
		{"Duration", fmt.Sprintf("%ds (ramp %ds)", run.Workload.DurationSec, run.Workload.RampTimeSec)},
		{"Time unit", "microseconds (µs)"},
		{"Nodes", strings.Join(nodeNames, ", ")},
		{"Validation", validationText(run.Validation.Passed)},
		{"Mock data", fmt.Sprintf("%v", run.Mock)},
	}
	for i, pair := range kv {
		row := i + 3
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), pair.k)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), pair.v)
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), st.label)
		if pair.k == "Validation" {
			if run.Validation.Passed {
				f.SetCellStyle(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), st.pass)
			} else {
				f.SetCellStyle(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), st.fail)
			}
		}
	}
	f.SetColWidth(sheet, "A", "A", 22)
	f.SetColWidth(sheet, "B", "B", 48)
	return idx, nil
}

func validationText(passed bool) string {
	if passed {
		return "PASSED"
	}
	return "FAILED"
}

func writeDurationsSheet(f *excelize.File, run *schema.RunResult, st excelStyles) error {
	sheet := "Durations"
	_, _ = f.NewSheet(sheet)
	headers := []string{"Node", "Role", "Target", "Profile", "Start", "End", "Duration"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}
	f.SetCellStyle(sheet, "A1", "G1", st.header)

	start := run.Timestamps.StartedAt.UTC().Format(time.RFC3339)
	end := run.Timestamps.CompletedAt.UTC().Format(time.RFC3339)
	dur := runDurationSec(run)

	type durRow struct {
		node, role, target, profile string
	}
	var drows []durRow
	drows = append(drows, durRow{"Aggregate", "aggregate", runTarget(run), run.Profile})
	for _, c := range run.Clients {
		tgt := c.Target
		if tgt == "" {
			tgt = runTarget(run)
		}
		drows = append(drows, durRow{nodeLabel(c.Host), "client", tgt, run.Profile})
	}
	for _, t := range run.Targets {
		drows = append(drows, durRow{nodeLabel(t.Target), "target", t.Target, run.Profile})
	}

	for i, d := range drows {
		row := i + 2
		vals := []any{d.node, d.role, d.target, d.profile, start, end, dur}
		for j, v := range vals {
			cell, _ := excelize.CoordinatesToCellName(j+1, row)
			f.SetCellValue(sheet, cell, v)
		}
	}
	f.SetColWidth(sheet, "A", "G", 20)
	return nil
}

func runDurationSec(run *schema.RunResult) int {
	if !run.Timestamps.StartedAt.IsZero() && !run.Timestamps.CompletedAt.IsZero() {
		return int(run.Timestamps.CompletedAt.Sub(run.Timestamps.StartedAt).Seconds())
	}
	return run.Workload.DurationSec
}

func writeReportSheet(f *excelize.File, run *schema.RunResult, st excelStyles) error {
	sheet := "Report"
	_, _ = f.NewSheet(sheet)
	f.SetCellValue(sheet, "A1", "Aggregate metrics")
	f.SetCellStyle(sheet, "A1", "A1", st.title)

	f.SetCellValue(sheet, "A3", "Metric")
	f.SetCellValue(sheet, "B3", "Value")
	f.SetCellStyle(sheet, "A3", "B3", st.header)

	metrics := []struct {
		name string
		val  any
	}{
		{"IOPS", run.Results.IOPS},
		{"Read IOPS", run.Results.ReadIOPS},
		{"Write IOPS", run.Results.WriteIOPS},
		{"Throughput (MB/s)", run.Results.ThroughputMBps},
		{"Ops/sec", run.Results.OpsPerSec},
		{"Latency min (µs)", run.Results.LatencyUS.Min},
		{"Latency mean (µs)", run.Results.LatencyUS.Mean},
		{"Latency p50 (µs)", run.Results.LatencyUS.P50},
		{"Latency p75 (µs)", run.Results.LatencyUS.P75},
		{"Latency p90 (µs)", run.Results.LatencyUS.P90},
		{"Latency p95 (µs)", run.Results.LatencyUS.P95},
		{"Latency p99 (µs)", run.Results.LatencyUS.P99},
		{"Latency p99.9 (µs)", run.Results.LatencyUS.P999},
		{"Latency p99.99 (µs)", run.Results.LatencyUS.P9999},
		{"Latency max (µs)", run.Results.LatencyUS.Max},
		{"CPU %", run.Results.CPUPercent},
		{"Duration (s)", run.Workload.DurationSec},
		{"Ramp (s)", run.Workload.RampTimeSec},
	}
	for i, m := range metrics {
		row := 4 + i
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), m.name)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), m.val)
	}
	f.SetColWidth(sheet, "A", "A", 22)
	f.SetColWidth(sheet, "B", "B", 36)
	return nil
}

// Nodes column layout (1-based): 1 Node, 2 Role, 3 IOPS, 4 Read IOPS, 5 Write IOPS,
// 6 MB/s, 7 min, 8 mean, 9 p50, 10 p75, 11 p90, 12 p95, 13 p99, 14 p99.9, 15 p99.99, 16 max
const (
	colNode = 1
	colRole = 2
	colIOPS = 3
	colReadIOPS = 4
	colWriteIOPS = 5
	colMBps = 6
	colMinLat = 7
	colMeanLat = 8
	colMaxLat = 16
)

func writeNodesSheet(f *excelize.File, rows []nodeRow, st excelStyles) (int, error) {
	sheet := "Nodes"
	_, _ = f.NewSheet(sheet)
	headers := []string{
		"Node", "Role", "IOPS", "Read IOPS", "Write IOPS", "MB/s",
		"min", "mean", "p50", "p75", "p90", "p95", "p99", "p99.9", "p99.99", "max",
	}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}
	f.SetCellStyle(sheet, "A1", "P1", st.header)

	for i, r := range rows {
		rowNum := i + 2
		lv := latencyRow(r.res.LatencyUS)
		vals := []any{
			r.label, r.role, r.res.IOPS, r.res.ReadIOPS, r.res.WriteIOPS, r.res.ThroughputMBps,
		}
		for _, v := range lv {
			vals = append(vals, v)
		}
		for j, v := range vals {
			cell, _ := excelize.CoordinatesToCellName(j+1, rowNum)
			f.SetCellValue(sheet, cell, v)
		}
	}
	f.SetColWidth(sheet, "A", "P", 14)
	return len(rows) + 1, nil
}

func writeTotalThroughputMBSheet(f *excelize.File, lastRow int, st excelStyles) error {
	sheet := "Total_Throughput_MB"
	_, _ = f.NewSheet(sheet)
	return addNodesBarChart(f, sheet, "Nodes", colMBps, lastRow, "Total throughput (MB/s) by node", st)
}

func writeTotalMinLatencySheet(f *excelize.File, lastRow int, st excelStyles) error {
	sheet := "Total_Min_Latency"
	_, _ = f.NewSheet(sheet)
	return addNodesBarChart(f, sheet, "Nodes", colMinLat, lastRow, "Total min latency (µs) by node", st)
}

func writeTotalAvgLatencySheet(f *excelize.File, lastRow int, st excelStyles) error {
	sheet := "Total_Avg_Latency"
	_, _ = f.NewSheet(sheet)
	return addNodesBarChart(f, sheet, "Nodes", colMeanLat, lastRow, "Total mean latency (µs) by node", st)
}

func writeTotalMaxLatencySheet(f *excelize.File, lastRow int, st excelStyles) error {
	sheet := "Total_Max_Latency"
	_, _ = f.NewSheet(sheet)
	return addNodesBarChart(f, sheet, "Nodes", colMaxLat, lastRow, "Total max latency (µs) by node", st)
}

func writeWriteReadSheet(f *excelize.File, rows []nodeRow, st excelStyles) error {
	if !hasReadWriteData(rows) {
		return nil
	}
	sheet := "Write_Read"
	_, _ = f.NewSheet(sheet)
	headers := []string{"Node", "Read IOPS", "Write IOPS"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}
	f.SetCellStyle(sheet, "A1", "C1", st.header)
	for i, r := range rows {
		rowNum := i + 2
		f.SetCellValue(sheet, fmt.Sprintf("A%d", rowNum), r.label)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", rowNum), r.res.ReadIOPS)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", rowNum), r.res.WriteIOPS)
	}
	lastRow := len(rows) + 1
	f.SetColWidth(sheet, "A", "C", 18)

	if err := f.AddChart(sheet, "E2", &excelize.Chart{
		Type: excelize.Col,
		Series: []excelize.ChartSeries{
			{
				Name:       sheet + "!$B$1",
				Categories: fmt.Sprintf("%s!$A$2:$A$%d", sheet, lastRow),
				Values:     fmt.Sprintf("%s!$B$2:$B$%d", sheet, lastRow),
			},
			{
				Name:       sheet + "!$C$1",
				Categories: fmt.Sprintf("%s!$A$2:$A$%d", sheet, lastRow),
				Values:     fmt.Sprintf("%s!$C$2:$C$%d", sheet, lastRow),
			},
		},
		Title: excelize.ChartTitle{
			Paragraph: []excelize.RichTextRun{{Text: "Read vs write IOPS by node"}},
		},
		Format: excelize.GraphicOptions{ScaleX: 1.15, ScaleY: 1.15},
	}); err != nil {
		return err
	}
	return nil
}

func addNodesBarChart(f *excelize.File, chartSheet, dataSheet string, valueCol, lastRow int, title string, _ excelStyles) error {
	colLetter, _ := excelize.ColumnNumberToName(valueCol)
	headerCell := fmt.Sprintf("%s!$%s$1", dataSheet, colLetter)
	categories := fmt.Sprintf("%s!$A$2:$A$%d", dataSheet, lastRow)
	values := fmt.Sprintf("%s!$%s$2:$%s$%d", dataSheet, colLetter, colLetter, lastRow)
	return f.AddChart(chartSheet, "A2", &excelize.Chart{
		Type: excelize.Col,
		Series: []excelize.ChartSeries{{
			Name:       headerCell,
			Categories: categories,
			Values:     values,
		}},
		Title: excelize.ChartTitle{
			Paragraph: []excelize.RichTextRun{{Text: title}},
		},
		Format: excelize.GraphicOptions{ScaleX: 1.2, ScaleY: 1.2},
	})
}
