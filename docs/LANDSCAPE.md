# Storage Benchmarking Landscape

Reference document for StrataBench design decisions. Maps existing tools by storage layer and identifies gaps StrataBench addresses.

---

## Layer 1: Raw hardware / block

| Tool | HDD | SSD | NVMe | AFA | Notes |
|------|-----|-----|------|-----|-------|
| **fio** | ✅ | ✅ | ✅ | ✅ | Industry standard; complex config |
| **SPDK perf** | ❌ | ✅ | ✅ | partial | Userspace peak IOPS; no filesystem |
| **vdbench** | ✅ | ✅ | ✅ | ✅ | Enterprise multi-LUN; data verify |
| **elbencho** | ✅ | ✅ | ✅ | partial | Unified file/block/object |
| **hdparm** | ✅ | partial | ❌ | ❌ | Quick sequential read only |
| **Bonnie++** | ✅ | ✅ | partial | ❌ | Single-threaded; outdated |
| **smartmontools** | health | health | health | health | SMART data, not benchmarking |

**Typical performance (4K random)**

| Media | IOPS | Latency |
|-------|------|---------|
| HDD | 50–200 | 5–20 ms |
| SATA SSD | 8K–20K | 0.1–1 ms |
| NVMe SSD | 50K–1M+ | 10–100 µs |
| AFA (block) | 100K–500K+ | sub-ms p99 |

**StrataBench approach:** Own Rust engine for standard block tests; fio for complex patterns; SPDK for peak NVMe; vdbench for AFA.

---

## Layer 2: Virtual machine / hypervisor

| Tool | KVM | VMware | Hyper-V | Notes |
|------|-----|--------|---------|-------|
| **fio** (in guest) | ✅ | ✅ | ✅ | Run inside VM, not on hypervisor |
| **HCIBench** | ❌ | ✅ vSAN | partial | VMware HCI automation |
| **IOmeter** | partial | ✅ | ✅ | Legacy Windows |
| **vdbench** | ✅ | ✅ | ✅ | Multi-VM coordinated |

**Key insight:** VM benchmarks must run inside the guest. Hypervisor-shell tests measure the wrong path.

**StrataBench approach:** Agent deploys fio inside VMs; virtio tuning recommendations in validator.

---

## Layer 3: File / parallel filesystem

| Tool | Local FS | NFS | Lustre | CephFS |
|------|----------|-----|--------|--------|
| **fio** | ✅ | ✅ | ✅ | ✅ |
| **IOzone** | ✅ | ✅ | partial | partial |
| **mdtest / IOR** | partial | partial | ✅ | ✅ |
| **elbencho** | ✅ | ✅ | ✅ | partial |
| **MLPerf Storage** | ✅ | partial | ✅ | partial |

**StrataBench approach:** elbencho wrapper + own file engine for directory tree workloads.

---

## Layer 4: Object / S3 cluster

| Tool | S3 HTTP | Distributed | RDMA | Complex stages |
|------|---------|-------------|------|----------------|
| **Warp** (MinIO) | ✅ | ✅ | ✅ | basic |
| **COSBench** (Intel) | ✅ | ✅ | ❌ | ✅ XML stages |
| **GOSBench** | ✅ | ✅ | ❌ | ✅ |
| **sai3-bench** | ✅ | ✅ | ❌ | ✅ multi-protocol |
| **SBK** (MinIO driver) | ✅ | ✅ | ❌ | basic |
| **httpbench** | ✅ | manual | ❌ | basic |
| **rados bench** | ❌ (RADOS) | ✅ | ❌ | basic |

**StrataBench approach:** Warp for S3 cluster + RDMA; GOSBench for staged workloads; own engine for simple S3 HTTP.

---

## Layer 5: Application / database / messaging

| Tool | Databases | Message queues | KV stores |
|------|-----------|----------------|-----------|
| **SBK** | ✅ 10+ | ✅ 15+ | ✅ RocksDB etc. |
| **HammerDB** | ✅ TPC-C/H | ❌ | ❌ |
| **sysbench** | ✅ MySQL | ❌ | ❌ |
| **MLPerf Storage** | partial | ❌ | ✅ vector DB |

**StrataBench approach:** SBK CSV import in Phase 2; native drivers in Phase 4 if needed.

---

## Reference projects (inspiration, not dependencies)

| Project | URL | What we learn |
|---------|-----|---------------|
| SBK | github.com/kmgowda/SBK | Driver SPI, percentiles, Grafana |
| sbk-charts | github.com/kmgowda/sbk-charts | AI analysis, Excel reports |
| elbencho | github.com/breuner/elbencho | Unified file/block/object |
| sai3-bench | github.com/russfellows/sai3-bench | Multi-protocol Rust, distributed |
| HCIBench | github.com/vmware-labs/hci-benchmark-appliance | VM deploy automation |
| Warp | github.com/minio/warp | S3 distributed + RDMA |
| GOSBench | github.com/mulbc/gosbench | COSBench replacement |
| MLPerf Storage | github.com/mlcommons/storage | AI workload profiles |

---

## Gap analysis: why StrataBench?

| Gap | Existing tools | StrataBench solution |
|-----|----------------|---------------------|
| Cross-layer comparison | Each tool isolated | Unified result schema |
| Honest test validation | Manual expertise | Rule-based validator agent |
| Agentic test design | None | Planner + Analyst agents |
| One CLI for all layers | 10+ different tools | Single `stratabench` CLI |
| Heavy load + lean tool | fio complex; SPDK niche | Own engine + integrate edges |
| Regression tracking | Manual spreadsheets | Built-in baseline comparison |
| NL → test plan | None | Ollama-powered planner |

---

## Integration strategy (not rewrite)

```
StrataBench owns:
  ├── Agent layer (planner, validator, analyst)
  ├── Hardware discovery
  ├── Workload profiles
  ├── Result schema + storage
  ├── Cross-layer comparison
  └── StrataBench engine (Rust) — block, file, S3 HTTP

StrataBench integrates:
  ├── fio      — when: complex block, AFA, exotic patterns
  ├── SPDK     — when: peak NVMe IOPS needed
  ├── Warp     — when: S3 cluster or RDMA
  ├── GOSBench — when: staged S3 workloads
  ├── vdbench  — when: enterprise AFA multi-LUN
  ├── elbencho — when: file/block/object unified distributed
  └── SBK      — when: database/MQ application layer
```

**Rule of thumb:** If an engine took 5+ years to mature, integrate it. If it's our unique value (validation, agents, schema), build it.
