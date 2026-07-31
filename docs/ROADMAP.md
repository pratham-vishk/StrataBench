# StrataBench Roadmap

## Phase 1 — Foundation (MVP)

**Goal:** Runnable CLI with honest validation and first benchmark engine.

### Deliverables

- [ ] Repository scaffolding (`cmd/`, `crates/`, `internal/`, `profiles/`)
- [ ] Normalized result schema (JSON) + SQLite storage
- [ ] Rule-based validator (5 core rules)
- [ ] Hardware discovery module (NVMe, block devices, NUMA, memory)
- [ ] fio wrapper (profile → job file → JSON parser)
- [ ] StrataBench engine v0.1 (Rust): block I/O via `O_DIRECT` + libaio
- [ ] CLI: `stratabench run`, `stratabench validate`, `stratabench report`
- [ ] 5 built-in workload profiles
- [ ] Basic HTML report generator
- [ ] Planner agent v0.1 (profile selection from keywords, no LLM required)

### Profiles (Phase 1)

| Profile | Layer | Engine |
|---------|-------|--------|
| `hdd-sequential-read` | block | fio |
| `ssd-random-4k` | block | stratabench |
| `nvme-random-oltp` | block | fio |
| `s3-put-throughput` | object | stratabench (S3 HTTP) |
| `s3-get-throughput` | object | stratabench (S3 HTTP) |

### Success criteria

- `stratabench validate` catches invalid test (dataset < cache) before run
- fio run produces normalized JSON in SQLite
- HTML report shows IOPS, throughput, p50/p95/p99

---

## Phase 2 — Multi-engine + distributed

**Goal:** S3 cluster testing and multi-node orchestration.

### Deliverables

- [ ] Warp wrapper (HTTP + distributed coordinator mode)
- [ ] GOSBench wrapper
- [ ] elbencho wrapper (file/block)
- [ ] `stratabench-agent` daemon (Go, gRPC)
- [ ] SSH-based multi-node deployment
- [ ] SBK CSV import → normalized schema
- [ ] Excel report generation
- [ ] REST API (`/runs`, `/profiles`, `/report/{id}`)
- [ ] Planner agent v0.2 (Ollama integration for NL → profile)
- [ ] 10 additional profiles (VM, file, S3 cluster)

### Profiles (Phase 2 additions)

| Profile | Layer | Engine |
|---------|-------|--------|
| `vm-disk-random` | vm-block | fio (inside VM) |
| `file-parallel-read` | file | elbencho |
| `s3-cluster-put-get` | object | warp (distributed) |
| `s3-mixed-workload` | object | warp |
| `nvme-max-stress` | block | stratabench (64 threads, QD 128) |

### Success criteria

- 6-node Warp distributed run aggregated into one report
- Agent deployed on 3 nodes, coordinated block test
- Cross-run comparison in CLI

---

## Phase 3 — Enterprise + observability

**Goal:** AFA, peak NVMe, VM suites, live monitoring.

### Deliverables

- [x] SPDK perf wrapper (peak NVMe IOPS)
- [x] vdbench wrapper (multi-LUN AFA)
- [ ] VM test suite (auto-deploy fio inside KVM/VMware guests)
- [x] Warp RDMA mode (`--rdma=cpu`)
- [x] Prometheus metrics exporter
- [x] Grafana dashboards (bundled)
- [ ] PostgreSQL backend option
- [x] Regression baselines (per profile + target)
- [ ] Analyst agent (anomaly detection, regression alerts)
- [x] Cross-layer comparison reports

### Success criteria

- AFA multi-LUN test via vdbench with unified report
- SPDK peak vs. fio comparison on same NVMe device
- Grafana dashboard live during 1-hour stress test

---

## Phase 4 — Agentic + ecosystem

**Goal:** Full agentic lifecycle and open-source release readiness.

### Deliverables

- [ ] Full agentic loop: plan → validate → run → analyze → report (NL end-to-end)
- [x] Regression tracking (baseline per profile + target)
- [ ] Hardware inventory database (NVMe model, firmware, SMART history)
- [ ] Kubernetes operator for in-cluster benchmarking
- [ ] Public documentation site
- [ ] Contributor guide + CI/CD (GitHub Actions)
- [ ] **Public open-source release**
- [ ] SBK driver integration (Kafka, RocksDB, PostgreSQL)

### Success criteria

- User says *"benchmark my S3 cluster for read-heavy AI workload"* → full report without manual config
- Regression alert when p99 degrades 15% vs. 30-day baseline
- 10 external contributors, passing CI

---

## Versioning

Semantic versioning from first public release:

- `0.1.0` — Phase 1 MVP
- `0.2.0` — Phase 2 distributed
- `0.3.0` — Phase 3 enterprise
- `1.0.0` — Phase 4 public OSS release

---

## Open-source release checklist

- [ ] Apache 2.0 LICENSE committed
- [ ] CONTRIBUTING.md with code style and PR process
- [ ] CI: build, test, lint on Linux
- [ ] Security policy (SECURITY.md)
- [ ] Code of conduct
- [ ] Documentation site
- [ ] Docker images published
- [ ] README badges (build status, license)
