package report

import (
	"path/filepath"

	"github.com/pratham-vishk/stratabench/internal/analyst"
	"github.com/pratham-vishk/stratabench/internal/baseline"
	"github.com/pratham-vishk/stratabench/internal/export"
	"github.com/pratham-vishk/stratabench/internal/paths"
	"github.com/pratham-vishk/stratabench/internal/schema"
)

// Artifacts holds paths to generated report files.
type Artifacts struct {
	HTML  string
	JSON  string
	Excel string
	PDF   string
}

// WriteRunArtifacts generates HTML report card, JSON, Excel, and PDF for a run.
func WriteRunArtifacts(run *schema.RunResult, opts Options) (Artifacts, error) {
	dir := paths.ReportsDir()
	a := Artifacts{
		HTML:  filepath.Join(dir, run.RunID+".html"),
		JSON:  filepath.Join(dir, run.RunID+".json"),
		Excel: filepath.Join(dir, run.RunID+".xlsx"),
		PDF:   filepath.Join(dir, run.RunID+".pdf"),
	}
	if err := WriteHTMLWithOptions(run, opts, a.HTML); err != nil {
		return a, err
	}
	if err := export.WriteJSON(run, a.JSON); err != nil {
		return a, err
	}
	if err := export.WriteExcel(run, a.Excel); err != nil {
		return a, err
	}
	if err := WritePDFWithOptions(run, opts, a.PDF); err != nil {
		return a, err
	}
	return a, nil
}

// OptionsFromAnalysis builds report options from analyst output and regression alerts.
func OptionsFromAnalysis(insights []analyst.Insight, summary string, alerts []baseline.Alert) Options {
	return Options{Insights: insights, Summary: summary, Alerts: alerts}
}

// OptionsFromRun creates minimal options for a run without analysis.
func OptionsFromRun(_ *schema.RunResult) Options {
	return Options{}
}
