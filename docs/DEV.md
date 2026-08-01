# StrataBench Development Guide

## Prerequisites

- Go 1.25+
- Linux or WSL2 for real `fio` benchmarks
- `fio` (optional): `sudo apt install fio`
- `warp` (optional): MinIO Warp for S3 profiles

## Build

```bash
make build
# builds bin/stratabench, bin/stratabench-agent, bin/stratabench-api
```

## Commands

```bash
./bin/stratabench profiles
./bin/stratabench plan "s3 cluster warp"
./bin/stratabench validate --profile nvme-random-oltp --cache-bytes 10737418240
./bin/stratabench run --profile ssd-random-4k --target /tmp/test --mock
./bin/stratabench runs
./bin/stratabench compare runs --run-id <a> --run-id-b <b>
./bin/stratabench plan "nvme oltp" --ollama
./bin/stratabench agent "s3 cluster read heavy" --target 10.0.1.10:9000 --mock
./bin/stratabench cross-layer --profiles nvme-random-oltp,s3-put-throughput --target /tmp/test --mock
./bin/stratabench import sbk results.csv
./bin/stratabench baseline set --run-id <uuid>
./bin/stratabench baseline show
./bin/stratabench baseline check --run-id <uuid>
./bin/stratabench export --run-id <uuid>
./bin/stratabench report --run-id <uuid>
```

## REST API (Phase 3)

```bash
./bin/stratabench-api
curl http://localhost:8080/api/v1/health
curl http://localhost:8080/api/v1/profiles
curl -X POST http://localhost:8080/api/v1/runs \
  -H 'Content-Type: application/json' \
  -d '{"profile":"ssd-random-4k","target":"/tmp/test","mock":true}'
curl -X POST http://localhost:8080/api/v1/validate \
  -H 'Content-Type: application/json' \
  -d '{"profile":"ssd-random-4k","target":"/tmp/test","mock":true}'
curl http://localhost:8080/api/v1/report/<run-id>
curl -X POST http://localhost:8080/api/v1/compare \
  -H 'Content-Type: application/json' \
  -d '{"run_id":"<a>","run_id_b":"<b>"}'
curl http://localhost:8080/metrics
```

# After a run, open the visual report card
./bin/stratabench report --run-id <uuid> --open
./bin/stratabench export excel --run-id <uuid>
./bin/stratabench export excel --profile nvme-random-oltp --last 10

Reports live in `.stratabench/reports/` as `.html`, `.xlsx`, and `.json`.
See [LAB-BOOTSTRAP.md](LAB-BOOTSTRAP.md) for cluster workflows.

## Distributed mode (Phase 2)

On each client VM:

```bash
export STRATABENCH_AGENT_TOKEN="your-secret"   # optional but recommended
./bin/stratabench-agent
# listens on :7777 (STRATABENCH_AGENT_LISTEN to override)
```

Set the same `STRATABENCH_AGENT_TOKEN` on the coordinator when using `--clients`.

From coordinator:

```bash
./bin/stratabench run --profile ssd-random-4k --target /dev/nvme0n1 --mock \
  --clients 10.0.1.1:7777,10.0.1.2:7777,10.0.1.3:7777 --topology pool
```

Native Warp coordinator mode (port **7761**, not agent 7777):

```bash
./bin/stratabench run --profile s3-cluster-put-get --target 10.0.1.10:9000 \
  --warp-clients 10.0.1.1:7761,10.0.1.2:7761
```

Aggregates sum IOPS/throughput; tail latency uses max across clients.

## Warp (S3)

```bash
export WARP_ACCESS_KEY=minioadmin
export WARP_SECRET_KEY=minioadmin
./bin/stratabench run --profile s3-cluster-put-get --target 10.0.1.10:9000
```

## Data layout

```
.stratabench/
  stratabench.db    # SQLite run history (or PostgreSQL when STRATABENCH_DATABASE_URL is set)
  reports/          # HTML + JSON reports
  work/             # fio job files, temp data
```

## Mock mode

Use `--mock` on Windows or when hardware is unavailable.

## Live run monitoring

```bash
# Watch a run (async API or long multi-node run)
./bin/stratabench watch --run-id <uuid>

# Prometheus gauges during run (scrape /metrics on API)
# stratabench_run_assignment_progress, stratabench_run_assignments_total
```

Import Grafana dashboard from `deploy/grafana/stratabench-dashboard.json`.

## Environment variables

| Variable | Purpose |
|----------|---------|
| `STRATABENCH_ROOT` | Repo root (auto-detected) |
| `STRATABENCH_MOCK_CACHE_BYTES` | Override cache size for validation |
| `STRATABENCH_AGENT_LISTEN` | Agent bind address (default `:7777`) |
| `STRATABENCH_AGENT_TOKEN` | Shared bearer token for agent HTTP API |
| `STRATABENCH_DATABASE_URL` | PostgreSQL DSN (optional; default is SQLite) |
| `STRATABENCH_AGENT_TLS_CERT` / `_KEY` | Agent HTTPS server certificate |
| `STRATABENCH_AGENT_TLS_CA` | CA for mutual TLS (server requires client certs when set) |
| `STRATABENCH_AGENT_TLS_CLIENT_CERT` / `_KEY` | Coordinator client cert for HTTPS agents |
| `GOSBENCH_SERVER_BIN` | Path to `gosbench-server` binary (default: search PATH) |
| `GOSBENCH_ACCESS_KEY` / `GOSBENCH_SECRET_KEY` | S3 credentials for GOSBench |
| `STRATABENCH_ENGINE_BIN` | Native engine binary (default: `stratabench-engine` on PATH) |
| `STRATABENCH_SBK_BRIDGE` | External SBK Python bridge executable |
| `WARP_ACCESS_KEY` / `WARP_SECRET_KEY` | S3 credentials for Warp |
| `OLLAMA_URL` | Ollama API base URL (default `http://localhost:11434`) |
| `OLLAMA_MODEL` | Model for planner (default `llama3.2`) |

## Hardware inventory

```bash
./bin/stratabench inventory collect
./bin/stratabench inventory list
```

Hardware snapshots are auto-saved on each benchmark run.

## VM guest benchmarks

For `vm-block` profiles, use SSH target format:

```bash
./bin/stratabench run --profile vm-disk-random --target root@10.0.1.20:/dev/vdb
```

Requires `ssh`/`scp` access to the guest with `fio` installed inside the VM.

## Kubernetes

```bash
kubectl apply -f deploy/k8s/
kubectl port-forward -n stratabench svc/stratabench-api 8080:8080
curl http://localhost:8080/api/v1/health
```

See `deploy/k8s/` for API Deployment, agent DaemonSet, and scheduled CronJob example.

## Docker Compose

```bash
docker compose up api    # :8080
docker compose up agent  # :7777
```

```bash
# Full lifecycle from natural language
./bin/stratabench agent "nvme oltp database" --target /dev/nvme0n1 --mock

# With Ollama planner (requires running `ollama serve`)
./bin/stratabench agent "s3 cluster read heavy AI workload" --target 10.0.1.10:9000 --ollama --model llama3.2

# Plan only
./bin/stratabench plan "afa multi lun flash" --ollama
```

## Tests

```bash
go test ./...
```

## Dell lab

See [DELL-LAB.md](DELL-LAB.md) for VM layout, ports, and distributed runs.
