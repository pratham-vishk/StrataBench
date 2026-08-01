# Workload Profiles

Declarative YAML definitions for benchmark tests. **30 profiles** across physical and virtual layers.

See [ENGINE-COVERAGE.md](../docs/ENGINE-COVERAGE.md) for the full engine × deployment matrix.

## Layers

| Layer | Deployment | Engines |
|-------|------------|---------|
| `block` | Physical device | fio, vdbench, spdk |
| `vm-block` | SSH into VM guest | fio |
| `vm-afa` | SSH into VM guest (multi-LUN) | vdbench |
| `file` | Physical mount | elbencho |
| `vm-file` | SSH into VM guest | elbencho |
| `object` | S3 endpoint | warp |
| `vm-object` | S3 on VM | warp |
| `application` | Physical service | sbk |
| `vm-application` | Agent on VM guest | sbk |

## All profiles

### Block (physical)

| Profile | Engine | Load |
|---------|--------|------|
| `hdd-sequential-read` | fio | light |
| `ssd-random-4k` | fio | medium |
| `nvme-random-oltp` | fio | heavy |
| `nvme-max-stress` | fio | extreme |
| `spdk-nvme-peak` | spdk | extreme |
| `afa-multi-lun` | vdbench | heavy |

### VM block (virtual)

| Profile | Engine | Load |
|---------|--------|------|
| `vm-disk-random` | fio | medium |
| `vm-hdd-sequential` | fio | light |
| `vm-disk-sequential` | fio | light |
| `vm-nvme-oltp` | fio | heavy |
| `vm-nvme-passthrough` | fio | extreme |
| `vm-disk-stress` | fio | extreme |

### VM AFA (virtual)

| Profile | Engine | Load |
|---------|--------|------|
| `vm-afa-multi-lun` | vdbench | heavy |

### File (physical)

| Profile | Engine | Load |
|---------|--------|------|
| `file-parallel-read` | elbencho | heavy |
| `file-parallel-write` | elbencho | heavy |

### VM file (virtual)

| Profile | Engine | Load |
|---------|--------|------|
| `vm-file-parallel-read` | elbencho | heavy |
| `vm-file-parallel-write` | elbencho | heavy |

### Object (physical)

| Profile | Engine | Load |
|---------|--------|------|
| `s3-put-throughput` | warp | medium |
| `s3-get-throughput` | warp | medium |
| `s3-mixed-workload` | warp | heavy |
| `s3-cluster-put-get` | warp | heavy |
| `s3-cluster-rdma` | warp | heavy |
| `s3-gosbench-write` | gosbench | medium |

### VM object (virtual)

| Profile | Engine | Load |
|---------|--------|------|
| `vm-s3-put-throughput` | warp | medium |
| `vm-s3-rdma` | warp | heavy |

### Application (physical)

| Profile | Engine | Load |
|---------|--------|------|
| `app-postgres-tpc-c` | sbk | heavy |
| `app-kafka-producer` | sbk | heavy |
| `app-rocksdb-read` | sbk | heavy |

### VM application (virtual)

| Profile | Engine | Load |
|---------|--------|------|
| `vm-app-postgres` | sbk | heavy |
| `vm-app-kafka` | sbk | heavy |

## Usage

```bash
# Physical block
stratabench run --profile nvme-random-oltp --target /dev/nvme0n1

# Virtual block (SSH into guest)
stratabench run --profile vm-disk-random --target root@10.0.1.20:/dev/vdb

# Virtual app (agent on guest)
stratabench run --profile vm-app-postgres \
  --target "postgres://bench@localhost/db" --clients 10.0.1.20:7777

# Virtual HDD / NVMe / AFA / S3 RDMA
stratabench run --profile vm-hdd-sequential --target root@10.0.1.20:/dev/vdb
stratabench run --profile vm-nvme-passthrough --target root@10.0.1.20:/dev/nvme0n1
stratabench run --profile vm-afa-multi-lun --target root@10.0.1.20:/dev/sdb,/dev/sdc,/dev/sdd
stratabench run --profile vm-s3-rdma --target 10.0.1.20:9000

# Mock (no tools required)
stratabench run --profile nvme-random-oltp --target /dev/null --mock
```
