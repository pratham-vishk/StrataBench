# Engine Coverage Matrix

StrataBench orchestrates **real benchmark tools** on **physical** (bare metal / direct device) and **virtual** (VM guest / VM-hosted services) targets.

## Priority workloads: HDD, NVMe, AFA, S3 RDMA

| Workload | Physical profile | Virtual profile | Engine |
|----------|------------------|-----------------|--------|
| **HDD** | `hdd-sequential-read` | `vm-hdd-sequential` | fio (SSH in guest) |
| **NVMe** | `nvme-random-oltp`, `nvme-max-stress`, `spdk-nvme-peak` | `vm-nvme-oltp`, `vm-nvme-passthrough` | fio / spdk |
| **AFA** | `afa-multi-lun` | `vm-afa-multi-lun` | vdbench (SSH in guest) |
| **S3 RDMA** | `s3-cluster-rdma` | `vm-s3-rdma` | warp `--rdma=cpu` |

```bash
# Physical
stratabench run --profile hdd-sequential-read --target /dev/sda
stratabench run --profile nvme-random-oltp --target /dev/nvme0n1
stratabench run --profile afa-multi-lun --target /dev/sdb,/dev/sdc,/dev/sdd
stratabench run --profile s3-cluster-rdma --target 10.0.1.10:9000

# Virtual
stratabench run --profile vm-hdd-sequential --target root@10.0.1.20:/dev/vdb
stratabench run --profile vm-nvme-passthrough --target root@10.0.1.20:/dev/nvme0n1
stratabench run --profile vm-afa-multi-lun --target root@10.0.1.20:/dev/sdb,/dev/sdc,/dev/sdd
stratabench run --profile vm-s3-rdma --target 10.0.1.20:9000
```

### Virtual prerequisites

| Profile | Guest requirement |
|---------|-------------------|
| `vm-hdd-sequential` | Rotational virtio/SCSI disk (`/dev/vdb`) |
| `vm-nvme-passthrough` | NVMe device passed through via vfio-pci (`/dev/nvme0n1` in guest) |
| `vm-afa-multi-lun` | Multiple array LUNs via vSCSI/RDM + vdbench installed in guest |
| `vm-s3-rdma` | MinIO in guest + SR-IOV/RDMA-capable vNIC |

## Engines

| Engine | Tool | Physical | Virtual | Profiles |
|--------|------|----------|---------|----------|
| **fio** | Linux fio | block | vm-block (SSH) | `hdd-*`, `ssd-*`, `nvme-*`, `vm-disk-*`, `vm-hdd-*`, `vm-nvme-*` |
| **vdbench** | Oracle vdbench | block (AFA) | vm-afa (SSH) | `afa-multi-lun`, `vm-afa-multi-lun` |
| **spdk** | SPDK perf | block (NVMe userspace) | — (host PCIe only) | `spdk-nvme-peak` |
| **elbencho** | elbencho | file | vm-file (SSH) | `file-parallel-*`, `vm-file-parallel-*` |
| **warp** | MinIO Warp | object (+ RDMA) | vm-object (+ RDMA) | `s3-*`, `vm-s3-*` |
| **gosbench** | GOSBench server | object (staged S3) | — | `s3-gosbench-write` |
| **sbk** | pgbench / db_bench / kafka | application | vm-application (agent) | `app-*`, `vm-app-*` |
| **native** | `stratabench-engine` (Go/Rust) | block (opt-in) | — | `block-native-oltp`, any profile with `engine: stratabench` |
| **mock** | Synthetic | all (`--mock`) | all (`--mock`) | any profile |

## Layer × deployment

```
Layer            Physical target              Virtual target
─────────────────────────────────────────────────────────────────
block            /dev/nvme0n1                 —
vm-block         —                            root@10.0.1.20:/dev/vdb
vm-afa           —                            root@10.0.1.20:/dev/sdb,/dev/sdc
file             /mnt/nfs/share               —
vm-file          —                            root@10.0.1.20:/mnt/data
object           10.0.1.10:9000               10.0.1.20:9000
vm-object        —                            10.0.1.20:9000 (RDMA optional)
application      postgres://host/db           agent on VM + --clients
vm-application   —                            10.0.1.20:7777 + localhost DSN
```

## Physical commands (Dell lab)

```bash
# HDD
stratabench run --profile hdd-sequential-read --target /dev/sda

# NVMe
stratabench run --profile nvme-random-oltp --target /dev/nvme0n1
stratabench run --profile nvme-max-stress --target /dev/nvme0n1
stratabench run --profile spdk-nvme-peak --target 0000:01:00.0

# AFA
stratabench run --profile afa-multi-lun --target /dev/sdb,/dev/sdc,/dev/sdd

# S3 RDMA
export WARP_ACCESS_KEY=minioadmin WARP_SECRET_KEY=minioadmin
stratabench run --profile s3-cluster-rdma --target 10.0.1.10:9000

# S3 staged (GOSBench)
export GOSBENCH_ACCESS_KEY=minioadmin GOSBENCH_SECRET_KEY=minioadmin
stratabench run --profile s3-gosbench-write --target 10.0.1.10:9000
```

## Virtual commands (Dell lab)

```bash
# HDD in guest
stratabench run --profile vm-hdd-sequential --target root@10.0.1.20:/dev/vdb

# NVMe passthrough in guest
stratabench run --profile vm-nvme-passthrough --target root@10.0.1.20:/dev/nvme0n1
stratabench run --profile vm-nvme-oltp --target root@10.0.1.20:/dev/vdb

# AFA LUNs in guest
stratabench run --profile vm-afa-multi-lun --target root@10.0.1.20:/dev/sdb,/dev/sdc,/dev/sdd

# S3 RDMA to MinIO in guest
export WARP_ACCESS_KEY=minioadmin WARP_SECRET_KEY=minioadmin
stratabench run --profile vm-s3-rdma --target 10.0.1.20:9000
```

## Profile count

| Layer | Physical | Virtual | Total |
|-------|----------|---------|-------|
| block | 7 | — | 7 |
| vm-block | — | 6 | 6 |
| vm-afa | — | 1 | 1 |
| file | 2 | — | 2 |
| vm-file | — | 2 | 2 |
| object | 6 | — | 6 |
| vm-object | — | 2 | 2 |
| application | 3 | — | 3 |
| vm-application | — | 2 | 2 |
| **Total** | **18** | **13** | **31** |
