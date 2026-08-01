package report

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pratham-vishk/stratabench/internal/compare"
	"github.com/pratham-vishk/stratabench/internal/paths"
	"github.com/pratham-vishk/stratabench/internal/provenance"
	"github.com/pratham-vishk/stratabench/internal/schema"
	"github.com/pratham-vishk/stratabench/internal/version"
)

const compareHTMLTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>StrataBench — Benchmark comparison</title>
  <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap" rel="stylesheet">
  <script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.1/dist/chart.umd.min.js"></script>
  <style>
    :root{--bg:#080c10;--surface:#0f1419;--card:#151b24;--border:#2a3548;--text:#f0f4f8;--muted:#8b9cb3;
      --accent:#14b8a6;--good:#4ade80;--bad:#f87171;--sidebar-w:240px;--gap:1.5rem;--section-gap:3.5rem}
    *{box-sizing:border-box;margin:0;padding:0}
    body{font-family:'Inter',system-ui,sans-serif;background:var(--bg);color:var(--text);line-height:1.6}
    .sidebar{position:fixed;left:0;top:0;bottom:0;width:var(--sidebar-w);background:var(--surface);border-right:1px solid var(--border);padding:1.5rem 1rem;overflow-y:auto}
    .sidebar h2{font-size:.95rem;color:var(--accent)} .sidebar p{font-size:.75rem;color:var(--muted)}
    .nav a{display:block;padding:.5rem .75rem;color:var(--muted);text-decoration:none;font-size:.82rem;border-radius:6px;margin-bottom:2px}
    .nav a:hover{background:var(--card);color:var(--text)}
    main{margin-left:var(--sidebar-w);padding:2rem 2.5rem 4rem;max-width:1200px}
    h1{font-size:1.6rem;margin-bottom:.35rem}
    .subtitle{color:var(--muted);margin-bottom:2rem}
    .verdict{padding:1.15rem 1.35rem;border-radius:12px;margin-bottom:2rem;border:1px solid var(--border)}
    .verdict.improved{border-color:#166534;background:#052e1622}
    .verdict.regressed{border-color:#7f1d1d;background:#450a0a22}
    .verdict.neutral{border-color:var(--border);background:var(--card)}
    .grid-2{display:grid;grid-template-columns:1fr 1fr;gap:var(--gap);margin-bottom:var(--section-gap)}
    .panel{background:var(--card);border:1px solid var(--border);border-radius:12px;padding:1.35rem;margin-bottom:var(--gap)}
    .panel.grafana{border-left:3px solid var(--accent)}
    .panel h3{font-size:.72rem;color:var(--muted);text-transform:uppercase;margin:0 0 1rem}
    .chart-box{height:360px;position:relative}
    table{width:100%;border-collapse:collapse;font-size:.82rem}
    th,td{padding:.55rem .7rem;border-bottom:1px solid var(--border);text-align:right}
    th:first-child,td:first-child{text-align:left}
    .good{color:var(--good)} .bad{color:var(--bad)} .neutral{color:var(--muted)}
    .report-links{display:flex;gap:1rem;margin:1rem 0 2rem;flex-wrap:wrap}
    .report-links a{color:var(--accent);text-decoration:none;padding:.5rem 1rem;border:1px solid var(--border);border-radius:8px;font-size:.85rem}
    .report-links a:hover{background:var(--card)}
    footer{margin-top:3rem;color:var(--muted);font-size:.72rem}
    a{color:var(--accent)}
    .note{color:var(--muted);font-size:.85rem;margin-bottom:1.5rem;padding:.75rem 1rem;background:var(--card);border-radius:8px;border:1px solid var(--border)}
  @media(max-width:900px){.grid-2{grid-template-columns:1fr}main{margin-left:0;padding:1rem}}
  </style>
</head>
<body>
<aside class="sidebar">
  <h2>StrataBench</h2><p>Benchmark compare</p>
  <nav class="nav" style="margin-top:1.5rem">
    <a href="#summary">Summary</a>
    <a href="#runs">Base vs Candidate</a>
    {{if .HasIntervals}}<a href="#grafana">Operations (Grafana)</a>{{end}}
    <a href="#charts">Summary charts</a>
    <a href="#metrics">All metrics</a>
  </nav>
</aside>
<main>
  <h1 id="summary">Benchmark comparison</h1>
  <p class="subtitle">{{.Profile}} · {{.Target}} · {{.OpsLabel}} · sequential base → candidate</p>
  <p class="note">Base benchmark ran first; candidate ran after (not parallel). Compare relies on the completed base run.</p>
  <div class="verdict {{.Diff.Verdict}}">
    <strong>{{title .Diff.Verdict}}</strong> — {{.Diff.Summary}}
  </div>
  {{if or .BaseReportLink .HeadReportLink}}
  <div class="report-links">
    {{if .BaseReportLink}}<a href="{{.BaseReportLink}}">→ Base benchmark report (full Grafana dashboard)</a>{{end}}
    {{if .HeadReportLink}}<a href="{{.HeadReportLink}}">→ Candidate benchmark report (full Grafana dashboard)</a>{{end}}
  </div>
  {{end}}

  <section id="runs" style="margin-bottom:var(--section-gap)">
    <div class="grid-2">
      <div class="panel"><h3>Base (baseline)</h3>
        <p><strong>{{.Diff.BaseLabel}}</strong></p>
        <p style="margin:.5rem 0;color:var(--muted)">Run <code>{{.Diff.BaseRunID}}</code></p>
        <p>{{.OpsLabel}} <strong>{{printf "%.0f" .Base.Results.IOPS}}</strong> · p99 <strong>{{printf "%.0f" .Base.Results.LatencyUS.P99}} µs</strong></p>
      </div>
      <div class="panel"><h3>Candidate (after base)</h3>
        <p><strong>{{.Diff.HeadLabel}}</strong></p>
        <p style="margin:.5rem 0;color:var(--muted)">Run <code>{{.Diff.HeadRunID}}</code></p>
        <p>{{.OpsLabel}} <strong>{{printf "%.0f" .Head.Results.IOPS}}</strong> · p99 <strong>{{printf "%.0f" .Head.Results.LatencyUS.P99}} µs</strong></p>
      </div>
    </div>
  </section>

  {{if .HasIntervals}}
  <section id="grafana" style="margin-bottom:var(--section-gap)">
    <h2 style="font-size:1.05rem;margin-bottom:1.25rem">Operations dashboard — base vs candidate overlay</h2>
    <div class="panel grafana"><h3>{{.OpsLabel}} over time</h3><div class="chart-box"><canvas id="grafanaOps"></canvas></div></div>
    <div class="panel grafana"><h3>Throughput (MB/s) over time</h3><div class="chart-box"><canvas id="grafanaMbps"></canvas></div></div>
    <div class="panel grafana"><h3>Latency (avg µs) over time</h3><div class="chart-box"><canvas id="grafanaLat"></canvas></div></div>
  </section>
  {{end}}

  <section id="charts" style="margin-bottom:var(--section-gap)">
    <div class="panel"><h3>Throughput comparison</h3><div class="chart-box"><canvas id="cmpThroughput"></canvas></div></div>
    <div class="panel"><h3>Latency comparison (p50 / p99 / max)</h3><div class="chart-box"><canvas id="cmpLatency"></canvas></div></div>
    <div class="panel"><h3>Candidate vs base delta (%)</h3><div class="chart-box"><canvas id="cmpDelta"></canvas></div></div>
  </section>

  <section id="metrics">
    <div class="panel">
      <h3>Metric comparison (candidate vs base)</h3>
      <table>
        <tr><th>Metric</th><th>Base</th><th>Candidate</th><th>Δ%</th><th></th></tr>
        {{range .Diff.Metrics}}{{if or (gt .Base 0.0) (gt .Head 0.0)}}
        <tr>
          <td>{{.Name}}</td><td>{{printf "%.2f" .Base}}</td><td>{{printf "%.2f" .Head}}</td>
          <td>{{printf "%+.1f" .DeltaPct}}%</td>
          <td class="{{.Better}}">{{.Better}}</td>
        </tr>{{end}}{{end}}
      </table>
    </div>
  </section>
  <footer>StrataBench · {{.GeneratedAt}} · compare report (sequential benchmarks)</footer>
</main>
<script>
const cmp = {{.ChartsJS}};
const palette = ['#14b8a6','#38bdf8','#fbbf24','#f472b6'];
const tick = '#8b9cb3', grid = '#1e2836';
const baseOpts = {responsive:true,maintainAspectRatio:false,
  plugins:{legend:{labels:{color:tick}}},
  scales:{y:{beginAtZero:true,grid:{color:grid},ticks:{color:tick}},x:{grid:{display:false},ticks:{color:tick,maxRotation:45}}}};
function overlayChart(id, labels, baseData, headData, label, unit) {
  const el = document.getElementById(id); if (!el || !labels.length) return;
  new Chart(el, {type:'line', data:{labels, datasets:[
    {label:'Base '+label, data:baseData, borderColor:palette[0], backgroundColor:palette[0]+'18', borderWidth:2, tension:0.3, pointRadius:2},
    {label:'Candidate '+label, data:headData, borderColor:palette[2], backgroundColor:palette[2]+'18', borderWidth:2, borderDash:[6,4], tension:0.3, pointRadius:2}
  ]}, options:baseOpts});
}
if (cmp.hasIntervals) {
  overlayChart('grafanaOps', cmp.intervalLabels, cmp.baseIOPSOver, cmp.headIOPSOver, cmp.opsLabel, '');
  overlayChart('grafanaMbps', cmp.intervalLabels, cmp.baseMbpsOver, cmp.headMbpsOver, 'MB/s', '');
  overlayChart('grafanaLat', cmp.intervalLabels, cmp.baseLatOver, cmp.headLatOver, 'µs', '');
}
new Chart(document.getElementById('cmpThroughput'), {type:'bar',data:{
  labels:['Base','Candidate'],
  datasets:[
    {label:cmp.opsLabel,data:[cmp.baseIOPS,cmp.headIOPS],backgroundColor:palette[0]+'bb'},
    {label:'MB/s',data:[cmp.baseMBps,cmp.headMBps],backgroundColor:palette[1]+'bb'}
  ]},options:baseOpts});
new Chart(document.getElementById('cmpLatency'), {type:'bar',data:{
  labels:['p50','p99','max'],
  datasets:[
    {label:'Base µs',data:[cmp.baseP50,cmp.baseP99,cmp.baseMax],backgroundColor:palette[0]+'bb'},
    {label:'Candidate µs',data:[cmp.headP50,cmp.headP99,cmp.headMax],backgroundColor:palette[2]+'bb'}
  ]},options:baseOpts});
new Chart(document.getElementById('cmpDelta'), {type:'bar',data:{
  labels:cmp.deltaLabels,
  datasets:[{label:'Δ%',data:cmp.deltaPcts,backgroundColor:cmp.deltaPcts.map(v=>v>=0?palette[0]+'bb':palette[3]+'bb')}]
},options:{...baseOpts,plugins:{legend:{display:false}}}});
</script>
</body>
</html>`

type comparePageData struct {
	Base, Head       *schema.RunResult
	Diff             compare.DiffResult
	Profile          string
	Target           string
	Version          string
	GeneratedAt      string
	OpsLabel         string
	HasIntervals     bool
	BaseReportLink   string
	HeadReportLink   string
	ChartsJS         template.JS
}

type compareChartsPayload struct {
	OpsLabel       string    `json:"opsLabel"`
	BaseIOPS       float64   `json:"baseIOPS"`
	HeadIOPS       float64   `json:"headIOPS"`
	BaseMBps       float64   `json:"baseMBps"`
	HeadMBps       float64   `json:"headMBps"`
	BaseP50        float64   `json:"baseP50"`
	BaseP99        float64   `json:"baseP99"`
	BaseMax        float64   `json:"baseMax"`
	HeadP50        float64   `json:"headP50"`
	HeadP99        float64   `json:"headP99"`
	HeadMax        float64   `json:"headMax"`
	DeltaLabels    []string  `json:"deltaLabels"`
	DeltaPcts      []float64 `json:"deltaPcts"`
	HasIntervals   bool      `json:"hasIntervals"`
	IntervalLabels []string  `json:"intervalLabels"`
	BaseIOPSOver   []float64 `json:"baseIOPSOver"`
	HeadIOPSOver   []float64 `json:"headIOPSOver"`
	BaseMbpsOver   []float64 `json:"baseMbpsOver"`
	HeadMbpsOver   []float64 `json:"headMbpsOver"`
	BaseLatOver    []float64 `json:"baseLatOver"`
	HeadLatOver    []float64 `json:"headLatOver"`
}

func buildCompareCharts(base, head *schema.RunResult, diff compare.DiffResult) (compareChartsPayload, error) {
	lbl := workloadLabels(base).OpsRate
	var labels []string
	var pcts []float64
	for _, m := range diff.Metrics {
		if m.Name == "IOPS" || m.Name == "Throughput (MB/s)" || strings.HasPrefix(m.Name, "Latency p") {
			labels = append(labels, m.Name)
			pcts = append(pcts, m.DeltaPct)
		}
		if len(labels) >= 8 {
			break
		}
	}
	payload := compareChartsPayload{
		OpsLabel: lbl,
		BaseIOPS: base.Results.IOPS, HeadIOPS: head.Results.IOPS,
		BaseMBps: base.Results.ThroughputMBps, HeadMBps: head.Results.ThroughputMBps,
		BaseP50: base.Results.LatencyUS.P50, BaseP99: base.Results.LatencyUS.P99, BaseMax: base.Results.LatencyUS.Max,
		HeadP50: head.Results.LatencyUS.P50, HeadP99: head.Results.LatencyUS.P99, HeadMax: head.Results.LatencyUS.Max,
		DeltaLabels: labels, DeltaPcts: pcts,
	}
	biv, hiv := base.Results.Intervals, head.Results.Intervals
	if len(biv) > 0 && len(hiv) > 0 {
		n := len(biv)
		if len(hiv) < n {
			n = len(hiv)
		}
		payload.HasIntervals = true
		for i := 0; i < n; i++ {
			payload.IntervalLabels = append(payload.IntervalLabels, fmt.Sprintf("T%d", i+1))
			payload.BaseIOPSOver = append(payload.BaseIOPSOver, biv[i].IOPS)
			payload.HeadIOPSOver = append(payload.HeadIOPSOver, hiv[i].IOPS)
			payload.BaseMbpsOver = append(payload.BaseMbpsOver, biv[i].ThroughputMBps)
			payload.HeadMbpsOver = append(payload.HeadMbpsOver, hiv[i].ThroughputMBps)
			payload.BaseLatOver = append(payload.BaseLatOver, biv[i].AvgLatencyUS)
			payload.HeadLatOver = append(payload.HeadLatOver, hiv[i].AvgLatencyUS)
		}
	}
	return payload, nil
}

// WriteCompareHTML writes a comparison report. Optional reportLinks: [baseHTML, candidateHTML].
func WriteCompareHTML(base, head *schema.RunResult, diff compare.DiffResult, outPath string, reportLinks ...string) error {
	if outPath == "" {
		outPath = compare.CompareReportPath(base.RunID, head.RunID)
	}
	if !filepath.IsAbs(outPath) {
		outPath = filepath.Join(paths.RepoRoot(), outPath)
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	payload, err := buildCompareCharts(base, head, diff)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	tmpl := template.Must(template.New("compare").Funcs(template.FuncMap{
		"title": func(s string) string {
			if s == "" {
				return s
			}
			return strings.ToUpper(s[:1]) + s[1:]
		},
	}).Parse(compareHTMLTemplate))
	opsLabel := workloadLabels(base).OpsRate
	data := comparePageData{
		Base: base, Head: head, Diff: diff,
		Profile: diff.Profile, Target: diff.Target,
		Version:      version.Version,
		GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
		OpsLabel:     opsLabel,
		HasIntervals: payload.HasIntervals,
		ChartsJS:     template.JS(raw),
	}
	if len(reportLinks) >= 1 {
		data.BaseReportLink = reportLinks[0]
	}
	if len(reportLinks) >= 2 {
		data.HeadReportLink = reportLinks[1]
	}
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := tmpl.Execute(f, data); err != nil {
		return err
	}
	fmt.Printf("compare report written: %s\n", outPath)
	return nil
}

// WriteCompareArtifacts generates compare HTML plus individual run reports.
func WriteCompareArtifacts(base, head *schema.RunResult, diff compare.DiffResult) (string, error) {
	baseProv := provenance.Label(base.Provenance)
	headProv := provenance.Label(head.Provenance)
	_ = WriteHTMLOnly(base, Options{Summary: "Base — " + baseProv}, filepath.Join(paths.ReportsDir(), base.RunID+".html"))
	_ = WriteHTMLOnly(head, Options{Summary: "Candidate — " + headProv}, filepath.Join(paths.ReportsDir(), head.RunID+".html"))
	out := compare.CompareReportPath(base.RunID, head.RunID)
	if !filepath.IsAbs(out) {
		out = filepath.Join(paths.ReportsDir(), filepath.Base(out))
	}
	if err := WriteCompareHTML(base, head, diff, out); err != nil {
		return "", err
	}
	return out, nil
}
