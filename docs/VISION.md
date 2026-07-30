# StrataBench Vision

## Mission

Build the **open-source storage benchmark platform** that any team can trust — from a laptop HDD to a petabyte-scale S3 cluster — with agentic intelligence that designs honest tests, runs them at full load, and explains what the numbers actually mean.

## The problem

### Fragmented tooling

Teams benchmark storage with different tools per layer:

- **Block / NVMe / AFA** → fio, SPDK, vdbench
- **Virtual machines / HCI** → HCIBench, manual fio inside guests
- **Object / S3 clusters** → Warp, COSBench, GOSBench
- **Applications / databases** → SBK, HammerDB, sysbench

Each tool has its own config format, output format, and assumptions. Comparing block performance to S3 gateway performance requires manual correlation — if it happens at all.

### Dishonest numbers

Most benchmark results in the wild are misleading:

- Dataset smaller than array cache → measuring RAM, not disk
- No ramp / steady-state period → reporting burst, not sustained performance
- Missing tail latency (p99, p99.9) → hiding the latency users actually feel
- Wrong block size or queue depth for the claimed workload → hero numbers, not production reality

No existing tool systematically catches these mistakes **before** the test runs.

### No cross-layer insight

A storage admin might see:

- Block: 500K IOPS @ 80µs p99
- S3: 12K ops/s @ 45ms p99

Without a unified platform, the insight — *"S3 gateway is the bottleneck, not hardware"* — is manual guesswork.

## Our solution

StrataBench is three things:

### 1. Honest benchmark validator

Rule-based validation (deterministic, not LLM guesswork) that checks:

- Dataset size vs. known or discovered cache capacity
- Runtime vs. steady-state requirements
- Block size / QD / R/W mix vs. declared workload intent
- Required percentile reporting for the workload class

Invalid tests are blocked or corrected before execution.

### 2. Unified orchestration platform

One CLI, one result schema, one report — across:

| Layer | Targets | Primary engine |
|-------|---------|----------------|
| Raw hardware | HDD, SSD, NVMe | StrataBench engine + fio |
| Peak NVMe | Userspace max IOPS | SPDK perf |
| Enterprise AFA | Multi-LUN, data verify | vdbench |
| Virtual machine | KVM, VMware, vSAN | fio inside VM + agent deploy |
| File / parallel FS | NFS, Lustre, CephFS | StrataBench engine + elbencho |
| Object / S3 cluster | MinIO, Ceph RGW, AWS S3 | Warp + GOSBench |
| S3 over RDMA | GPU-direct, RDMA path | Warp `--rdma` |
| Application | Kafka, RocksDB, PostgreSQL | SBK (integrated) |

### 3. Agentic benchmark lifecycle

```
Natural language request
        │
        ▼
   Planner Agent    → discovers hardware, infers workload, selects profile
        │
        ▼
  Validator Agent   → checks honesty rules, fixes or rejects plan
        │
        ▼
   Runner Agent     → orchestrates engines across nodes
        │
        ▼
  Analyst Agent     → detects anomalies, regressions, bottlenecks
        │
        ▼
  Reporter Agent    → human-readable report + charts + recommendations
```

Agents use local LLMs (Ollama) by default — storage teams keep data on-premise.

## What we build vs. what we integrate

| Build ourselves | Integrate |
|-----------------|-----------|
| Agent orchestration (Planner, Validator, Analyst) | fio — complex block patterns |
| Hardware discovery (NVMe, SMART, NUMA, NIC) | SPDK perf — peak NVMe |
| Honest test profile engine | Warp — S3 cluster + RDMA |
| StrataBench I/O engine (Rust) | GOSBench — distributed S3 |
| Result schema + time-series store | SBK — DB/MQ layer (Phase 3) |
| Cross-layer comparison engine | vdbench — enterprise AFA |
| CLI + API | Grafana — live dashboards |

**Lightweight engine ≠ weak loading.** Our Rust engine is lean in code but capable of full stress: multi-thread, high queue depth, distributed agents, long duration.

## Target users

- **Storage engineers** validating NVMe, AFA, and SAN performance
- **Cloud / platform teams** benchmarking S3-compatible object stores
- **Virtualization admins** testing VM disk and HCI cluster performance
- **Performance engineers** who need honest, reproducible, comparable numbers
- **Open-source contributors** who want one platform instead of ten tools

## Success criteria

1. A user can say *"benchmark my NVMe for database workload"* and get a validated, honest test plan in seconds.
2. Results from block, file, and object layers are comparable via one schema.
3. Invalid tests are caught before execution with a clear explanation.
4. Heavy load tests saturate storage without the benchmark tool becoming the bottleneck.
5. The project is contributor-friendly and ready for public open-source release.

## Name

**StrataBench** — *strata* (layers) + *bench* (benchmark).

Storage has layers: hardware → block → file → object → application. StrataBench benchmarks every stratum.

## Open-source intent

This project is designed from day one for public open-source release under Apache 2.0. Private development now; public when Phase 1 is scaffolded and documented.
