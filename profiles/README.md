# Workload Profiles

Declarative YAML definitions for benchmark tests. Profiles are validated by the StrataBench validator before execution.

## Profile structure

```yaml
name: string          # unique profile identifier
version: "1.0"
layer: block | vm-block | file | object | application
engine: stratabench | fio | spdk | vdbench | warp | gosbench | elbencho
description: string
load: light | medium | heavy | extreme

validation:
  require_direct_io: true
  min_runtime_sec: 300
  min_ramp_sec: 60
  require_percentiles: [50, 95, 99, 99.9]
  dataset_vs_cache: gt    # dataset must be greater than cache

params:
  # engine-specific parameters

metrics:
  - iops
  - throughput_mbps
  - latency_p50
  - latency_p99
```

## Built-in profiles (Phase 1)

| Profile | File | Layer | Engine | Load |
|---------|------|-------|--------|------|
| HDD sequential read | `hdd-sequential-read.yaml` | block | fio | light |
| SSD random 4K | `ssd-random-4k.yaml` | block | stratabench | medium |
| NVMe OLTP | `nvme-random-oltp.yaml` | block | fio | heavy |
| S3 PUT throughput | `s3-put-throughput.yaml` | object | stratabench | medium |
| S3 GET throughput | `s3-get-throughput.yaml` | object | stratabench | medium |

## Usage

```bash
# Validate before run
stratabench validate --profile nvme-random-oltp --target /dev/nvme0n1

# Run
stratabench run --profile nvme-random-oltp --target /dev/nvme0n1

# Override params
stratabench run --profile nvme-random-oltp --target /dev/nvme0n1 \
  --override duration_sec=1200
```

## Creating custom profiles

1. Copy an existing profile from this directory
2. Adjust `params` for your workload
3. Run `stratabench validate --profile your-profile.yaml` to check rules
4. Submit via PR to contribute to the built-in library
