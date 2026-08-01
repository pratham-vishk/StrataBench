# Hardware Validation by Use Case

Every StrataBench profile has **workload validation** (cache size, ramp time, percentiles) and **hardware validation** (tools, devices, NICs). Enable with:

```bash
stratabench validate --profile nvme-random-oltp --target /dev/nvme0n1 --check-hardware
stratabench run --profile afa-multi-lun --target /dev/sdb,/dev/sdc --check-hardware
```

`--check-hardware` is **on by default** for `validate` and `run` (skipped with `--mock`).

---

## Use case matrix

| Use case | Profile(s) | Required tools | Host hardware | Guest hardware (virtual) |
|----------|------------|----------------|---------------|--------------------------|
| **HDD sequential** | `hdd-sequential-read` | fio | Rotational block device (`/dev/sd*`) | virtio/SCSI disk in VM |
| **SSD random 4K** | `ssd-random-4k` | fio | SSD or NVMe | virtio disk |
| **NVMe OLTP** | `nvme-random-oltp` | fio | NVMe device, 32GB+ RAM | `vm-nvme-oltp` |
| **NVMe stress** | `nvme-max-stress` | fio | NVMe, 64GB+ RAM recommended | `vm-disk-stress` |
| **NVMe passthrough** | `spdk-nvme-peak` | SPDK perf | PCIe NVMe, hugepages | `vm-nvme-passthrough` (vfio-pci) |
| **AFA multi-LUN** | `afa-multi-lun` | vdbench | 2+ array LUNs | `vm-afa-multi-lun` (vSCSI/RDM) |
| **File parallel** | `file-parallel-*` | elbencho | NFS/Lustre/CephFS mount | `vm-file-*` (SSH + mount) |
| **S3 PUT/GET** | `s3-put/get-throughput` | warp | MinIO/S3 endpoint, network | `vm-s3-put-throughput` |
| **S3 cluster** | `s3-cluster-put-get` | warp | Multi-node MinIO cluster | agents on client nodes |
| **S3 RDMA** | `s3-cluster-rdma`, `vm-s3-rdma` | warp | **RDMA NIC** (`rdma link show`) | SR-IOV RDMA vNIC in guest |
| **PostgreSQL** | `app-postgres-tpc-c` | pgbench | PostgreSQL server, 8GB+ RAM | `vm-app-postgres` + agent |
| **Kafka** | `app-kafka-producer` | kafka-producer-perf-test | Kafka broker | `vm-app-kafka` + agent |
| **RocksDB** | `app-rocksdb-read` | db_bench | RocksDB data path | agent on host with DB |

---

## What gets checked

| Rule | When | Severity |
|------|------|----------|
| `tool:*` | Engine binary in PATH (fio, warp, vdbench…) | warning |
| `memory` | Host RAM vs profile minimum | warning |
| `nvme_present` | NVMe/SSD profiles | warning |
| `hdd_present` | HDD profiles | warning |
| `block_device_count` | AFA multi-LUN (2+ devices) | warning |
| `rdma` | S3 RDMA profiles | warning |
| `ssh` | VM profiles (guest execution) | **error** |
| `target_device` | `/dev/*` target exists in inventory | warning |

Warnings do not block the run by default. SSH missing on VM profiles is an error.

---

## Per-use-case Dell lab checklist

### Before any run

```bash
stratabench inventory collect
stratabench smart collect          # physical only
stratabench validate --profile <name> --target <target> --check-hardware
```

### HDD

| Check | Command / criteria |
|-------|-------------------|
| Rotational device | `inventory list` shows `rotational: true` |
| Direct I/O | `validate --profile hdd-sequential-read` passes |
| SMART healthy | `smart list` — no reallocated sector spike |

### NVMe

| Check | Command / criteria |
|-------|-------------------|
| Device visible | `nvme list` or inventory shows NVMe |
| Firmware noted | inventory records model + firmware |
| Dataset > cache | validator `dataset_gt_cache` passes |
| SMART / wear | `smart collect` — wear < 80% |

### AFA

| Check | Command / criteria |
|-------|-------------------|
| 2+ LUNs mapped | `ls /dev/sd*` — count matches profile |
| vdbench installed | `which vdbench` |
| Multi-path (optional) | document active paths |

### S3 / RDMA

| Check | Command / criteria |
|-------|-------------------|
| MinIO reachable | `curl http://endpoint:9000/minio/health/live` |
| Warp creds set | `WARP_ACCESS_KEY`, `WARP_SECRET_KEY` |
| RDMA links up | `rdma link show` (RDMA profiles only) |
| MTU 9000 (optional) | jumbo frames for RDMA performance |

### VM workloads

| Check | Command / criteria |
|-------|-------------------|
| SSH key auth | `ssh root@guest hostname` |
| Guest fio/vdbench | tool installed inside guest |
| NVMe passthrough | `lspci` in guest shows NVMe |
| Agent on guest (app) | `stratabench-agent` on :7777 |

---

## Topology + hardware

When using `--clients` or `--targets`, run hardware validation **on each node**:

```bash
# On each client before distributed run
stratabench inventory collect
stratabench validate --profile ssd-random-4k --target /dev/nvme0n1 --check-hardware

# Coordinator
stratabench run --profile ssd-random-4k --target /dev/nvme0n1 \
  --clients 10.0.1.1:7777,10.0.1.2:7777 --check-hardware
```

See [TOPOLOGY.md](TOPOLOGY.md) for client/server assignment patterns.
