# Live monitoring

StrataBench exposes run progress while benchmarks execute — useful for long multi-node or async API runs.

## CLI

```bash
# Start async run and watch until done
stratabench run --profile s3-cluster-put-get --target 10.0.1.10:9000 \
  --clients 10.0.1.1:7777,10.0.1.2:7777 --mock --async --watch

# Or start async, then watch separately
stratabench run --profile nvme-random-oltp --target /dev/nvme0n1 --async --mock --skip-validate
stratabench watch --run-id <uuid>
```

## REST API

```bash
# Start background run
curl -X POST http://localhost:8080/api/v1/runs \
  -H "Content-Type: application/json" \
  -d '{"profile":"nvme-random-oltp","target":"/dev/null","mock":true,"async":true,"skip_validate":true}'

# Poll progress (while running)
curl http://localhost:8080/api/v1/runs/<uuid>/progress

# Fetch result (when completed)
curl http://localhost:8080/api/v1/runs/<uuid>
```

## Prometheus + Grafana

Start the API (`stratabench-api` or `docker compose up api`) and scrape `http://localhost:8080/metrics`.

| Metric | Description |
|--------|-------------|
| `stratabench_run_assignment_progress` | 0–1 fraction of topology assignments done |
| `stratabench_run_assignments_total` | Total assignments for active runs |
| `stratabench_iops` | IOPS from completed runs |
| `stratabench_runs_total` | Completed run counter |

Import `deploy/grafana/stratabench-dashboard.json` for a dashboard with live assignment progress.

## MCP

- `stratabench_run` with `"async": true` → returns `run_id` immediately
- `stratabench_run_progress` with `"run_id"` → phase, assignments completed

## SSE progress stream

For dashboards or scripts that want push updates:

```bash
curl -N http://localhost:8080/api/v1/runs/<uuid>/stream
```

Events: `progress` while running, `done` when completed/failed.

## Limitations

- Progress tracks **topology assignments** (client×target jobs), not per-second fio/Warp interval samples.
- Interval time-series appear in HTML reports **after** the run completes.
- For thermal/SMART live monitoring, use host tools alongside StrataBench (`smartctl`, `nvme`, etc.).
