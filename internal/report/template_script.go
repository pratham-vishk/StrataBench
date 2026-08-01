package report

const chartScript = `
const CH = {{.ChartsJS}};
const palette = ['#14b8a6','#38bdf8','#fbbf24','#f472b6','#a78bfa','#4ade80','#fb923c','#818cf8','#e879f9','#34d399'];
const gridColor = '#1e2836', tickColor = '#8b9cb3';
const chartInstances = [];
const baseOpts = {
  responsive: true, maintainAspectRatio: false,
  plugins: { legend: { labels: { color: tickColor, boxWidth: 12 } } },
  scales: {
    y: { beginAtZero: true, grid: { color: gridColor }, ticks: { color: tickColor } },
    x: { grid: { display: false }, ticks: { color: tickColor, maxRotation: 45, autoSkip: true, maxTicksLimit: 18 } }
  }
};
function lineDS(d, idx) {
  const c = palette[idx % palette.length];
  return {
    label: d.label, data: d.data, yAxisID: d.yAxisID || 'y',
    borderColor: c, backgroundColor: d.fill ? c + '33' : c + '18',
    borderWidth: 2, borderDash: d.dashed ? [6,4] : [],
    pointRadius: d.fill ? 0 : 2, tension: 0.3,
    fill: d.fill ? 'origin' : false
  };
}
function mkLine(id, labels, datasets, hideLegend, dualAxis) {
  const el = document.getElementById(id); if (!el) return;
  const ds = datasets.map((d,i) => lineDS(d,i));
  const scales = { x: baseOpts.scales.x, y: { ...baseOpts.scales.y, position: 'left' } };
  if (dualAxis) scales.y1 = { beginAtZero: true, position: 'right', grid: { drawOnChartArea: false }, ticks: { color: tickColor } };
  chartInstances.push(new Chart(el, { type: 'line', data: { labels, datasets: ds },
    options: { ...baseOpts, plugins: { legend: { display: !hideLegend, labels: { color: tickColor } } }, scales } }));
}
function mkBar(id, labels, datasets, stacked) {
  const el = document.getElementById(id); if (!el) return;
  chartInstances.push(new Chart(el, { type: 'bar', data: { labels, datasets: datasets.map((d,i) => ({
    ...d, backgroundColor: palette[i % palette.length] + 'bb', borderRadius: 4
  })) }, options: { ...baseOpts, scales: {
    x: { ...baseOpts.scales.x, stacked }, y: { ...baseOpts.scales.y, stacked } } } }));
}
Object.entries(CH.charts || {}).forEach(([id, spec]) => {
  if (spec.kind === 'bar') {
    mkBar(id, spec.labels, spec.datasets, spec.stacked);
  } else {
    mkLine(id, spec.labels, spec.datasets, spec.hideLegend, spec.dualAxis);
  }
});
window.addEventListener('resize', () => chartInstances.forEach(c => c.resize()));
(function(){
  const links = document.querySelectorAll('#sideNav a[data-section]');
  const sections = [...links].map(a => document.getElementById(a.dataset.section)).filter(Boolean);
  const onScroll = () => {
    let cur = sections[0]?.id || '';
    for (const s of sections) {
      if (s.getBoundingClientRect().top <= 120) cur = s.id;
    }
    links.forEach(a => a.classList.toggle('active', a.dataset.section === cur));
  };
  window.addEventListener('scroll', onScroll, {passive:true});
  onScroll();
})();
`

const chartScriptEnd = `
</script>
</body>
</html>`
