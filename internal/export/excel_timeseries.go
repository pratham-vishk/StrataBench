package export

import (
	"fmt"

	"github.com/xuri/excelize/v2"

	"github.com/pratham-vishk/stratabench/internal/metrics"
	"github.com/pratham-vishk/stratabench/internal/schema"
)

func writeIntervalsSheet(f *excelize.File, run *schema.RunResult, st excelStyles) error {
	if len(run.Results.Intervals) == 0 {
		return nil
	}
	sheet := "Intervals"
	_, _ = f.NewSheet(sheet)
	headers := []string{
		"Seq", "Timestamp", "Elapsed (s)", "IOPS", "Read IOPS", "Write IOPS",
		"MB/s", "Read MB/s", "Write MB/s", "Avg µs", "Min µs", "Max µs",
		"Write timeouts", "Read timeouts",
	}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}
	f.SetCellStyle(sheet, "A1", "N1", st.header)

	for i, iv := range run.Results.Intervals {
		row := i + 2
		ts := ""
		if !iv.Timestamp.IsZero() {
			ts = iv.Timestamp.Format("2006-01-02T15:04:05Z")
		}
		vals := []any{
			iv.Seq, ts, iv.ElapsedSec, iv.IOPS, iv.ReadIOPS, iv.WriteIOPS,
			iv.ThroughputMBps, iv.ReadMBps, iv.WriteMBps,
			iv.AvgLatencyUS, iv.MinLatencyUS, iv.MaxLatencyUS,
			iv.WriteTimeoutEvents, iv.ReadTimeoutEvents,
		}
		for j, v := range vals {
			cell, _ := excelize.CoordinatesToCellName(j+1, row)
			f.SetCellValue(sheet, cell, v)
		}
	}
	lastRow := len(run.Results.Intervals) + 1
	f.SetColWidth(sheet, "A", "N", 14)

	if err := writeIntervalLineChart(f, sheet, "Throughput_MB", "G", lastRow, "MB/s", "Throughput variation (MB/s)"); err != nil {
		return err
	}
	if err := writeIntervalLineChart(f, sheet, "Throughput_Records", "D", lastRow, "IOPS", "Throughput variation (records/s)"); err != nil {
		return err
	}
	if err := writeIntervalLineChart(f, sheet, "Latencies-1", "J", lastRow, "Avg µs", "Average latency variation (µs)"); err != nil {
		return err
	}
	if err := writeIntervalLineChart(f, sheet, "Latencies-2", "K", lastRow, "Min µs", "Min latency variation (µs)"); err != nil {
		return err
	}
	if err := writeIntervalLineChart(f, sheet, "Latencies-3", "L", lastRow, "Max µs", "Max latency variation (µs)"); err != nil {
		return err
	}
	return nil
}

func writeIntervalLineChart(f *excelize.File, dataSheet, chartSheet, valueCol string, lastRow int, seriesName, title string) error {
	_, _ = f.NewSheet(chartSheet)
	col := valueCol
	return f.AddChart(chartSheet, "A2", &excelize.Chart{
		Type: excelize.Line,
		Series: []excelize.ChartSeries{{
			Name:       fmt.Sprintf("%s!$%s$1", dataSheet, col),
			Categories: fmt.Sprintf("%s!$A$2:$A$%d", dataSheet, lastRow),
			Values:     fmt.Sprintf("%s!$%s$2:$%s$%d", dataSheet, col, col, lastRow),
		}},
		Title: excelize.ChartTitle{
			Paragraph: []excelize.RichTextRun{{Text: title}},
		},
		Format: excelize.GraphicOptions{ScaleX: 1.2, ScaleY: 1.2},
	})
}

func writeTotalPercentileSheets(f *excelize.File, run *schema.RunResult, rows []nodeRow, st excelStyles) error {
	labels, aggVals := metrics.PercentileSeries(run.Results)
	if len(labels) < 3 {
		return nil
	}

	for gi, group := range metrics.PercentileGroups {
		sheet := fmt.Sprintf("Total_Percentiles_%d", gi+1)
		_, _ = f.NewSheet(sheet)
		f.SetCellValue(sheet, "A1", "Percentile")
		f.SetCellStyle(sheet, "A1", "A1", st.header)

		var groupLabels []string
		row := 2
		for _, gl := range group {
			if v, ok := indexValue(labels, aggVals, gl); ok {
				f.SetCellValue(sheet, fmt.Sprintf("A%d", row), gl)
				f.SetCellValue(sheet, fmt.Sprintf("B%d", row), v)
				groupLabels = append(groupLabels, gl)
				row++
			}
		}
		if row <= 2 {
			continue
		}
		lastDataRow := row - 1

		col := 3
		for _, nr := range rows {
			pl, pv := metrics.PercentileSeries(nr.res)
			colLetter, _ := excelize.ColumnNumberToName(col)
			f.SetCellValue(sheet, colLetter+"1", nr.label)
			f.SetCellStyle(sheet, colLetter+"1", colLetter+"1", st.header)
			r := 2
			for _, gl := range groupLabels {
				if v, ok := indexValue(pl, pv, gl); ok {
					f.SetCellValue(sheet, fmt.Sprintf("%s%d", colLetter, r), v)
				}
				r++
			}
			col++
		}

		if err := f.AddChart(sheet, "A"+fmt.Sprint(lastDataRow+3), &excelize.Chart{
			Type: excelize.Line,
			Series: buildPercentileChartSeries(sheet, col-1, lastDataRow, groupLabels),
			Title: excelize.ChartTitle{
				Paragraph: []excelize.RichTextRun{{Text: sheet + " (µs)"}},
			},
			Format: excelize.GraphicOptions{ScaleX: 1.15, ScaleY: 1.15},
		}); err != nil {
			return err
		}
	}
	return nil
}

func buildPercentileChartSeries(sheet string, lastCol, lastRow int, _ []string) []excelize.ChartSeries {
	var series []excelize.ChartSeries
	for c := 2; c <= lastCol; c++ {
		colLetter, _ := excelize.ColumnNumberToName(c)
		series = append(series, excelize.ChartSeries{
			Name:       fmt.Sprintf("%s!$%s$1", sheet, colLetter),
			Categories: fmt.Sprintf("%s!$A$2:$A$%d", sheet, lastRow),
			Values:     fmt.Sprintf("%s!$%s$2:$%s$%d", sheet, colLetter, colLetter, lastRow),
		})
	}
	return series
}

func writePercentileHistogramSheet(f *excelize.File, run *schema.RunResult, st excelStyles) error {
	labels, counts := metrics.PercentileCountSeries(run.Results)
	if len(labels) == 0 {
		return nil
	}
	sheet := "Total_Percentiles_Histogram"
	_, _ = f.NewSheet(sheet)
	f.SetCellValue(sheet, "A1", "Percentile")
	f.SetCellValue(sheet, "B1", "Count")
	f.SetCellStyle(sheet, "A1", "B1", st.header)
	for i, l := range labels {
		row := i + 2
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), l)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), counts[i])
	}
	lastRow := len(labels) + 1
	f.SetColWidth(sheet, "A", "B", 16)

	return f.AddChart(sheet, "D2", &excelize.Chart{
		Type: excelize.Col,
		Series: []excelize.ChartSeries{{
			Name:       sheet + "!$B$1",
			Categories: fmt.Sprintf("%s!$A$2:$A$%d", sheet, lastRow),
			Values:     fmt.Sprintf("%s!$B$2:$B$%d", sheet, lastRow),
		}},
		Title: excelize.ChartTitle{
			Paragraph: []excelize.RichTextRun{{Text: "Percentile operation counts"}},
		},
		Format: excelize.GraphicOptions{ScaleX: 1.2, ScaleY: 1.2},
	})
}

func writeTimeoutSheets(f *excelize.File, run *schema.RunResult, st excelStyles) error {
	if len(run.Results.Intervals) == 0 {
		return nil
	}
	hasTimeouts := false
	for _, iv := range run.Results.Intervals {
		if iv.WriteTimeoutEvents > 0 || iv.ReadTimeoutEvents > 0 {
			hasTimeouts = true
			break
		}
	}
	if !hasTimeouts {
		return nil
	}

	sheet := "RW_TimeoutEvents"
	_, _ = f.NewSheet(sheet)
	f.SetCellValue(sheet, "A1", "Seq")
	f.SetCellValue(sheet, "B1", "Write timeouts")
	f.SetCellValue(sheet, "C1", "Read timeouts")
	f.SetCellStyle(sheet, "A1", "C1", st.header)
	for i, iv := range run.Results.Intervals {
		row := i + 2
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), iv.Seq)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), iv.WriteTimeoutEvents)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), iv.ReadTimeoutEvents)
	}
	lastRow := len(run.Results.Intervals) + 1
	return f.AddChart(sheet, "E2", &excelize.Chart{
		Type: excelize.Line,
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
			Paragraph: []excelize.RichTextRun{{Text: "Read/write timeout events"}},
		},
	})
}

func indexValue(labels []string, vals []float64, key string) (float64, bool) {
	for i, l := range labels {
		if l == key {
			return vals[i], true
		}
	}
	return 0, false
}
