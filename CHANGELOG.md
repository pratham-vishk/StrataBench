# Changelog

All notable changes to StrataBench are documented in this file.

## [0.8.0-rc12] - 2026-08-01

### Added
- **Warp live interval streaming** — parse stdout/stderr MiB/s + obj/s lines during runs; `--benchdata` + `warp analyze` for post-run interval CSV
- **Warp HTML report intervals** — completed S3 runs attach time-bucket series from benchdata analysis

### Changed
- Warp runner streams live samples when `OnInterval` is set (async watch / SSE / Prometheus)

## [0.8.0-rc11] - 2026-08-01

### Added
- **fio live interval tailing** — polls `write_iops_log` / `write_bw_log` during fio runs and emits `OnInterval` callbacks

### Changed
- fio runner uses start + log watch + wait when `OnInterval` is set (enables Prometheus/SSE live metrics for real block runs)
- SQLite store enables WAL + `busy_timeout` for concurrent async run polling

## [0.8.0-rc10] - 2026-08-01

### Added
- **Live interval streaming** — mock runs emit per-bucket IOPS/throughput/latency during execution
- **Prometheus live gauges** — `stratabench_live_iops`, `stratabench_live_throughput_mbps`, `stratabench_live_avg_latency_us`
- **SSE `interval` events** — `GET /api/v1/runs/{id}/stream` pushes time-bucket samples while running
- **`stratabench watch`** — shows live IOPS/MBps/latency when intervals are available

### Changed
- Progress API and MCP `stratabench_run_progress` include `latest_interval` and `interval_buckets`

## [0.8.0-rc9] - 2026-08-01

### Added
- **SSE progress stream** — `GET /api/v1/runs/{id}/stream` (Server-Sent Events until done)
- **`stratabench runs`** — shows `STATUS` column (running/completed/failed)
- **Docs** — updated site index, `crates/README.md`

## [0.8.0-rc8] - 2026-08-01

### Added
- **CLI `--async` / `--watch`** — background runs with live progress on terminal
- **Rust engine scaffold** — `crates/stratabench-engine` (`make build-rust`)
- **SBK bridge example** — `examples/sbk-bridge/sbk_bridge.py` + `.py` auto-invocation via `STRATABENCH_PYTHON`
- **MCP** — `stratabench_import_json` tool
- **Docs** — `docs/MONITORING.md` live monitoring guide

### Changed
- SBK runner tries `STRATABENCH_SBK_BRIDGE` after any native driver failure

## [0.8.0-rc7] - 2026-08-01

### Added
- **`stratabench-engine` stub** — Go reference binary implementing native engine contract (`make build-engine`)
- **`stratabench watch`** — CLI live progress until run completes
- **MCP** — `stratabench_run_progress` tool; `async` on `stratabench_run`
- **SBK Python bridge** — `STRATABENCH_SBK_BRIDGE` env for external Python runner (same JSON contract)
- **Grafana dashboard** — live `stratabench_run_assignment_progress` panel

### Changed
- Docker image and CI build include `stratabench-engine`

## [0.8.0-rc6] - 2026-08-01

### Added
- **Async API runs** — `POST /api/v1/runs` with `"async": true` returns `202 Accepted` and run ID
- **SBK JSON import** — `stratabench import sbk-json` and `importsbk.ParseJSON`
- **Native engine bridge** — `stratabench-engine` external binary contract (`STRATABENCH_ENGINE_BIN`)
- **Live Prometheus progress** — `stratabench_run_assignment_progress` gauge during in-flight runs
- **Docs** — `docs/NATIVE-ENGINE.md` for Rust engine integration

### Changed
- `engine: stratabench` invokes external binary when present; fails honestly otherwise

## [0.8.0-rc5] - 2026-08-01

### Added
- **PostgreSQL store** — set `STRATABENCH_DATABASE_URL` to use shared Postgres instead of SQLite
- **GOSBench engine** — `gosbench-server` wrapper with auto-generated YAML config; profile `s3-gosbench-write`
- **Agent mTLS** — optional TLS via `STRATABENCH_AGENT_TLS_*` env vars (server + client certs)
- **Run progress** — in-memory progress during multi-assignment runs; `GET /api/v1/runs/{id}/progress`

### Changed
- Orchestrator uses `store.OpenDefault` (Postgres or SQLite)
- Remote agent client configures mTLS transport when connecting over HTTPS

## [0.8.0-rc4] - 2026-08-01

### Added
- **Multi-node HTML reports** — per-node interval overlays, Grafana panels, and interval tables; cluster aggregate charts merge time buckets across nodes
- **REST API** — `POST /api/v1/validate`, `GET|POST /api/v1/report/{id}`, `POST /api/v1/compare`
- **MCP tools** — `stratabench_compare_runs`, `stratabench_report`, `stratabench_baseline_check`, `stratabench_export_json`; `stratabench_run` supports `clients`, `targets`, `topology`, `warp_clients`
- **Agent auth** — bearer token via `STRATABENCH_AGENT_TOKEN` on agent and remote client
- **CLI** — `--warp-clients` for native Warp coordinator mode (port 7761)
- **Run honesty** — engines fail without `--mock` when tools are missing; synthetic output flagged on non-mock runs
- **Tests** — aggregate interval merge, multi-node report, MCP extended tools, remote client, orchestrator, crosslayer analysis

### Changed
- Distributed runs record coordinator client rows for local multi-assignment topologies
- Remote agent runs respect `SkipValidate` from coordinator (not hardcoded)
- Docker image installs MinIO Warp; CI builds MCP and runs mock end-to-end smoke tests
- Docs aligned to 29 profiles; topology and architecture status tables updated

## [0.7.0-rc1] - 2026-08-01

### Added
- **Topology engine** — all client/server patterns: `single`, `pool`, `sweep`, `shard`, `matrix`
- CLI `--targets` and `--topology`; per-target results in `run.targets[]`
- Virtual HDD, NVMe passthrough, AFA, S3 RDMA profiles
- `vm_vdbench.go` for multi-LUN AFA inside VM guests
- `docs/TOPOLOGY.md`, rewritten README, updated docs site

### Changed
- Repository is **public**; GitHub Pages live
- 29 workload profiles across physical and virtual layers

## [0.6.0-rc1] - 2026-08-01

### Added
- **25 workload profiles** — full physical + virtual coverage for all engines
- VM profiles: `vm-disk-sequential`, `vm-nvme-oltp`, `vm-disk-stress`, `vm-file-*`, `vm-s3-*`, `vm-app-*`
- Physical profiles: `file-parallel-write`, `s3-mixed-workload`
- `vm-file` elbencho via SSH (`vm_elbencho.go`)
- `docs/ENGINE-COVERAGE.md` — engine × deployment matrix
- Example manifests for physical and virtual benchmarks
- Repository is now **public**; GitHub Pages enabled

### Changed
- `ssd-random-4k`, `nvme-max-stress` → real **fio** engine (was mock)
- `s3-put-throughput`, `s3-get-throughput` → real **warp** engine (was mock)
- `DELL-LAB-VALIDATION.md` covers all 7 engines on physical and virtual targets

## [0.5.0-rc1] - 2026-08-01

### Added
- In-cluster Kubernetes operator (`cmd/stratabench-operator`) — watches `Benchmark` CRs, runs `manifest.Apply`, updates `.status`
- `deploy/k8s/operator-deployment.yaml` with RBAC
- `examples/benchmark-mock.yaml` for smoke testing
- `docs/DELL-LAB-VALIDATION.md` pre-v1.0 hardware checklist
- Intent-based `manifest.Apply` (agent loop) for CRs with `spec.intent`

### Changed
- CRD spec requires only `target` (profile or intent)
- Docker image includes `stratabench-operator` binary

## [0.4.0-rc1] - 2026-08-01

### Added
- Agentic loop: `stratabench agent` (plan → validate → run → analyze → report)
- Ollama planner and NL report summaries
- Analyst agent with regression, tail latency, and client variance detection
- 30-day rolling regression baselines
- Hardware inventory and SMART health history
- SBK application profiles (Kafka, RocksDB, PostgreSQL)
- Native SBK drivers: pgbench, db_bench, kafka-producer-perf-test (with synthetic fallback)
- Kubernetes manifests, CRD (`benchmarks.stratabench.io`), and `stratabench apply`
- REST API extensions: inventory, analyze, agent
- Docker, docker-compose, GHCR CI, GitHub Pages docs
- 15 workload profiles across block, file, object, VM, and application layers

### Changed
- Dockerfile uses flexible CMD (no fixed ENTRYPOINT)
- Version command reports `StrataBench 0.4.0-rc1`

## [0.3.0] - Phase 3

- REST API, Prometheus metrics, vdbench, Warp RDMA
- Cross-layer analysis, SBK CSV import, Grafana dashboard

## [0.2.0] - Phase 2

- Distributed agent, Warp wrapper, compare/export/baseline

## [0.1.0] - Phase 1

- CLI, validator, mock + fio engines, SQLite, HTML reports
