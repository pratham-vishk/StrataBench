package export

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/xuri/excelize/v2"

	"github.com/pratham-vishk/stratabench/internal/metrics"
	"github.com/pratham-vishk/stratabench/internal/schema"
)

// WriteExcel exports a single run as a styled workbook (sbk-charts-style sheets).
func WriteExcel(run *schema.RunResult, path string) error {
	f := excelize.NewFile()
	defer f.Close()

	_ = f.DeleteSheet("Sheet1")
	st := newExcelStyles(f)
	rows := collectNodeRows(run)

	summaryIdx, err := writeSummarySheet(f, run, st)
	if err != nil {
		return err
	}
	if err := writeDurationsSheet(f, run, st); err != nil {
		return err
	}
	if err := writeReportSheet(f, run, st); err != nil {
		return err
	}
	if err := writeLatencySheet(f, run, st.header); err != nil {
		return err
	}
	lastRow, err := writeNodesSheet(f, rows, st)
	if err != nil {
		return err
	}
	if err := writeTotalThroughputMBSheet(f, lastRow, st); err != nil {
		return err
	}
	if err := writeWriteReadSheet(f, rows, st); err != nil {
		return err
	}
	if err := writeTotalMinLatencySheet(f, lastRow, st); err != nil {
		return err
	}
	if err := writeTotalAvgLatencySheet(f, lastRow, st); err != nil {
		return err
	}
	if err := writeTotalMaxLatencySheet(f, lastRow, st); err != nil {
		return err
	}
	if err := writeTotalPercentileSheets(f, run, rows, st); err != nil {
		return err
	}
	if err := writePercentileHistogramSheet(f, run, st); err != nil {
		return err
	}
	if err := writeIntervalsSheet(f, run, st); err != nil {
		return err
	}
	if err := writeTimeoutSheets(f, run, st); err != nil {
		return err
	}

	f.SetActiveSheet(summaryIdx)

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := f.SaveAs(path); err != nil {
		return err
	}
	fmt.Printf("excel written: %s\n", path)
	return nil
}

func writeLatencySheet(f *excelize.File, run *schema.RunResult, headerStyle int) error {
	sheet := "Latency"
	_, _ = f.NewSheet(sheet)
	labels, vals := metrics.PercentileSeries(run.Results)
	if len(labels) == 0 {
		labels = []string{"min", "mean", "p50", "p75", "p90", "p95", "p99", "p99.9", "p99.99", "max"}
		vals = latencyRow(run.Results.LatencyUS)
	}

	f.SetCellValue(sheet, "A1", "Percentile")
	f.SetCellValue(sheet, "B1", "Aggregate (µs)")
	f.SetCellStyle(sheet, "A1", "B1", headerStyle)
	for i, l := range labels {
		row := i + 2
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), l)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), vals[i])
	}
	f.SetColWidth(sheet, "A", "B", 16)

	// Per-node percentile columns (sbk-charts Total_Percentiles style).
	rows := collectNodeRows(run)
	for i, r := range rows {
		col, _ := excelize.ColumnNumberToName(i + 3)
		pl, pv := metrics.PercentileSeries(r.res)
		if len(pl) == 0 {
			pl, pv = labels, latencyRow(r.res.LatencyUS)
		}
		f.SetCellValue(sheet, col+"1", r.label+" (µs)")
		for j, v := range pv {
			f.SetCellValue(sheet, fmt.Sprintf("%s%d", col, j+2), v)
		}
	}

	lastRow := len(labels) + 1
	if err := f.AddChart(sheet, "D2", &excelize.Chart{
		Type: excelize.Line,
		Series: []excelize.ChartSeries{{
			Name:       sheet + "!$B$1",
			Categories: fmt.Sprintf("%s!$A$2:$A$%d", sheet, lastRow),
			Values:     fmt.Sprintf("%s!$B$2:$B$%d", sheet, lastRow),
		}},
		Title: excelize.ChartTitle{
			Paragraph: []excelize.RichTextRun{{Text: "Aggregate latency percentiles (µs)"}},
		},
		Format: excelize.GraphicOptions{ScaleX: 1.2, ScaleY: 1.2},
	}); err != nil {
		return err
	}
	return nil
}

func latencyRow(lat schema.LatencyUS) []float64 {
	return []float64{
		lat.Min, lat.Mean, lat.P50, lat.P75, lat.P90,
		lat.P95, lat.P99, lat.P999, lat.P9999, lat.Max,
	}
}

// WriteExcelRuns exports multiple runs to a history sheet with comparison charts.
func WriteExcelRuns(runs []*schema.RunResult, path string) error {
	f := excelize.NewFile()
	defer f.Close()
	st := newExcelStyles(f)

	summaryIdx, _ := f.NewSheet("Summary")
	_ = f.DeleteSheet("Sheet1")
	f.SetCellValue("Summary", "A1", "StrataBench run history")
	f.SetCellStyle("Summary", "A1", "A1", st.title)
	f.SetCellValue("Summary", "A3", "Runs")
	f.SetCellValue("Summary", "B3", len(runs))
	f.SetCellValue("Summary", "A4", "Generated")
	f.SetCellValue("Summary", "B4", fmt.Sprintf("%d profiles compared", countProfiles(runs)))

	sheet := "History"
	_, _ = f.NewSheet(sheet)
	headers := []string{"Run ID", "Profile", "Target", "IOPS", "Read IOPS", "Write IOPS", "MB/s", "min µs", "mean µs", "p99 µs", "max µs", "Validation", "Mock", "Completed"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}
	f.SetCellStyle(sheet, "A1", "N1", st.header)

	for i, run := range runs {
		row := i + 2
		target := runTarget(run)
		val := validationText(run.Validation.Passed)
		lat := run.Results.LatencyUS
		vals := []any{
			run.RunID, run.Profile, target,
			run.Results.IOPS, run.Results.ReadIOPS, run.Results.WriteIOPS,
			run.Results.ThroughputMBps, lat.Min, lat.Mean, lat.P99, lat.Max,
			val, run.Mock, run.Timestamps.CompletedAt.Format("2006-01-02T15:04:05Z"),
		}
		for j, v := range vals {
			cell, _ := excelize.CoordinatesToCellName(j+1, row)
			f.SetCellValue(sheet, cell, v)
		}
	}
	lastRow := len(runs) + 1
	f.SetColWidth(sheet, "A", "N", 16)

	if len(runs) > 0 {
		if err := f.AddChart(sheet, "P2", &excelize.Chart{
			Type: excelize.Col,
			Series: []excelize.ChartSeries{{
				Name:       sheet + "!$D$1",
				Categories: fmt.Sprintf("%s!$B$2:$B$%d", sheet, lastRow),
				Values:     fmt.Sprintf("%s!$D$2:$D$%d", sheet, lastRow),
			}},
			Title: excelize.ChartTitle{
				Paragraph: []excelize.RichTextRun{{Text: "IOPS by profile/run"}},
			},
		}); err != nil {
			return err
		}
		if err := f.AddChart(sheet, "P18", &excelize.Chart{
			Type: excelize.Col,
			Series: []excelize.ChartSeries{{
				Name:       sheet + "!$G$1",
				Categories: fmt.Sprintf("%s!$B$2:$B$%d", sheet, lastRow),
				Values:     fmt.Sprintf("%s!$G$2:$G$%d", sheet, lastRow),
			}},
			Title: excelize.ChartTitle{
				Paragraph: []excelize.RichTextRun{{Text: "Throughput (MB/s) by profile/run"}},
			},
		}); err != nil {
			return err
		}
	}

	f.SetActiveSheet(summaryIdx)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := f.SaveAs(path); err != nil {
		return err
	}
	fmt.Printf("excel history written: %s\n", path)
	return nil
}

func countProfiles(runs []*schema.RunResult) int {
	seen := map[string]struct{}{}
	for _, r := range runs {
		seen[r.Profile] = struct{}{}
	}
	return len(seen)
}
