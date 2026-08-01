# StrataBench

[![CI](https://github.com/pratham-vishk/StrataBench/actions/workflows/ci.yml/badge.svg)](https://github.com/pratham-vishk/StrataBench/actions/workflows/ci.yml)
[![Docker](https://github.com/pratham-vishk/StrataBench/actions/workflows/docker.yml/badge.svg)](https://github.com/pratham-vishk/StrataBench/actions/workflows/docker.yml)
![License](https://img.shields.io/badge/license-Apache%202.0-blue)
![Go](https://img.shields.io/badge/go-1.25+-00ADD8)
![Profiles](https://img.shields.io/badge/profiles-30+-brightgreen)

**Agentic, honest storage benchmarking for every layer — HDD, NVMe, AFA, S3, VM, and application workloads.**

StrataBench is an open-source platform that **orchestrates** industry-standard engines (fio, SPDK, vdbench, Warp, elbencho, pgbench), **validates** workload design before you run, and **reports** unified results across physical bare metal, virtual machines, and distributed clusters.

> *Stop trusting benchmark numbers you haven't validated. StrataBench tells you if your test is honest — before you act on the results.*

**Docs:** [pratham-vishk.github.io/StrataBench](https://pratham-vishk.github.io/StrataBench/) · **Container:** `ghcr.io/pratham-vishk/stratabench`

---

## Keywords

`storage benchmark` · `NVMe benchmark` · `HDD performance` · `all-flash array` · `AFA` · `S3 benchmark` · `MinIO Warp` · `RDMA` · `fio orchestration` · `SPDK perf` · `vdbench` · `VM storage` · `HCI benchmark` · `PostgreSQL pgbench` · `Kafka throughput` · `distributed benchmark` · `multi-node` · `Kubernetes operator` · `storage validation` · `IOPS` · `latency p99` · `regression testing` · `SMART monitoring` · `agentic AI` · `Dell lab` · `enterprise storage`

---

## Why StrataBench?

| | fio | SPDK | Warp | vdbench | SBK | HCIBench | **StrataBench** |
|---|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| Block / NVMe | ✅ | ✅ | — | ✅ | — | ✅ | ✅ |
| HDD / SSD | ✅ | — | — | ✅ | — | ✅ | ✅ |
| File / parallel FS | partial | — | — | — | ✅ | — | ✅ |
| S3 / object + RDMA | — | — | ✅ | — | partial | — | ✅ |
| VM / guest workloads | manual | — | — | — | partial | ✅ | ✅ |
| Multi-client topologies | manual | — | partial | — | — | partial | ✅ |
| Multi-server topologies | — | — | partial | partial | — | — | ✅ |
| Pre-run validation | — | — | — | — | — | — | ✅ |
| Unified reporting | — | — | — | — | — | — | ✅ |
| Agentic NL → report | — | — | — | — | partial | — | ✅ |
| Regression baselines | — | — | — | — | — | — | ✅ |

**One platform. Every storage layer. Honest numbers.**

---

## What works today

### Storage layers & engines

| Layer | Engines | Physical | Virtual (VM) |
|-------|---------|:--------:|:------------:|
| **Block** | fio, SPDK, vdbench | HDD, SSD, NVMe, AFA multi-LUN | fio via SSH, NVMe passthrough |
| **File** | elbencho | NFS, Lustre, CephFS | elbencho in guest via SSH |
| **Object** | MinIO Warp | S3 PUT/GET, mixed, cluster, **RDMA** | MinIO in VM, S3 RDMA |
| **Application** | pgbench, db_bench, kafka-perf | PostgreSQL, RocksDB, Kafka | agent on guest VM |
| **Mock** | synthetic | `--mock` on any profile | `--mock` on any profile |

### Priority workloads — physical **and** virtual

| Workload | Physical | Virtual |
|----------|----------|---------|
| **HDD** | `hdd-sequential-read` | `vm-hdd-sequential` |
| **NVMe** | `nvme-random-oltp`, `nvme-max-stress`, `spdk-nvme-peak` | `vm-nvme-oltp`, `vm-nvme-passthrough` |
| **AFA** | `afa-multi-lun` | `vm-afa-multi-lun` |
| **S3 RDMA** | `s3-cluster-rdma` | `vm-s3-rdma` |

### Distributed topologies — all scenarios

| Scenario | Flag | Mode |
|----------|------|------|
| 1 client → 1 server | `--target` | `single` |
| N clients → 1 server | `--clients` | `pool` |
| 1 client → N servers | `--targets` | `sweep` |
| N clients → M servers | `--clients` + `--targets` | `shard` |
| N clients × M servers | `--topology matrix` | `matrix` |

### Platform features

- **Validator** — cache size, steady state, tail latency rules before every run
- **30+ workload profiles** — declarative YAML, extensible
- **Agentic loop** — `stratabench agent "nvme oltp database"` → plan → validate → run → analyze → report
- **MCP server** — `stratabench-mcp` exposes 7 tools for Cursor, Claude, and other CLI models
- **LLM planner** — Ollama or OpenAI-compatible APIs; keyword fallback
- **Regression tracking** — explicit baselines + 30-day rolling comparison
- **Branch compare** — benchmark two git branches, HTML impact report (`compare branches`)
- **Hardware inventory** — NVMe model, firmware, block devices, SMART history
- **REST API** + **Prometheus metrics** + **Grafana dashboard**
- **Kubernetes** — CRD, in-cluster operator, DaemonSet agents, CronJobs
- **Cross-layer analysis** — compare block vs object vs app in one report
- **SBK import** — ingest Storage Benchmark Kit CSV results

---

## Quick start

### Install

```bash
git clone https://github.com/pratham-vishk/StrataBench.git
cd StrataBench
make build
stratabench init
```

Or pull the container:

```bash
docker pull ghcr.io/pratham-vishk/stratabench:latest
docker run --rm ghcr.io/pratham-vishk/stratabench:latest version
```

### Run (no hardware required)

```bash
# List 30+ built-in profiles
./bin/stratabench profiles

# Mock run — works on Windows, macOS, Linux
./bin/stratabench run --profile nvme-random-oltp --target /dev/null --mock

# Sample benchmark — same flow, copies HTML/Excel/JSON to examples/sample-report/output/
./bin/stratabench sample --open-report
# or: make sample

# Full agentic loop
./bin/stratabench agent "ssd random 4k workload" --target /tmp/test --mock
```

### Use with CLI models (Cursor, Claude Code, Devin)

**Claude Code** and **Devin** work out of the box — clone the repo; MCP configs are committed (`.mcp.json`, `.devin/mcp_config.json`).

```bash
make build-mcp   # optional; go run works via .mcp.json
```

| Platform | Setup |
|----------|-------|
| **Claude Code** | Open repo → approve `stratabench` MCP (`/mcp`) |
| **Devin** | Clone repo → reads `AGENTS.md` + `CLAUDE.md` + `.devin/mcp_config.json` |
| **Claude Desktop** | Merge `examples/mcp-claude-desktop.json` |
| **Cursor** | Add `examples/mcp-cursor.json` to MCP settings |

See [AGENTS.md](AGENTS.md) and [docs/AGENTIC.md](docs/AGENTIC.md) for full setup.

```bash
# LLM planner (Ollama local or OpenAI-compatible)
export OPENAI_API_KEY=sk-...   # or: ollama serve
./bin/stratabench plan "s3 rdma cluster" --llm
./bin/stratabench agent "afa multi lun" --target /dev/sdb --llm --mock
```

### Run on real storage (Linux)

```bash
# NVMe OLTP — workload + hardware validation (on by default)
./bin/stratabench validate --profile nvme-random-oltp --target /dev/nvme0n1 --cache-bytes 34359738368
./bin/stratabench run --profile nvme-random-oltp --target /dev/nvme0n1

# AFA multi-LUN
./bin/stratabench run --profile afa-multi-lun --target /dev/sdb,/dev/sdc,/dev/sdd

# S3 cluster with RDMA
export WARP_ACCESS_KEY=minioadmin WARP_SECRET_KEY=minioadmin
./bin/stratabench run --profile s3-cluster-rdma --target 10.0.1.10:9000

# VM guest (fio inside VM via SSH)
./bin/stratabench run --profile vm-nvme-passthrough --target root@10.0.1.20:/dev/nvme0n1
```

### Distributed — multi-client, multi-server

```bash
# Start agents on client nodes
stratabench-agent   # listens on :7777

# N clients → 1 NVMe server (pool)
stratabench run --profile ssd-random-4k --target /dev/nvme0n1 \
  --clients 10.0.1.1:7777,10.0.1.2:7777,10.0.1.3:7777

# 1 client → N S3 servers (sweep)
stratabench run --profile s3-put-throughput \
  --targets 10.0.1.10:9000,10.0.1.11:9000,10.0.1.12:9000

# N clients → M servers (shard)
stratabench run --profile afa-multi-lun \
  --targets /dev/sdb,/dev/sdc,/dev/sdd \
  --clients 10.0.1.1:7777,10.0.1.2:7777,10.0.1.3:7777 \
  --topology shard
```

### Regression & reporting

```bash
stratabench baseline set --run-id <uuid>
stratabench run --profile nvme-random-oltp --target /dev/nvme0n1 --check-baseline
stratabench report --run-id <uuid>
stratabench analyze --run-id <uuid>
```

---

## Architecture

```
Natural language / CLI / API / Kubernetes CRD
                    │
                    ▼
         ┌──────────────────────┐
         │  Agent Layer         │
         │  Planner → Validator │
         │  → Analyst → Reporter│
         └──────────┬───────────┘
                    ▼
         ┌──────────────────────┐
         │  Orchestrator        │
         │  Topology engine     │
         │  30+ YAML profiles   │
         └──────────┬───────────┘
                    ▼
    ┌───────────────┼───────────────┐
    ▼               ▼               ▼
  fio           vdbench           Warp
  SPDK          elbencho          SBK (pgbench…)
    │               │               │
    └───────────────┴───────────────┘
                    ▼
         Unified result schema
         SQLite · HTML · JSON · Prometheus
```

We **orchestrate** proven tools — we don't replace fio or Warp. We add validation, topology, aggregation, and honest reporting on top.

---

## Deployment

| Method | Command |
|--------|---------|
| **Binary** | `make build` → `./bin/stratabench` |
| **Docker** | `docker compose up api` → REST on `:8080` |
| **Kubernetes** | `kubectl apply -k deploy/k8s/` |
| **K8s CRD** | `stratabench apply -f examples/benchmark-mock.yaml` |
| **Grafana** | `deploy/grafana/stratabench-dashboard.json` |

```bash
# Full K8s stack: API, agents, operator, PVC, CronJob
kubectl apply -k deploy/k8s/
kubectl apply -f examples/benchmark-topology-pool.yaml
kubectl get benchmarks -n stratabench -w
```

---

## CLI reference

| Command | Description |
|---------|-------------|
| `profiles` | List workload profiles |
| `plan` | Suggest profile from natural language |
| `validate` | Check workload design + hardware for profile (`--check-hardware`) |
| `run` | Execute benchmark (local or distributed) |
| `agent` | Full agentic loop end-to-end |
| `apply` | Apply Kubernetes-style benchmark manifest |
| `baseline` | Set / show / check regression baselines |
| `inventory` | Collect hardware inventory |
| `smart` | SMART health history |
| `analyze` | Tail latency, variance, regression insights |
| `cross-layer` | Multi-profile bottleneck analysis |
| `import sbk` | Import SBK CSV results |
| `compare` | Compare runs (`compare runs`) or git branches (`compare branches`) |
| `init` | Create `.stratabench` data directories |
| `report` | Generate HTML report |

Flags: `--profile` · `--target` · `--targets` · `--clients` · `--topology` · `--mock` · `--check-baseline` · `--ollama`

---

## Documentation

| Doc | What's inside |
|-----|---------------|
| [ENGINE-COVERAGE.md](docs/ENGINE-COVERAGE.md) | Every engine × physical/virtual matrix |
| [TOPOLOGY.md](docs/TOPOLOGY.md) | Multi-client / multi-server patterns |
| [DEV.md](docs/DEV.md) | Build, test, full CLI reference |
| [BRANCH-COMPARE.md](docs/BRANCH-COMPARE.md) | Compare two code branches with benchmarks |
| [ARCHITECTURE.md](docs/ARCHITECTURE.md) | System design |
| [DELL-LAB.md](docs/DELL-LAB.md) | Dell lab VM cluster setup |
| [DELL-LAB-VALIDATION.md](docs/DELL-LAB-VALIDATION.md) | Hardware sign-off checklist |
| [VISION.md](docs/VISION.md) | Project goals |
| [ROADMAP.md](docs/ROADMAP.md) | What's shipped vs planned |
| [profiles/](profiles/) | 30+ workload YAML definitions |

---

## Built at Dell Technologies

StrataBench is designed for enterprise storage validation — NVMe arrays, AFA LUNs, S3 clusters with RDMA, VM workloads on HCI, and application-layer benchmarks (PostgreSQL, Kafka, RocksDB). Run it on Dell lab VMs, customer sites, or any Linux cluster.

---

## Contributing

Contributions welcome. See [CONTRIBUTING.md](CONTRIBUTING.md) and [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).

```bash
make test          # run all tests
make run-mock      # smoke test
```

---

## License

Apache License 2.0 — see [LICENSE](LICENSE).

---

<p align="center">
  <strong>StrataBench</strong> — honest storage benchmarks, every layer, every topology.<br>
  <a href="https://pratham-vishk.github.io/StrataBench/">Documentation</a> ·
  <a href="https://github.com/pratham-vishk/StrataBench/releases">Releases</a> ·
  <a href="https://github.com/pratham-vishk/StrataBench/issues">Issues</a>
</p>
