package report

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"runtime"

	"github.com/pratham-vishk/stratabench/internal/analyst"
	"github.com/pratham-vishk/stratabench/internal/schema"
)

// WriteHTML writes a visual report card for a run.
func WriteHTML(run *schema.RunResult, outPath string) error {
	return WriteHTMLWithOptions(run, Options{}, outPath)
}

// WriteHTMLWithInsights writes a report card with analyst output.
func WriteHTMLWithInsights(run *schema.RunResult, insights []analyst.Insight, summary, outPath string) error {
	return WriteHTMLWithOptions(run, Options{Insights: insights, Summary: summary}, outPath)
}

// WriteHTMLWithOptions writes the full report card.
func WriteHTMLWithOptions(run *schema.RunResult, opts Options, outPath string) error {
	cd, err := buildCardData(run, opts)
	if err != nil {
		return err
	}
	return writeCard(cd, outPath)
}

func writeCard(data CardData, outPath string) error {
	tmpl, err := template.New("card").Parse(cardHTMLTemplate + chartScript + chartScriptEnd)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := tmpl.Execute(f, data); err != nil {
		return err
	}
	fmt.Printf("report written: %s\n", outPath)
	return nil
}

// OpenInBrowser opens the report file in the default browser (best-effort).
func OpenInBrowser(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", "", abs}
	case "darwin":
		cmd = "open"
		args = []string{abs}
	default:
		cmd = "xdg-open"
		args = []string{abs}
	}
	return execOpen(cmd, args...)
}

// execOpen is overridden in tests.
var execOpen = func(name string, arg ...string) error {
	// use os/exec without importing in signature for testability - import exec in open_unix.go style
	return openBrowser(name, arg...)
}
