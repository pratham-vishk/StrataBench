# Engine Coverage Matrix

StrataBench orchestrates **real benchmark tools** on **physical** (bare metal / direct device) and **virtual** (VM guest / VM-hosted services) targets.

## Engines

| Engine | Tool | Physical | Virtual | Profiles |
|--------|------|----------|---------|----------|
| **fio** | Linux fio | block, vm-block (local) | vm-block (SSH guest) | `hdd-sequential-read`, `ssd-random-4k`, `nvme-random-oltp`, `nvme-max-stress`, `vm-disk-*` |
| **vdbench** | Oracle vdbench | block (multi-LUN) | — (bare metal only) | `afa-multi-lun` |
| **spdk** | SPDK perf | block (NVMe userspace) | — (requires PCIe passthrough) | `spdk-nvme-peak` |
| **elbencho** | elbencho | file | vm-file (SSH guest) | `file-parallel-*`, `vm-file-parallel-*` |
| **warp** | MinIO Warp | object | vm-object (VM-hosted S3) | `s3-*`, `vm-s3-*` |
| **sbk** | pgbench / db_bench / kafka-producer-perf-test | application | vm-application (via agent on guest) | `app-*`, `vm-app-*` |
| **mock** | Synthetic | all layers (`--mock`) | all layers (`--mock`) | any profile |

## Layer × deployment

```
Layer            Physical target              Virtual target
─────────────────────────────────────────────────────────────────
block            /dev/nvme0n1                 —
vm-block         —                            root@10.0.1.20:/dev/vdb
file             /mnt/nfs/share               —
vm-file          —                            root@10.0.1.20:/mnt/data
object           10.0.1.10:9000               10.0.1.20:9000 (MinIO in VM)
vm-object        —                            same as object (layer tag)
application      postgres://host/db           agent on VM + --clients
vm-application   —                            10.0.1.20:7777 + localhost DSN
```

## Physical commands (Dell lab)

```bash
# Block — fio
stratabench run --profile nvme-random-oltp --target /dev/nvme0n1
stratabench run --profile ssd-random-4k --target /dev/sda
stratabench run --profile nvme-max-stress --target /dev/nvme0n1

# Block — vdbench / SPDK
stratabench run --profile afa-multi-lun --target /dev/sd{b,c,d}
stratabench run --profile spdk-nvme-peak --target 0000:01:00.0

# File — elbencho
stratabench run --profile file-parallel-read --target /mnt/lustre/test
stratabench run --profile file-parallel-write --target /mnt/lustre/test

# Object — warp
export WARP_ACCESS_KEY=minioadmin WARP_SECRET_KEY=minioadmin
stratabench run --profile s3-put-throughput --target 10.0.1.10:9000
stratabench run --profile s3-cluster-put-get --target 10.0.1.10:9000 --clients 10.0.1.1:7777,10.0.1.2:7777

# Application — sbk
stratabench run --profile app-postgres-tpc-c --target "postgres://bench@10.0.1.30/stratabench"
stratabench run --profile app-kafka-producer --target 10.0.1.30:9092
stratabench run --profile app-rocksdb-read --target /data/rocksdb
```

## Virtual commands (Dell lab)

```bash
# VM block — fio via SSH into guest
stratabench run --profile vm-disk-random --target root@10.0.1.20:/dev/vdb
stratabench run --profile vm-disk-sequential --target root@10.0.1.20:/dev/vdb
stratabench run --profile vm-nvme-oltp --target root@10.0.1.20:/dev/vdb
stratabench run --profile vm-disk-stress --target root@10.0.1.20:/dev/vdb

# VM file — elbencho via SSH
stratabench run --profile vm-file-parallel-read --target root@10.0.1.20:/mnt/data
stratabench run --profile vm-file-parallel-write --target root@10.0.1.20:/mnt/data

# VM object — MinIO inside guest
stratabench run --profile vm-s3-put-throughput --target 10.0.1.20:9000

# VM application — agent on guest, DSN on localhost inside VM
ssh root@10.0.1.20 'stratabench-agent'   # or systemd unit
stratabench run --profile vm-app-postgres \
  --target "postgres://bench@localhost/stratabench" \
  --clients 10.0.1.20:7777
stratabench run --profile vm-app-kafka \
  --target localhost:9092 \
  --clients 10.0.1.20:7777
```

## Distributed (physical clients)

```bash
# On each client node
stratabench-agent

# Coordinator
stratabench run --profile ssd-random-4k --target /dev/nvme0n1 \
  --clients 10.0.1.1:7777,10.0.1.2:7777,10.0.1.3:7777
```

## Profile count

| Layer | Physical | Virtual | Total |
|-------|----------|---------|-------|
| block | 6 | — | 6 |
| vm-block | — | 4 | 4 |
| file | 2 | — | 2 |
| vm-file | — | 2 | 2 |
| object | 5 | — | 5 |
| vm-object | — | 1 | 1 |
| application | 3 | — | 3 |
| vm-application | — | 2 | 2 |
| **Total** | **16** | **9** | **25** |
