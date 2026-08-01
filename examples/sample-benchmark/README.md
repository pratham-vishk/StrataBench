# Sample reports (HTML only)

Grafana-style operations dashboards — no Excel.

## Generate all samples

```powershell
make samples
```

Runs **sequentially** (not parallel):
1. **Base benchmark** → `base-benchmark.html`
2. **Candidate benchmark** → `candidate-benchmark.html`
3. **Compare** (overlay charts) → `compare-sample.html`
4. SBK import → `sample-result.html`
5. S3 PUT → `s3-put-sample.html`

## Single commands

```powershell
# One benchmark HTML
stratabench sample --open-report

# Base → candidate → compare (sequential)
stratabench sample-compare --open-report
```

## Compare workflow

```
base run (mock)     ──► base-benchmark.html
       │
       ▼  (must finish first)
candidate run       ──► candidate-benchmark.html
       │
       ▼
compare report      ──► compare-sample.html  (Grafana overlay: base vs candidate)
```

Branch/git compare (`compare branches`) is separate — still sequential per branch checkout.
