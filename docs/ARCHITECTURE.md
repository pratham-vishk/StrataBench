# StrataBench Architecture

## Overview

StrataBench follows a **layered architecture**: agent intelligence on top, orchestration in the middle, pluggable engines at the bottom, unified results throughout.

```
┌─────────────────────────────────────────────────────────────────┐
│                     User Interfaces                              │
│              CLI  ·  REST API  ·  Natural Language               │
└────────────────────────────┬────────────────────────────────────┘
                             │
┌────────────────────────────▼────────────────────────────────────┐
│                      Agent Layer                                 │
│  ┌──────────┐  ┌───────────┐  ┌─────────┐  ┌──────────┐        │
│  │ Planner  │→ │ Validator │→ │ Analyst │→ │ Reporter │        │
│  └──────────┘  └───────────┘  └─────────┘  └──────────┘        │
│  LLM (Ollama) for NL · Rule engine for validation (deterministic)│
└────────────────────────────┬────────────────────────────────────┘
                             │
┌────────────────────────────▼────────────────────────────────────┐
│                   Orchestration Core                             │
│  Workload Profiles (YAML) · Scheduler · Engine Router            │
│  Hardware Discovery · Run State Machine                          │
└────────────────────────────┬────────────────────────────────────┘
                             │
         ┌───────────────────┼───────────────────┐
         │                   │                   │
┌────────▼────────┐ ┌────────▼────────┐ ┌───────▼────────┐
│ StrataBench     │ │ External        │ │ StrataBench    │
│ Engine (Rust)   │ │ Engines         │ │ Agent (nodes)  │
│                 │ │ fio·SPDK·Warp   │ │                │
│ block·file·s3   │ │ GOSBench·SBK    │ │ deploy·collect │
└────────┬────────┘ └────────┬────────┘ └───────┬────────┘
         │                   │                   │
         └───────────────────┼───────────────────┘
                             │
┌────────────────────────────▼────────────────────────────────────┐
│                    Result Pipeline                               │
│  Normalizer → SQLite/PostgreSQL → Comparator → Report Generator  │
│  Prometheus metrics · Grafana dashboards                         │
└─────────────────────────────────────────────────────────────────┘
```

### Implementation status (2026)

| Component | Shipped | Notes |
|-----------|---------|-------|
| Orchestration + agents | Go HTTP (`:7777`) | Bearer token + optional mTLS |
| Native StrataBench engine | Bridge | External `stratabench-engine` binary; Rust crate deferred |
| Result store | SQLite or PostgreSQL | `STRATABENCH_DATABASE_URL` for Postgres |
| Engines | fio, warp, vdbench, spdk, elbencho, sbk, gosbench | Native Rust engine deferred |
| Mid-run monitoring | Partial | Prometheus assignment progress + `/runs/{id}/progress` API |
| Reports | HTML + Excel | PDF not implemented |

---

## Components

### 1. User interfaces

| Interface | Purpose | Phase |
|-----------|---------|-------|
| `stratabench` CLI | Primary interface for run, plan, validate, report | 1 |
| REST API | Automation, CI/CD integration | 2 |
| Natural language | Agent-driven planning via Ollama | 1 (basic) / 4 (full) |

### 2. Agent layer

#### Planner Agent
- Input: natural language or structured request
- Discovers hardware via discovery module
- Maps intent to workload profile (e.g. "OLTP" → `nvme-random-oltp`)
- Outputs: executable test plan (YAML)

#### Validator Agent (rule-based, not LLM)
Deterministic checks:

| Rule | Example failure |
|------|-----------------|
| `dataset > cache` | 10GB test on 2TB cache array |
| `steady_state` | runtime < ramp_time + 300s |
| `tail_latency` | percentile_list missing p99 |
| `direct_io` | `direct=0` for block device test |
| `workload_match` | 4K random for declared "streaming" workload |

#### Runner (execution, not LLM)
- Dispatches plan to orchestrator
- Monitors mid-run (thermal throttle, latency spike detection)
- Collects raw engine output

#### Analyst Agent
- Compares run to historical baselines
- Detects anomalies (p99 spike at minute 4 → possible throttle)
- Cross-layer correlation when multiple layers tested

#### Reporter Agent
- Generates HTML/PDF/Excel reports
- Natural language summary via local LLM
- Regression alerts

### 3. Orchestration core

#### Workload profiles (`profiles/*.yaml`)
Declarative test definitions. See [profiles/README.md](../profiles/README.md).

#### Engine router
Selects engine based on profile `layer` and `engine` fields:

```rust
match profile.engine {
    Engine::StrataBench => run_stratabench_engine(profile),
    Engine::Fio         => run_fio_wrapper(profile),
    Engine::Spdk        => run_spdk_wrapper(profile),
    Engine::Warp        => run_warp_wrapper(profile),
    Engine::Gosbench    => run_gosbench_wrapper(profile),
}
```

#### Hardware discovery
Collects before each run:

- Block devices: `nvme list`, `lsblk`, SMART via `smartctl`
- NUMA topology: `numactl --hardware`
- Network: NIC speed, RDMA capability (`rdma link`)
- Memory: total RAM (for cache size estimation)
- VM context: hypervisor, virtio version (when inside guest)

### 4. StrataBench engine (Rust)

Our own I/O engine — lean code, heavy load capability.

```
stratabench-engine/
├── block/     # O_DIRECT, libaio, multi-thread, high QD
├── file/      # POSIX file I/O, directory tree workloads
├── s3/        # S3 HTTP (AWS SDK / reqwest)
└── metrics/   # Per-op latency histogram, throughput
```

Design constraints:
- Pre-allocated I/O buffers (no alloc per operation)
- Lock-free per-thread queues
- Nanosecond latency recording
- CSV + JSON output matching [RESULT_SCHEMA.md](RESULT_SCHEMA.md)

Does **not** replace fio/SPDK for edge cases — covers 80% of block/file/S3 HTTP workloads natively.

### 5. External engine wrappers

Thin adapters that:
1. Translate StrataBench profile → engine-native config
2. Execute subprocess
3. Parse output → normalized schema

| Engine | Wrapper input | Parser output |
|--------|---------------|---------------|
| fio | `.fio` job file | JSON (`--output-format=json`) |
| SPDK perf | CLI args | stdout regex |
| Warp | CLI args | JSON report |
| GOSBench | config file | Prometheus / JSON |
| SBK | CSV import | existing CSV format |

### 6. StrataBench agent

Lightweight daemon deployed on benchmark nodes:

```
stratabench-agent --listen :7777
```

Responsibilities:
- Receive run instructions from orchestrator
- Execute local engine or wrapper
- Stream metrics to orchestrator
- Report node health (CPU, memory, disk temp)

Multi-node pattern (similar to Warp coordinator + clients):

```
Orchestrator
    ├── agent@node1  → fio on /dev/nvme0n1
    ├── agent@node2  → fio on /dev/nvme0n1
    └── agent@node3  → warp client → S3 endpoint
```

### 7. Result pipeline

#### Normalizer
All engine outputs → [RESULT_SCHEMA.md](RESULT_SCHEMA.md) JSON.

#### Storage
- Phase 1: SQLite (single node)
- Phase 3: PostgreSQL (multi-user, regression history)

#### Comparator
- Run vs. run diff
- Cross-layer: block IOPS vs. S3 ops/s with bottleneck inference
- Regression: alert if p99 degrades > 10% vs. baseline

#### Reports
- HTML dashboard (Phase 1)
- Excel via chart generation (Phase 2)
- Grafana live (Phase 3)

---

## Data flow (single run)

```
1. User:  stratabench plan "OLTP on NVMe"
2. Planner: hardware discovery → select nvme-random-oltp profile
3. Validator: dataset 200GB > cache ✓, runtime 600s ✓, p99 ✓
4. Router:  engine=fio (block layer)
5. Agent:   deploy fio job on target node
6. fio:     runs 600s, outputs JSON
7. Normalizer: JSON → StrataBench result schema
8. Analyst:  no anomalies, within baseline
9. Reporter: HTML report + "185K IOPS, p99=450µs"
10. Store:  SQLite, run_id=uuid
```

---

## Technology stack

| Component | Technology | Rationale |
|-----------|------------|-----------|
| Core engine | Rust | Performance, safety, single binary |
| Orchestrator + agent | Go | Concurrency, easy deploy, gRPC |
| Agent LLM | Ollama (local) | On-premise, no data leaves network |
| Validator | Rule engine (Go/Rust) | Deterministic, fast, no hallucination |
| Profiles | YAML | Human-readable, version-controllable |
| Result DB | SQLite → PostgreSQL | Start simple, scale later |
| Metrics | Prometheus + Grafana | Industry standard |
| Packaging | Docker + deb/rpm | Storage teams expect bare-metal deploy |

---

## Repository structure (target)

```
StrataBench/
├── README.md
├── LICENSE
├── CONTRIBUTING.md
├── docs/
│   ├── VISION.md
│   ├── ARCHITECTURE.md
│   ├── ROADMAP.md
│   ├── LANDSCAPE.md
│   └── RESULT_SCHEMA.md
├── profiles/              # Workload profile library
├── crates/
│   └── stratabench-engine/   # Rust I/O engine
├── cmd/
│   ├── stratabench/          # CLI (Go)
│   └── stratabench-agent/    # Node agent (Go)
├── internal/
│   ├── orchestrator/
│   ├── validator/
│   ├── discovery/
│   ├── normalizer/
│   └── wrappers/             # fio, spdk, warp adapters
├── agents/                   # Agent prompts + state machine
└── deploy/
    ├── docker-compose.yml
    └── grafana/
```

---

## Security considerations

- Agents authenticate to orchestrator via mTLS
- S3 credentials via env vars / vault, never logged
- Local LLM only by default (no benchmark data to cloud APIs)
- Raw device access requires explicit `--allow-raw-device` flag

---

## Non-goals (v1)

- Rewriting fio, SPDK, or Warp
- Windows-native engine (Linux first; Windows via WSL/fio later)
- GUI-first (CLI first, web dashboard Phase 3)
- Cloud SaaS hosting of results
