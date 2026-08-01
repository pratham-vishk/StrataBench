# StrataBench

Agentic, honest storage benchmarking for every layer.

## Quick links

| Document | Description |
|----------|-------------|
| [Development Guide](DEV.md) | Build, test, and CLI reference |
| [Architecture](ARCHITECTURE.md) | System design |
| [Roadmap](ROADMAP.md) | Implementation phases |
| [Dell Lab Guide](DELL-LAB.md) | VM cluster deployment |
| [Dell Lab Validation](DELL-LAB-VALIDATION.md) | Pre-v1.0 hardware checklist |
| [Vision](VISION.md) | Project goals |
| [Contributing on GitHub](https://github.com/pratham-vishk/StrataBench/blob/master/CONTRIBUTING.md) | How to contribute |

## Quick start

```bash
git clone https://github.com/pratham-vishk/StrataBench.git
cd StrataBench
make build
./bin/stratabench agent "nvme oltp" --target /dev/nvme0n1 --mock
```

## Docker

```bash
docker pull ghcr.io/pratham-vishk/stratabench:0.5.0-rc1
docker run --rm ghcr.io/pratham-vishk/stratabench:0.5.0-rc1 version
```

## Kubernetes

```bash
kubectl apply -k deploy/k8s/
kubectl apply -f examples/benchmark-mock.yaml
kubectl get benchmarks -n stratabench -w
```

## Deployment

- **Docker:** `ghcr.io/pratham-vishk/stratabench`
- **Kubernetes:** manifests in `deploy/k8s/` (includes operator)
- **Grafana:** dashboard in `deploy/grafana/`
