# Dell Lab Deployment

Use this guide when running StrataBench on Dell lab VMs (static IPs, shared storage cluster).

## Topology

```
Jump host (coordinator)
  ├── stratabench CLI
  └── stratabench-api (optional, :8080)

Client VMs (3+)
  └── stratabench-agent (:7777)

Storage targets
  ├── Block: /dev/nvme0n1 or array LUN
  ├── File: NFS/Lustre mount
  └── Object: MinIO/S3 endpoint
```

## Ports

| Service | Port | Purpose |
|---------|------|---------|
| stratabench-agent | 7777 | Remote benchmark execution |
| stratabench-api | 8080 | REST API + `/metrics` |
| Prometheus scrape | 8080 | `GET /metrics` |
| MinIO/S3 | 9000 | Warp profiles |

Open firewall rules between coordinator ↔ clients on **7777** and **8080** (if using API).

## Setup

### 1. Build on Linux (or cross-compile)

```bash
make build
```

Copy `bin/stratabench`, `bin/stratabench-agent`, and `profiles/` to each VM.

### 2. Start agents on clients

```bash
export STRATABENCH_AGENT_LISTEN=:7777
./bin/stratabench-agent
```

Verify from coordinator:

```bash
curl http://10.0.1.1:7777/v1/health
```

### 3. Run distributed benchmark

```bash
./bin/stratabench run \
  --profile nvme-random-oltp \
  --target /dev/nvme0n1 \
  --clients 10.0.1.1:7777,10.0.1.2:7777,10.0.1.3:7777
```

### 4. S3 cluster (Warp)

```bash
export WARP_ACCESS_KEY=...
export WARP_SECRET_KEY=...
./bin/stratabench run \
  --profile s3-cluster-put-get \
  --target 10.0.1.10:9000 \
  --clients 10.0.1.1:7777,10.0.1.2:7777
```

### 5. API + Prometheus (optional)

```bash
./bin/stratabench-api
# GET  http://<host>:8080/api/v1/health
# POST http://<host>:8080/api/v1/runs  {"profile":"ssd-random-4k","target":"/dev/sdb","mock":false}
# GET  http://<host>:8080/metrics
```

Import Grafana dashboard from `deploy/grafana/stratabench-dashboard.json`.

## Recommended profiles

| Profile | Layer | Engine | Notes |
|---------|-------|--------|-------|
| `nvme-random-oltp` | block | fio | OLTP-style 16K random |
| `nvme-max-stress` | block | mock/stratabench | Heavy QD stress |
| `spdk-nvme-peak` | block | SPDK | Userspace peak IOPS |
| `vm-disk-random` | vm-block | fio | Guest disk on HCI |
| `s3-cluster-put-get` | object | warp | Multi-node S3 |
| `s3-cluster-rdma` | object | warp | S3 with RDMA (`--rdma=cpu`) |
| `afa-multi-lun` | block | vdbench | Multi-LUN AFA random read |
| `file-parallel-read` | file | elbencho | NFS/Lustre parallel read |

## Cross-layer analysis

Run multiple layers and compare bottlenecks:

```bash
./bin/stratabench cross-layer \
  --profiles nvme-random-oltp,s3-put-throughput \
  --target /dev/nvme0n1
```

## SBK import

Import legacy SBK CSV results:

```bash
./bin/stratabench import sbk /path/to/sbk-results.csv
```

## Troubleshooting

- **Agent unreachable:** Check firewall on 7777, verify `STRATABENCH_AGENT_LISTEN`.
- **fio not found:** `sudo apt install fio` on client VMs.
- **Validation failed:** Increase working set above cache (`--cache-bytes`) or use `--skip-validate` for exploratory runs.
- **Warp auth errors:** Set `WARP_ACCESS_KEY` and `WARP_SECRET_KEY` on coordinator and agents.
