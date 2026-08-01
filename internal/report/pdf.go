package report

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-pdf/fpdf"
	"github.com/pratham-vishk/stratabench/internal/schema"
)

// WritePDF writes a one-page executive summary PDF for a run.
func WritePDF(run *schema.RunResult, outPath string) error {
	return WritePDFWithOptions(run, Options{}, outPath)
}

// WritePDFWithOptions writes a PDF report using the same card model as HTML.
func WritePDFWithOptions(run *schema.RunResult, opts Options, outPath string) error {
	cd, err := buildCardData(run, opts)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}

	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(14, 14, 14)
	pdf.SetAutoPageBreak(true, 14)
	pdf.AddPage()

	pdf.SetFont("Arial", "B", 18)
	pdf.CellFormat(0, 10, "StrataBench Report", "", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 10)
	pdf.SetTextColor(90, 90, 90)
	pdf.CellFormat(0, 5, fmt.Sprintf("Generated %s · %s", cd.GeneratedAt, cd.Version), "", 1, "L", false, 0, "")
	pdf.SetTextColor(0, 0, 0)
	pdf.Ln(4)

	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(0, 7, cd.BenchmarkLabel, "", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(0, 5, fmt.Sprintf("%s · %s · %s", run.Profile, cd.EngineLabel, run.Topology), "", 1, "L", false, 0, "")
	pdf.Ln(2)

	writePDFSection(pdf, "Key metrics")
	for _, kpi := range cd.KPIs {
		unit := kpi.Unit
		if unit != "" {
			unit = " " + unit
		}
		writePDFRow(pdf, kpi.Label, kpi.Value+unit)
	}

	writePDFSection(pdf, "Run summary")
	for _, row := range cd.SummaryRows {
		writePDFRow(pdf, row.Key, row.Value)
	}

	if len(cd.MetricRows) > 0 {
		writePDFSection(pdf, "Detailed metrics")
		limit := len(cd.MetricRows)
		if limit > 14 {
			limit = 14
		}
		for _, row := range cd.MetricRows[:limit] {
			writePDFRow(pdf, row.Key, row.Value)
		}
	}

	if len(cd.PercentileRows) > 0 {
		writePDFSection(pdf, "Latency percentiles")
		for _, row := range cd.PercentileRows {
			writePDFRow(pdf, row.Label, row.Latency+" µs")
		}
	}

	if len(opts.Alerts) > 0 {
		writePDFSection(pdf, "Regression alerts")
		for _, a := range opts.Alerts {
			writePDFRow(pdf, a.Metric, fmt.Sprintf("%.1f%% vs baseline", a.DeltaPct))
		}
	}

	if len(opts.Insights) > 0 {
		writePDFSection(pdf, "Insights")
		for i, ins := range opts.Insights {
			if i >= 6 {
				break
			}
			label := ins.Type
			if label == "" {
				label = ins.Severity
			}
			writePDFRow(pdf, label, ins.Message)
		}
	}

	if cd.Summary != "" {
		writePDFSection(pdf, "Analysis")
		pdf.SetFont("Arial", "", 9)
		pdf.MultiCell(0, 4.5, strings.TrimSpace(cd.Summary), "", "L", false)
	}

	pdf.SetY(-12)
	pdf.SetFont("Arial", "I", 8)
	pdf.SetTextColor(120, 120, 120)
	pdf.CellFormat(0, 5, fmt.Sprintf("Run ID %s · HTML report has full charts and interval tables", run.RunID), "", 0, "C", false, 0, "")

	if err := pdf.OutputFileAndClose(outPath); err != nil {
		return err
	}
	fmt.Printf("pdf written: %s\n", outPath)
	return nil
}

func writePDFSection(pdf *fpdf.Fpdf, title string) {
	pdf.Ln(3)
	pdf.SetFont("Arial", "B", 11)
	pdf.SetFillColor(240, 244, 248)
	pdf.CellFormat(0, 7, title, "", 1, "L", true, 0, "")
	pdf.SetFont("Arial", "", 9)
}

func writePDFRow(pdf *fpdf.Fpdf, key, value string) {
	key = truncatePDF(key, 42)
	value = truncatePDF(value, 72)
	pdf.CellFormat(62, 5.5, key, "", 0, "L", false, 0, "")
	pdf.CellFormat(0, 5.5, value, "", 1, "L", false, 0, "")
}

func truncatePDF(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}
