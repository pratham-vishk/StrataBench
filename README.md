# StrataBench

**Agentic, honest storage benchmarking for every layer — from a single HDD to a distributed S3 cluster.**

StrataBench is an open-source storage performance platform that orchestrates benchmarks across block, file, object, VM, and application storage — with built-in validation so your numbers are trustworthy before you act on them.

> *"The first storage benchmark platform that tells you if your numbers are honest — before you trust them."*

---

## Why StrataBench?

Storage benchmarking today is fragmented:

| Tool | Block | File | S3 | VM/HCI | Agentic | Honest validation |
|------|-------|------|-----|--------|---------|-------------------|
| fio | ✅ | partial | ❌ | manual | ❌ | ❌ |
| SPDK perf | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| Warp / GOSBench | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ |
| SBK | partial | ✅ | HTTP | partial | partial | ❌ |
| elbencho | ✅ | ✅ | ✅ | partial | ❌ | ❌ |
| HCIBench | ✅ | ❌ | ❌ | ✅ | ❌ | ❌ |
| **StrataBench** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |

No single tool covers all layers with unified reporting, cross-layer comparison, and agent-driven test design. StrataBench fills that gap.

---

## Core principles

1. **Honest by default** — validate workload design before running (cache size, steady state, tail latency).
2. **Orchestrate, don't reinvent** — integrate proven engines (fio, SPDK, Warp) where they excel; build our own where we add unique value.
3. **Agentic intelligence** — natural language → test plan → execution → analysis → report.
4. **Unified results** — one schema across all engines for comparison and regression tracking.
5. **Heavy load capable** — lightweight tool overhead, maximum pressure on storage under test.

---

## Architecture (high level)

```
User (CLI / API / natural language)
        │
        ▼
┌─────────────────────────────────────┐
│  Agent Layer                        │
│  Planner → Validator → Analyst      │
└─────────────────────────────────────┘
        │
        ▼
┌─────────────────────────────────────┐
│  Orchestrator + Workload Profiles   │
└─────────────────────────────────────┘
        │
        ├── StrataBench Engine (Rust) — block, file, S3 HTTP
        ├── fio          — complex block / AFA patterns
        ├── SPDK perf    — peak NVMe IOPS
        ├── Warp         — S3 cluster + RDMA
        └── GOSBench     — distributed S3 workloads
        │
        ▼
┌─────────────────────────────────────┐
│  Result Normalizer → DB → Reports   │
└─────────────────────────────────────┘
```

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for full design.

---

## Quick start

> **Status:** Early design phase. Implementation starting in Phase 1.

```bash
# Coming soon
stratabench plan "benchmark my NVMe for OLTP workload"
stratabench run --profile nvme-random-oltp --target /dev/nvme0n1
stratabench validate --profile s3-cluster-put-get
stratabench report --run-id <uuid>
```

---

## Documentation

| Document | Description |
|----------|-------------|
| [VISION.md](docs/VISION.md) | Project vision and goals |
| [ARCHITECTURE.md](docs/ARCHITECTURE.md) | System design and components |
| [ROADMAP.md](docs/ROADMAP.md) | Phased implementation plan |
| [LANDSCAPE.md](docs/LANDSCAPE.md) | Existing tools and where StrataBench fits |
| [RESULT_SCHEMA.md](docs/RESULT_SCHEMA.md) | Normalized benchmark result format |
| [profiles/](profiles/) | Example workload profile definitions |

---

## Roadmap summary

| Phase | Focus |
|-------|-------|
| **1** | Rust engine, fio wrapper, validator rules, SQLite, 5 profiles |
| **2** | Warp + elbencho, multi-node agent, SBK CSV import |
| **3** | SPDK + vdbench, VM suites, Grafana, S3 RDMA |
| **4** | Full agentic loop, regression tracking, K8s operator |

---

## License

Apache License 2.0 — see [LICENSE](LICENSE).

---

## Contributing

Contributions welcome once Phase 1 scaffolding is in place. See [CONTRIBUTING.md](CONTRIBUTING.md).
