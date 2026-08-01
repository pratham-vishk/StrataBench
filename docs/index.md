# StrataBench Documentation

**Agentic, honest storage benchmarking — HDD, NVMe, AFA, S3 RDMA, VM, and application workloads.**

[GitHub](https://github.com/pratham-vishk/StrataBench) · [README](../README.md)

## Highlights

- **31 profiles** — block, file, object, VM, application (physical + virtual)
- **8 engines** — fio, SPDK, vdbench, Warp, elbencho, SBK, gosbench, native stub
- **All topologies** — 1:1, N:1 pool, 1:N sweep, N:M shard, N×M matrix
- **Validator** — honest workload rules + per-use-case hardware checks before every run
- **Agentic loop** — natural language → plan → validate → run → report
- **14 MCP tools** — Cursor, Claude Code, Devin
- **Async runs** — API, CLI, SSE progress stream, Prometheus live gauges
- **Kubernetes** — CRD, operator, agents, Docker on GHCR

## Documentation

| Document | Description |
|----------|-------------|
| [Engine Coverage](ENGINE-COVERAGE.md) | HDD / NVMe / AFA / S3 RDMA physical + virtual |
| [Hardware Validation](HARDWARE-VALIDATION.md) | Per-use-case tools, devices, and NIC checks |
| [Agentic Guide](AGENTIC.md) | MCP server, CLI models, LLM setup |
| [Topology Guide](TOPOLOGY.md) | Multi-client / multi-server patterns |
| [Monitoring](MONITORING.md) | Live progress, Prometheus, Grafana, SSE |
| [PostgreSQL Store](POSTGRES.md) | Shared database backend |
| [Native Engine](NATIVE-ENGINE.md) | `stratabench-engine` binary contract |
| [Development Guide](DEV.md) | Build, test, CLI reference |
| [Architecture](ARCHITECTURE.md) | System design |
| [Dell Lab Guide](DELL-LAB.md) | VM cluster deployment |
| [Dell Lab Validation](DELL-LAB-VALIDATION.md) | Hardware sign-off checklist |
| [Roadmap](ROADMAP.md) | Shipped vs planned |
| [Vision](VISION.md) | Project goals |

## Quick start

```bash
git clone https://github.com/pratham-vishk/StrataBench.git
cd StrataBench && make build

# Mock (no hardware)
./bin/stratabench run --profile nvme-random-oltp --target /dev/null --mock

# Async + watch
./bin/stratabench run --profile nvme-random-oltp --target /dev/null --mock --async --watch

# Agentic loop
./bin/stratabench agent "nvme oltp database" --target /dev/nvme0n1 --mock

# Distributed
stratabench run --profile ssd-random-4k --target /dev/nvme0n1 \
  --clients 10.0.1.1:7777,10.0.1.2:7777 --topology pool
```

## Docker

```bash
docker pull ghcr.io/pratham-vishk/stratabench:latest
docker run --rm ghcr.io/pratham-vishk/stratabench:latest version
```

## Kubernetes

```bash
kubectl apply -k deploy/k8s/
kubectl apply -f examples/benchmark-topology-pool.yaml
kubectl get benchmarks -n stratabench -w
```
