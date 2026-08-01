# Changelog

All notable changes to StrataBench are documented in this file.

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
