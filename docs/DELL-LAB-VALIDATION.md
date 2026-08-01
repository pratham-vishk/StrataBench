# Dell Lab Validation Checklist

Complete validation on Linux VMs before promoting to **v1.0.0**. Covers every engine on **physical** and **virtual** targets.

See [ENGINE-COVERAGE.md](ENGINE-COVERAGE.md) for the full matrix (33 profiles).

## Quick start

```bash
stratabench lab bootstrap -f lab.yaml
stratabench lab check -f lab.yaml
stratabench lab validate -f lab.yaml          # print sign-off matrix + readiness
stratabench lab validate -f lab.yaml --smoke  # + mock smoke (5 profiles)
stratabench lab validate -f lab.yaml --smoke-all  # mock-validate all 33 profiles
stratabench lab validate -f lab.yaml --output lab-signoff.json
```

Override targets with env vars: `LAB_BLOCK_TARGET`, `LAB_AFA_LUNS`, `LAB_FILE_TARGET`, `LAB_POSTGRES_DSN`, `LAB_KAFKA_TARGET`, `LAB_SPDK_PCI`.

## Prerequisites

```bash
# Jump host + client VMs
sudo apt install fio smartmontools nvme-cli postgresql-client openssh-client
# Optional: vdbench, SPDK perf, elbencho, warp, kafka, rocksdb tools

stratabench inventory collect
```

## 0. Hardware validation (per use case)

Before each profile run, validate workload **and** host hardware:

```bash
stratabench validate --profile <name> --target <target> --check-hardware
```

See [HARDWARE-VALIDATION.md](HARDWARE-VALIDATION.md) for the per-use-case matrix (HDD, NVMe, AFA, S3 RDMA, VM, app). Hardware checks are **on by default** for `validate` and `run` (skipped with `--mock`).

| Use case | Hardware checks |
|----------|-----------------|
| HDD | rotational device in inventory |
| NVMe / SSD | NVMe or non-rotational block device |
| AFA | 2+ block LUNs, vdbench in PATH |
| S3 RDMA | RDMA NIC (`rdma link show`) |
| VM | `ssh` in PATH; guest device optional |
| App (pgbench/kafka/rocksdb) | driver binary in PATH, 8GB+ RAM |

## 1. Inventory and SMART (physical)

```bash
stratabench inventory collect
stratabench smart collect
```

## 2. Block — fio (physical)

| Profile | Command | Pass |
|---------|---------|------|
| `hdd-sequential-read` | `stratabench validate --profile hdd-sequential-read --target /dev/sda --check-hardware && stratabench run ...` | |
| `ssd-random-4k` | `stratabench validate --profile ssd-random-4k --target /dev/nvme0n1 --check-hardware && stratabench run ...` | |
| `nvme-random-oltp` | `stratabench validate --profile nvme-random-oltp --target /dev/nvme0n1 --check-hardware && stratabench run ...` | |
| `nvme-max-stress` | `stratabench validate --profile nvme-max-stress --target /dev/nvme0n1 --check-hardware && stratabench run ...` | |

## 2b. Block — native engine (physical, Linux)

| Profile | Command | Pass |
|---------|---------|------|
| `block-native-oltp` | `stratabench run --profile block-native-oltp --target /dev/nvme0n1` | |
| `block-native-io_uring` | `stratabench run --profile block-native-io_uring --target /dev/nvme0n1` | |

Verify live monitoring: `stratabench run --profile block-native-oltp --target /dev/nvme0n1 --async --watch`

## 3. Block — vdbench / SPDK (physical only)

| Profile | Command | Pass |
|---------|---------|------|
| `afa-multi-lun` | `stratabench run --profile afa-multi-lun --target /dev/sd{b,c,d}` | |
| `spdk-nvme-peak` | `stratabench run --profile spdk-nvme-peak --target 0000:01:00.0` | |

## 4. File — elbencho (physical)

| Profile | Command | Pass |
|---------|---------|------|
| `file-parallel-read` | `stratabench run --profile file-parallel-read --target /mnt/nfs/share` | |
| `file-parallel-write` | `stratabench run --profile file-parallel-write --target /mnt/nfs/share` | |

## 5. Object — warp (physical)

```bash
export WARP_ACCESS_KEY=... WARP_SECRET_KEY=...
```

| Profile | Command | Pass |
|---------|---------|------|
| `s3-put-throughput` | `stratabench run --profile s3-put-throughput --target 10.0.1.10:9000` | |
| `s3-get-throughput` | `stratabench run --profile s3-get-throughput --target 10.0.1.10:9000` | |
| `s3-mixed-workload` | `stratabench run --profile s3-mixed-workload --target 10.0.1.10:9000` | |
| `s3-cluster-put-get` | `stratabench run --profile s3-cluster-put-get --target 10.0.1.10:9000 --clients ...` | |
| `s3-cluster-rdma` | `stratabench run --profile s3-cluster-rdma --target 10.0.1.10:9000` | |

## 5b. Object — gosbench (physical)

| Profile | Command | Pass |
|---------|---------|------|
| `s3-gosbench-write` | `stratabench run --profile s3-gosbench-write --target 10.0.1.10:9000` | |
| `s3-gosbench-read` | `stratabench run --profile s3-gosbench-read --target 10.0.1.10:9000` | |

## 6. Application — sbk (physical)

| Profile | Command | Pass |
|---------|---------|------|
| `app-postgres-tpc-c` | `stratabench run --profile app-postgres-tpc-c --target "postgres://bench@host/db"` | |
| `app-kafka-producer` | `stratabench run --profile app-kafka-producer --target 10.0.1.30:9092` | |
| `app-rocksdb-read` | `stratabench run --profile app-rocksdb-read --target /data/rocksdb` | |

## 7. VM block — fio via SSH (virtual)

| Profile | Command | Pass |
|---------|---------|------|
| `vm-hdd-sequential` | `stratabench run --profile vm-hdd-sequential --target root@10.0.1.20:/dev/vdb` | |
| `vm-disk-random` | `stratabench run --profile vm-disk-random --target root@10.0.1.20:/dev/vdb` | |
| `vm-disk-sequential` | `stratabench run --profile vm-disk-sequential --target root@10.0.1.20:/dev/vdb` | |
| `vm-nvme-oltp` | `stratabench run --profile vm-nvme-oltp --target root@10.0.1.20:/dev/vdb` | |
| `vm-nvme-passthrough` | `stratabench run --profile vm-nvme-passthrough --target root@10.0.1.20:/dev/nvme0n1` | |
| `vm-disk-stress` | `stratabench run --profile vm-disk-stress --target root@10.0.1.20:/dev/vdb` | |

## 8. VM AFA — vdbench via SSH (virtual)

| Profile | Command | Pass |
|---------|---------|------|
| `vm-afa-multi-lun` | `stratabench run --profile vm-afa-multi-lun --target root@10.0.1.20:/dev/sdb,/dev/sdc,/dev/sdd` | |

## 9. VM file — elbencho via SSH (virtual)

| Profile | Command | Pass |
|---------|---------|------|
| `vm-file-parallel-read` | `stratabench run --profile vm-file-parallel-read --target root@10.0.1.20:/mnt/data` | |
| `vm-file-parallel-write` | `stratabench run --profile vm-file-parallel-write --target root@10.0.1.20:/mnt/data` | |

## 10. VM object — warp (virtual)

| Profile | Command | Pass |
|---------|---------|------|
| `vm-s3-put-throughput` | `stratabench run --profile vm-s3-put-throughput --target 10.0.1.20:9000` | |
| `vm-s3-rdma` | `stratabench run --profile vm-s3-rdma --target 10.0.1.20:9000` | |

## 11. VM application — sbk via agent (virtual)

```bash
# On guest VM
stratabench-agent
```

| Profile | Command | Pass |
|---------|---------|------|
| `vm-app-postgres` | `stratabench run --profile vm-app-postgres --target "postgres://bench@localhost/db" --clients 10.0.1.20:7777` | |
| `vm-app-kafka` | `stratabench run --profile vm-app-kafka --target localhost:9092 --clients 10.0.1.20:7777` | |

## 12. Live monitoring + regression

```bash
stratabench run --profile nvme-random-oltp --target /dev/null --mock --async --watch
stratabench agent "nvme oltp database" --target /dev/nvme0n1 --check-baseline
stratabench baseline set --run-id <uuid>
stratabench run --profile nvme-random-oltp --target /dev/nvme0n1 --check-baseline
```

## 13. Kubernetes

```bash
kubectl apply -k deploy/k8s/
kubectl apply -f examples/benchmark-mock.yaml
kubectl get benchmarks -n stratabench
```

## Sign-off

| Check | Pass |
|-------|------|
| All engines exercised (fio, vdbench, spdk, elbencho, warp, gosbench, stratabench, sbk, mock) | |
| Native block engine (pread + io_uring) on NVMe | |
| Live monitoring (async + watch + SSE) during at least one run | |
| HDD physical + virtual | |
| NVMe physical + virtual (incl. passthrough) | |
| AFA physical + virtual | |
| S3 RDMA physical + virtual | |
| Validator catches bad workload design | |
| Baseline regression alerts on re-run | |
| Operator sets benchmark status.runId | |
| HTML report readable | |
