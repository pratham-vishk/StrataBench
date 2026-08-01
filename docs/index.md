# StrataBench

Agentic, honest storage benchmarking for every layer.

## Quick links

| Document | Description |
|----------|-------------|
| [README](../README.md) | Project overview and quick start |
| [Development Guide](DEV.md) | Build, test, and CLI reference |
| [Architecture](ARCHITECTURE.md) | System design |
| [Roadmap](ROADMAP.md) | Implementation phases |
| [Dell Lab Guide](DELL-LAB.md) | VM cluster deployment |
| [Contributing](../CONTRIBUTING.md) | How to contribute |

## Quick start

```bash
git clone https://github.com/pratham-vishk/StrataBench.git
cd StrataBench
make build
./bin/stratabench agent "nvme oltp" --target /dev/nvme0n1 --mock
```

## Deployment

- **Docker:** `docker build -t stratabench .`
- **Kubernetes:** manifests in `deploy/k8s/`
- **Grafana:** dashboard in `deploy/grafana/`
