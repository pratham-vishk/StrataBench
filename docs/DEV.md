# StrataBench Development Guide

## Prerequisites

- Go 1.22+
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
./bin/stratabench compare --run-id <a> --run-id-b <b>
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
curl http://localhost:8080/metrics
```

Grafana dashboard: `deploy/grafana/stratabench-dashboard.json`

## Distributed mode (Phase 2)

On each client VM:

```bash
./bin/stratabench-agent
# listens on :7777 (STRATABENCH_AGENT_LISTEN to override)
```

From coordinator:

```bash
./bin/stratabench run --profile ssd-random-4k --target /dev/nvme0n1 --mock \
  --clients 10.0.1.1:7777,10.0.1.2:7777,10.0.1.3:7777
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
  stratabench.db    # SQLite run history
  reports/          # HTML + JSON reports
  work/             # fio job files, temp data
```

## Mock mode

Use `--mock` on Windows or when hardware is unavailable.

## Environment variables

| Variable | Purpose |
|----------|---------|
| `STRATABENCH_ROOT` | Repo root (auto-detected) |
| `STRATABENCH_MOCK_CACHE_BYTES` | Override cache size for validation |
| `STRATABENCH_AGENT_LISTEN` | Agent bind address (default `:7777`) |
| `WARP_ACCESS_KEY` / `WARP_SECRET_KEY` | S3 credentials for Warp |
| `OLLAMA_URL` | Ollama API base URL (default `http://localhost:11434`) |
| `OLLAMA_MODEL` | Model for planner (default `llama3.2`) |

## Agentic loop

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
