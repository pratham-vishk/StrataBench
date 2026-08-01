package report

// HTML report — fixed sidebar, spacious layout, dynamic chart sections.
const cardHTMLTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>StrataBench — {{.Run.Profile}}</title>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=JetBrains+Mono:wght@400;500&display=swap" rel="stylesheet">
  <script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.1/dist/chart.umd.min.js"></script>
  <style>
    :root {
      --bg:#080c10; --surface:#0f1419; --card:#151b24; --card2:#1a2230;
      --border:#2a3548; --border2:#1e2836; --text:#f0f4f8; --muted:#8b9cb3;
      --accent:#14b8a6; --accent2:#0d9488; --good:#4ade80; --bad:#f87171; --warn:#fbbf24;
      --sidebar-w:240px; --radius:12px; --gap:1.25rem; --section-gap:3.5rem;
    }
    *{box-sizing:border-box;margin:0;padding:0}
    html{scroll-behavior:smooth}
    body{font-family:'Inter',system-ui,sans-serif;background:var(--bg);color:var(--text);line-height:1.6;font-size:14px;overflow-x:hidden}
    .sidebar{position:fixed;left:0;top:0;bottom:0;width:var(--sidebar-w);background:var(--surface);border-right:1px solid var(--border2);display:flex;flex-direction:column;z-index:50}
    .sidebar-brand{padding:1.5rem 1.25rem 1rem;border-bottom:1px solid var(--border2);flex-shrink:0}
    .sidebar-brand .logo{font-weight:700;font-size:1rem;color:var(--accent)}
    .sidebar-brand .sub{font-size:.72rem;color:var(--muted);margin-top:.25rem}
    .nav{flex:1;overflow-y:auto;padding:.75rem .65rem 1.5rem}
    .nav .nav-label{font-size:.65rem;text-transform:uppercase;letter-spacing:.08em;color:var(--muted);padding:.5rem .75rem .35rem;font-weight:600}
    .nav a{display:block;padding:.5rem .85rem;border-radius:8px;color:var(--muted);text-decoration:none;font-size:.8rem;font-weight:500;line-height:1.35;margin-bottom:2px;transition:background .12s,color .12s}
    .nav a:hover{background:var(--card);color:var(--text)}
    .nav a.active{background:#14b8a618;color:var(--accent);border-left:2px solid var(--accent);padding-left:calc(.85rem - 2px)}
    main{margin-left:var(--sidebar-w);min-height:100vh;padding:2rem 2.5rem 4rem;max-width:1400px}
    .hero{margin-bottom:2rem}
    .hero h1{font-size:clamp(1.4rem,2.2vw,1.9rem);font-weight:700;letter-spacing:-.03em}
    .hero-meta{color:var(--muted);font-size:.9rem;margin-top:.5rem}
    .badges{display:flex;flex-wrap:wrap;gap:.5rem;margin-top:1.25rem}
    .badge{padding:.35rem .75rem;border-radius:999px;font-size:.72rem;font-weight:600;border:1px solid var(--border);background:var(--card)}
    .badge.pass{border-color:#166534;color:var(--good)} .badge.fail{border-color:#7f1d1d;color:var(--bad)}
    .badge.mock{border-color:#713f12;color:var(--warn)} .badge.engine{color:var(--accent);border-color:var(--accent2)}
    .badge.role-client{color:#60a5fa} .badge.role-target{color:#f472b6}
    .summary{background:linear-gradient(135deg,#0f1a24,#151b24);border:1px solid var(--border);border-left:3px solid var(--accent);padding:1.15rem 1.5rem;border-radius:var(--radius);margin:1.5rem 0;font-size:.9rem;color:#c8d5e3}
    .kpi-grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(170px,1fr));gap:var(--gap);margin:2rem 0}
    .kpi{background:var(--card);border:1px solid var(--border2);border-radius:var(--radius);padding:1.15rem 1.25rem}
    .kpi-label{font-size:.68rem;text-transform:uppercase;letter-spacing:.07em;color:var(--muted);font-weight:600}
    .kpi-value{font-size:1.5rem;font-weight:700;margin:.35rem 0;font-variant-numeric:tabular-nums}
    .kpi-unit{font-size:.82rem;color:var(--muted);font-weight:500}
    .kpi-hint{font-size:.7rem;color:var(--muted);margin-top:.2rem}
    .section{margin-top:var(--section-gap);scroll-margin-top:1.5rem}
    .section-head{display:flex;align-items:baseline;gap:.75rem;margin-bottom:1.25rem;padding-bottom:.75rem;border-bottom:1px solid var(--border2)}
    .section-head h2{font-size:1.05rem;font-weight:600}
    .section-head .count{font-size:.72rem;color:var(--muted);background:var(--card);padding:.2rem .55rem;border-radius:4px;border:1px solid var(--border2)}
    .panel{background:var(--card);border:1px solid var(--border2);border-radius:var(--radius);padding:1.35rem;margin-bottom:var(--gap)}
    #operations-dashboard .panel{border-left:3px solid var(--accent);background:linear-gradient(135deg,#0f1419 0%,#151b24 100%)}
    .panel h3{font-size:.72rem;margin:0 0 1rem;color:var(--muted);font-weight:600;text-transform:uppercase;letter-spacing:.06em}
    .charts-grid{display:flex;flex-direction:column;gap:var(--gap)}
    .chart-box{position:relative;height:340px;width:100%}
    .chart-box.tall{height:400px}
    .chart-box canvas{width:100%!important;height:100%!important}
    table{width:100%;border-collapse:collapse;font-size:.8rem}
    th,td{padding:.55rem .75rem;text-align:right;border-bottom:1px solid var(--border2)}
    th:first-child,td:first-child{text-align:left}
    th{color:var(--muted);font-weight:600;font-size:.72rem;text-transform:uppercase;background:var(--card2)}
    td{font-variant-numeric:tabular-nums;font-family:'JetBrains Mono',monospace;font-size:.78rem}
    .table-wrap{overflow-x:auto;border-radius:8px;border:1px solid var(--border2)}
    .kv td:first-child{color:var(--muted);width:38%;font-family:'Inter',sans-serif;font-weight:500}
    details.chart-section{border:1px solid var(--border2);border-radius:var(--radius);margin-bottom:var(--gap);background:var(--card)}
    details.chart-section>summary{cursor:pointer;padding:1rem 1.25rem;font-weight:600;font-size:.9rem;color:var(--text);list-style:none;display:flex;align-items:center;justify-content:space-between}
    details.chart-section>summary::-webkit-details-marker{display:none}
    details.chart-section>summary::after{content:'▾';color:var(--muted);font-size:.75rem}
    details.chart-section:not([open])>summary::after{content:'▸'}
    details.chart-section .charts-grid{padding:0 1.25rem 1.25rem}
    details.data-section{border:1px solid var(--border2);border-radius:var(--radius);margin-bottom:var(--gap);background:var(--card)}
    details.data-section>summary{cursor:pointer;padding:1rem 1.25rem;font-weight:600;color:var(--muted);list-style:none}
    details.data-section>summary::-webkit-details-marker{display:none}
    details.data-section .inner{padding:0 1.25rem 1.25rem}
    .insight{padding:.65rem 0;border-bottom:1px solid var(--border2);font-size:.88rem}
    .delta{font-size:.8rem;margin:.5rem 0 1rem}
    .delta.good{color:var(--good)} .delta.bad{color:var(--bad)} .delta.neutral{color:var(--muted)}
    footer{margin-top:4rem;padding-top:1.5rem;border-top:1px solid var(--border2);color:var(--muted);font-size:.72rem;display:flex;justify-content:space-between;flex-wrap:wrap;gap:.5rem}
    .mono{font-family:'JetBrains Mono',monospace;font-size:.75rem}
    .sidebar-toggle{display:none}
  @media(max-width:900px){
    .sidebar{transform:translateX(-100%);transition:transform .2s}
    .sidebar.open{transform:translateX(0)}
    main{margin-left:0;padding:1.25rem 1rem 3rem}
    .sidebar-toggle{display:block;position:fixed;top:.75rem;left:.75rem;z-index:60;background:var(--card);border:1px solid var(--border);color:var(--text);padding:.45rem .7rem;border-radius:8px;cursor:pointer}
  }
  </style>
</head>
<body>
<button class="sidebar-toggle" type="button" onclick="document.querySelector('.sidebar').classList.toggle('open')">☰</button>
<aside class="sidebar">
  <div class="sidebar-brand">
    <div class="logo">StrataBench</div>
    <div class="sub">Performance Report</div>
  </div>
  <nav class="nav" id="sideNav">
    <div class="nav-label">Jump to</div>
    {{range .NavSections}}<a href="#{{.ID}}" data-section="{{.ID}}">{{.Title}}</a>{{end}}
  </nav>
</aside>
<main>
  <header class="hero" id="overview">
    <h1>{{.Run.Profile}}</h1>
    <p class="hero-meta">{{.BenchmarkLabel}}{{if .OperationBadge}} · {{.OperationBadge}}{{end}} · {{.Run.Workload.DurationSec}}s · {{.ChartCount}} charts · StrataBench {{.Version}}</p>
    <div class="badges">
      {{if .ValidationOK}}<span class="badge pass">Validation passed</span>{{else}}<span class="badge fail">Validation failed</span>{{end}}
      {{if .Run.Mock}}<span class="badge mock">Mock data</span>{{end}}
      {{if .EngineLabel}}<span class="badge engine">{{.EngineLabel}}</span>{{end}}
      {{if .Run.Topology}}<span class="badge">{{.Run.Topology}}</span>{{end}}
    </div>
  </header>
  {{if .Summary}}<div class="summary">{{.Summary}}</div>{{end}}

  <div class="kpi-grid">
    {{range .KPIs}}
    <div class="kpi">
      <div class="kpi-label">{{.Label}}</div>
      <div class="kpi-value">{{.Value}}{{if .Unit}} <span class="kpi-unit">{{.Unit}}</span>{{end}}</div>
      <div class="kpi-hint">{{.Hint}}</div>
    </div>
    {{end}}
  </div>
  {{if .IOPSDelta}}<div class="delta {{.IOPSDeltaClass}}">{{.Labels.OpsRate}} vs baseline: {{.IOPSDelta}} · p99: <span class="{{.P99DeltaClass}}">{{.P99Delta}}</span></div>{{end}}

  <section class="section" id="overview-tables">
    <div class="section-head"><h2>Run summary</h2></div>
    <details class="data-section"><summary>Configuration &amp; metadata</summary><div class="inner table-wrap"><table class="kv">{{range .SummaryRows}}<tr><td>{{.Key}}</td><td>{{.Value}}</td></tr>{{end}}</table></div></details>
    <details class="data-section"><summary>Run timeline</summary><div class="inner table-wrap"><table>
      <tr><th>Node</th><th>Role</th><th>Target</th><th>Profile</th><th>Start</th><th>End</th><th>Duration</th></tr>
      {{range .DurationRows}}<tr><td>{{.Node}}</td><td>{{.Role}}</td><td>{{.Target}}</td><td>{{.Profile}}</td><td>{{.Start}}</td><td>{{.End}}</td><td>{{.Duration}}</td></tr>{{end}}
    </table></div></details>
    <details class="data-section"><summary>Aggregate metrics</summary><div class="inner table-wrap"><table class="kv">{{range .MetricRows}}<tr><td>{{.Key}}</td><td>{{.Value}}</td></tr>{{end}}</table></div></details>
  </section>

  {{if .HasTotals}}
  <section class="section" id="totals">
    <div class="section-head"><h2>Totals</h2><span class="count">volume &amp; records</span></div>
    <div class="panel table-wrap"><table class="kv">{{range .TotalRows}}<tr><td>{{.Key}}</td><td>{{.Value}}</td></tr>{{end}}</table></div>
  </section>
  {{end}}

  {{if .PercentileRows}}
  <section class="section" id="percentiles">
    <div class="section-head"><h2>Percentile matrix</h2><span class="count">{{len .PercentileRows}} points</span></div>
    <div class="panel table-wrap"><table class="perc-table">
      <tr><th>Percentile</th><th>Latency (µs)</th><th>Count</th></tr>
      {{range .PercentileRows}}<tr><td>{{.Label}}</td><td>{{.Latency}}</td><td>{{.Count}}</td></tr>{{end}}
    </table></div>
  </section>
  {{end}}

  {{range .ChartGroups}}
  <section class="section" id="{{if .ID}}{{.ID}}{{else}}{{.Title}}{{end}}">
    {{if .Collapsed}}
    <details class="chart-section">
      <summary><span>{{.Title}}</span><span class="count" style="font-size:.72rem;color:var(--muted)">{{len .Panels}} charts</span></summary>
      <div class="charts-grid">
        {{range .Panels}}<div class="panel"><h3>{{.Title}}</h3><div class="chart-box{{if .Tall}} tall{{end}}"><canvas id="{{.ID}}"></canvas></div></div>{{end}}
      </div>
    </details>
    {{else}}
    <div class="section-head"><h2>{{.Title}}</h2><span class="count">{{len .Panels}} charts</span></div>
    <div class="charts-grid">
      {{range .Panels}}<div class="panel"><h3>{{.Title}}</h3><div class="chart-box{{if .Tall}} tall{{end}}"><canvas id="{{.ID}}"></canvas></div></div>{{end}}
    </div>
    {{end}}
  </section>
  {{end}}

  <section class="section" id="nodes">
    <div class="section-head"><h2>Node matrix</h2><span class="count">{{len .NodeRows}} nodes</span></div>
    <div class="panel table-wrap"><table>
      <tr><th>Node</th><th>Role</th><th>{{.Labels.OpsRate}}</th><th>{{.Labels.ReadOp}}</th><th>{{.Labels.WriteOp}}</th><th>MB/s</th>
      <th>min</th><th>mean</th><th>p50</th><th>p75</th><th>p90</th><th>p95</th><th>p99</th><th>p99.9</th><th>p99.99</th><th>max</th></tr>
      {{range .NodeRows}}<tr>
        <td>{{.Label}}</td><td><span class="badge role-{{.Role}}">{{.Role}}</span></td>
        <td>{{.IOPS}}</td><td>{{.ReadIOPS}}</td><td>{{.WriteIOPS}}</td><td>{{.MBps}}</td>
        <td>{{.Min}}</td><td>{{.Mean}}</td><td>{{.P50}}</td><td>{{.P75}}</td><td>{{.P90}}</td><td>{{.P95}}</td>
        <td>{{.P99}}</td><td>{{.P999}}</td><td>{{.P9999}}</td><td>{{.Max}}</td>
      </tr>{{end}}
    </table></div>
  </section>

  {{if .HasIntervals}}
  <section class="section" id="intervals">
    <div class="section-head"><h2>Interval data</h2><span class="count">{{len .IntervalRows}} buckets · cluster aggregate</span></div>
    <div class="panel table-wrap"><table>
      <tr><th>#</th><th>Time</th><th>{{.Labels.OpsRate}}</th><th>{{.Labels.ReadOp}}</th><th>{{.Labels.WriteOp}}</th><th>MB/s</th><th>Avg µs</th><th>Min µs</th><th>Max µs</th><th>W timeout</th><th>R timeout</th></tr>
      {{range .IntervalRows}}<tr>
        <td>{{.Seq}}</td><td>{{.Timestamp}}</td><td>{{.IOPS}}</td><td>{{.ReadIOPS}}</td><td>{{.WriteIOPS}}</td>
        <td>{{.MBps}}</td><td>{{.Avg}}</td><td>{{.Min}}</td><td>{{.Max}}</td><td>{{.WTimeout}}</td><td>{{.RTimeout}}</td>
      </tr>{{end}}
    </table></div>
  </section>
  {{end}}

  {{if .NodeIntervalSections}}
  <section class="section" id="node-intervals">
    <div class="section-head"><h2>Per-node intervals</h2><span class="count">{{len .NodeIntervalSections}} series</span></div>
    {{range .NodeIntervalSections}}
    <details class="data-section">
      <summary><span>{{.Label}}</span> <span class="badge role-{{.Role}}">{{.Role}}</span> <span class="count" style="margin-left:.5rem">{{len .Rows}} buckets</span></summary>
      <div class="inner table-wrap"><table>
        <tr><th>#</th><th>Time</th><th>{{$.Labels.OpsRate}}</th><th>{{$.Labels.ReadOp}}</th><th>{{$.Labels.WriteOp}}</th><th>MB/s</th><th>Avg µs</th><th>Min µs</th><th>Max µs</th><th>W timeout</th><th>R timeout</th></tr>
        {{range .Rows}}<tr>
          <td>{{.Seq}}</td><td>{{.Timestamp}}</td><td>{{.IOPS}}</td><td>{{.ReadIOPS}}</td><td>{{.WriteIOPS}}</td>
          <td>{{.MBps}}</td><td>{{.Avg}}</td><td>{{.Min}}</td><td>{{.Max}}</td><td>{{.WTimeout}}</td><td>{{.RTimeout}}</td>
        </tr>{{end}}
      </table></div>
    </details>
    {{end}}
  </section>
  {{end}}

  {{if .Insights}}
  <section class="section" id="insights">
    <div class="section-head"><h2>Analyst insights</h2></div>
    <div class="panel">{{range .Insights}}<div class="insight {{.Severity}}">{{.Message}}</div>{{end}}</div>
  </section>
  {{end}}

  <footer>
    <span>StrataBench {{.Version}} · {{.GeneratedAt}}</span>
    <span class="mono">run {{.Run.RunID}}</span>
  </footer>
</main>
<script>
`
