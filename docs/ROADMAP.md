# StrataBench Roadmap

## Phase 1 — Foundation (MVP)

**Goal:** Runnable CLI with honest validation and first benchmark engine.

### Deliverables

- [x] Repository scaffolding (`cmd/`, `internal/`, `profiles/`) — Rust `crates/` engine deferred
- [x] Normalized result schema (JSON) + SQLite storage
- [x] Rule-based validator (5 core rules)
- [x] Hardware discovery module (NVMe, block devices, CPU, memory)
- [x] Hardware inventory database (SQLite)
- [x] fio wrapper (profile → job file → JSON parser)
- [x] StrataBench engine v0.2 (Rust): block I/O via `O_DIRECT` + pread/pwrite on Linux — synthetic fallback elsewhere
- [x] CLI: `stratabench run`, `stratabench validate`, `stratabench report`
- [x] 30 built-in workload profiles
- [x] HTML report generator (Grafana-style, multi-node, Excel export)
- [x] Planner agent v0.1 (profile selection from keywords, no LLM required)

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

- [x] Warp wrapper (HTTP; native coordinator via `--warp-clients` or profile `warp_clients`)
- [x] GOSBench wrapper (`gosbench-server` + generated YAML config; profile `s3-gosbench-write`)
- [x] elbencho wrapper (file/block)
- [x] `stratabench-agent` daemon (Go, HTTP JSON on :7777; optional `STRATABENCH_AGENT_TOKEN`)
- [x] SSH-based multi-node deployment (`stratabench lab`, `scripts/lab-*.sh`)
- [x] SBK CSV import → normalized schema
- [x] Excel report generation
- [x] REST API (`/runs`, `/profiles`, `/validate`, `/report/{id}`, `/compare`)
- [x] Planner agent v0.2 (Ollama integration for NL → profile)
- [x] 10+ additional profiles (VM, file, S3 cluster)

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
- [x] VM guest benchmarks via SSH (fio inside KVM/VMware guests)
- [x] Warp RDMA mode (`--rdma=cpu`)
- [x] Prometheus metrics exporter
- [x] Grafana dashboards (bundled)
- [x] PostgreSQL backend option (`STRATABENCH_DATABASE_URL`)
- [x] Regression baselines (per profile + target)
- [x] Analyst agent (anomaly detection, regression alerts)
- [x] Cross-layer comparison reports

### Success criteria

- AFA multi-LUN test via vdbench with unified report
- SPDK peak vs. fio comparison on same NVMe device
- Grafana dashboard live during 1-hour stress test — **assignment progress + live interval gauges (mock, fio, warp)**

---

## Phase 4 — Agentic + ecosystem

**Goal:** Full agentic lifecycle and open-source release readiness.

### Deliverables

- [x] Full agentic loop: plan → validate → run → analyze → report (NL end-to-end)
- [x] Ollama planner (v0.2) with keyword fallback
- [x] Analyst agent (regression, tail latency, client variance)
- [x] Regression tracking (baseline per profile + target)
- [x] Hardware inventory database (NVMe model, firmware, SMART history)
- [x] Kubernetes deployment manifests (API, agent, CronJob, CRD, operator)
- [x] Public documentation site (GitHub Pages)
- [x] Contributor guide + CI/CD (GitHub Actions)
- [x] **Public open-source release**
- [x] SBK native driver integration (pgbench, db_bench, kafka-producer-perf-test)
- [ ] SBK full Storage Benchmark Kit bridge — JSON import added; Python runner deferred

### Success criteria

- User says *"benchmark my S3 cluster for read-heavy AI workload"* → full report without manual config
- Regression alert when p99 degrades 15% vs. 30-day baseline
- [x] 30-day rolling regression when no explicit baseline is set
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

- [x] Apache 2.0 LICENSE committed
- [x] CONTRIBUTING.md with code style and PR process
- [x] CI: build, test on Linux
- [x] Security policy (SECURITY.md)
- [x] Code of conduct
- [x] Documentation site (GitHub Pages workflow)
- [x] Dockerfile
- [x] README badges (build status, license)
- [x] Docker CI workflow (GHCR publish on tag)
