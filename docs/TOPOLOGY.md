# Topology Guide

StrataBench supports all common client/server assignment patterns via `--topology`.

## Scenarios

| Scenario | Clients | Servers | Topology | Auto? |
|----------|---------|---------|----------|-------|
| 1 client → 1 server | 0–1 | 1 (`--target`) | `single` | Yes (default) |
| N clients → 1 server | N (`--clients`) | 1 | `pool` | Yes |
| 1 client → N servers | 0–1 | N (`--targets`) | `sweep` | Yes |
| N clients → M servers (paired) | N | M | `shard` | Yes (when N>1 and M>1) |
| N clients × M servers (all pairs) | N | M | `matrix` | Manual |

## Examples

### 1 client, 1 server (local)

```bash
stratabench run --profile nvme-random-oltp --target /dev/nvme0n1
```

### Multi-client, one server (pool)

All agents hammer the same device/endpoint. Results are summed.

```bash
stratabench run --profile ssd-random-4k --target /dev/nvme0n1 \
  --clients 10.0.1.1:7777,10.0.1.2:7777,10.0.1.3:7777 \
  --topology pool
```

### One client, multi-server (sweep)

Coordinator runs the profile against each target in parallel.

```bash
stratabench run --profile s3-put-throughput \
  --targets 10.0.1.10:9000,10.0.1.11:9000,10.0.1.12:9000 \
  --topology sweep
```

### Multi-client, multi-server (shard)

Round-robin pairing: client[i] → target[i % M].

```bash
stratabench run --profile nvme-random-oltp \
  --targets /dev/nvme0n1,/dev/nvme1n1,/dev/nvme2n1 \
  --clients 10.0.1.1:7777,10.0.1.2:7777,10.0.1.3:7777 \
  --topology shard
# c1→nvme0, c2→nvme1, c3→nvme0
```

### Multi-client × multi-server (matrix)

Every client runs against every server (cartesian product).

```bash
stratabench run --profile s3-cluster-rdma \
  --targets 10.0.1.10:9000,10.0.1.11:9000 \
  --clients 10.0.2.1:7777,10.0.2.2:7777 \
  --topology matrix
# 4 assignments: c1→s1, c1→s2, c2→s1, c2→s2
```

## Auto mode (default)

```bash
--topology auto
```

| Clients | Targets | Inferred mode |
|---------|---------|---------------|
| 0–1 | 1 | `single` |
| 0–1 | 2+ | `sweep` |
| 2+ | 1 | `pool` |
| 2+ | 2+ | `shard` |

Use `--topology matrix` explicitly for full cartesian fan-out.

### Native Warp coordinator mode

For cluster profiles (`s3-cluster-*`), use `--warp-clients` (Warp port **7761**) on a single coordinator run — distinct from `--clients` (StrataBench agents on port **7777**):

```bash
stratabench run --profile s3-cluster-put-get --target 10.0.1.10:9000 \
  --warp-clients 10.0.1.1:7761,10.0.1.2:7761,10.0.1.3:7761
```

For agent-based distribution (recommended), use `--clients` + `--topology pool` instead.

## Results

Each run stores:

- **`results`** — aggregated across all assignments
- **`clients[]`** — per client+target assignment (`host`, `target`, `results`)
- **`targets[]`** — per-server rollup (`target`, `results`)
- **`topology`** — mode used (`pool`, `sweep`, etc.)

## Kubernetes / API

```yaml
spec:
  profile: s3-cluster-rdma
  targets:
    - 10.0.1.10:9000
    - 10.0.1.11:9000
  clients:
    - 10.0.2.1:7777
    - 10.0.2.2:7777
  topology: shard
```

REST API: `POST /api/v1/runs` with `targets`, `clients`, `topology` fields.
