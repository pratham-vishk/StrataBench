package report

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"

	"github.com/pratham-vishk/stratabench/internal/schema"
)

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>StrataBench Report — {{.RunID}}</title>
  <style>
    body { font-family: system-ui, sans-serif; margin: 2rem; background: #0f1419; color: #e7ecf3; }
    h1 { color: #5eead4; }
    .meta { color: #94a3b8; margin-bottom: 1.5rem; }
    table { border-collapse: collapse; width: 100%; max-width: 720px; }
    th, td { border: 1px solid #334155; padding: 0.5rem 0.75rem; text-align: left; }
    th { background: #1e293b; }
    .ok { color: #4ade80; }
    .warn { color: #fbbf24; }
    .badge { display: inline-block; padding: 0.15rem 0.5rem; border-radius: 4px; background: #1e3a5f; font-size: 0.85rem; }
  </style>
</head>
<body>
  <h1>StrataBench Report</h1>
  <p class="meta">
    Run <code>{{.RunID}}</code> · Profile <strong>{{.Profile}}</strong> ·
    Engine <span class="badge">{{.Engine}}</span>
    {{if .Mock}}<span class="badge">MOCK</span>{{end}}
  </p>
  <h2>Validation</h2>
  <p class="{{if .Validation.Passed}}ok{{else}}warn{{end}}">
    {{if .Validation.Passed}}Passed{{else}}Failed{{end}} — checked: {{range $i, $r := .Validation.RulesChecked}}{{if $i}}, {{end}}{{$r}}{{end}}
  </p>
  {{if .Validation.Errors}}<ul>{{range .Validation.Errors}}<li class="warn">{{.}}</li>{{end}}</ul>{{end}}
  <h2>Results</h2>
  <table>
    <tr><th>Metric</th><th>Value</th></tr>
    <tr><td>IOPS</td><td>{{printf "%.0f" .Results.IOPS}}</td></tr>
    <tr><td>Throughput (MB/s)</td><td>{{printf "%.2f" .Results.ThroughputMBps}}</td></tr>
    <tr><td>Latency p50 (µs)</td><td>{{printf "%.1f" .Results.LatencyUS.P50}}</td></tr>
    <tr><td>Latency p95 (µs)</td><td>{{printf "%.1f" .Results.LatencyUS.P95}}</td></tr>
    <tr><td>Latency p99 (µs)</td><td>{{printf "%.1f" .Results.LatencyUS.P99}}</td></tr>
    <tr><td>Latency p99.9 (µs)</td><td>{{printf "%.1f" .Results.LatencyUS.P999}}</td></tr>
    <tr><td>CPU %</td><td>{{printf "%.1f" .Results.CPUPercent}}</td></tr>
  </table>
  <h2>Target</h2>
  <table>
    <tr><th>Layer</th><td>{{.Layer}}</td></tr>
    <tr><th>Target</th><td>{{.Target.Device}}</td></tr>
    <tr><th>Duration</th><td>{{.Workload.DurationSec}}s (ramp {{.Workload.RampTimeSec}}s)</td></tr>
  </table>
</body>
</html>`

func WriteHTML(run *schema.RunResult, outPath string) error {
	tmpl, err := template.New("report").Parse(htmlTemplate)
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
	if err := tmpl.Execute(f, run); err != nil {
		return err
	}
	fmt.Printf("report written: %s\n", outPath)
	return nil
}
