# StrataBench Result Schema

All benchmark engines normalize output to this JSON schema. Version: `1.0.0`.

## Top-level result object

```json
{
  "schema_version": "1.0.0",
  "run_id": "550e8400-e29b-41d4-a716-446655440000",
  "profile": "nvme-random-oltp",
  "layer": "block",
  "engine": "fio",
  "status": "completed",
  "validation": {
    "passed": true,
    "rules_checked": ["dataset_gt_cache", "steady_state", "tail_latency", "direct_io"],
    "warnings": []
  },
  "target": {
    "type": "nvme",
    "device": "/dev/nvme0n1",
    "host": "node1.example.com",
    "vm": null,
    "metadata": {
      "model": "Samsung PM9A3",
      "firmware": "GDC5A02Q",
      "capacity_bytes": 1920000000000,
      "numa_node": 0
    }
  },
  "workload": {
    "pattern": "randrw",
    "block_size": "16k",
    "read_write_mix": 70,
    "queue_depth": 32,
    "threads": 4,
    "dataset_size": "200g",
    "duration_sec": 600,
    "ramp_time_sec": 60,
    "direct_io": true
  },
  "results": {
    "iops": 185420,
    "read_iops": 129794,
    "write_iops": 55626,
    "throughput_mbps": 2897.2,
    "latency_us": {
      "min": 45,
      "max": 12500,
      "mean": 172,
      "p50": 120,
      "p75": 185,
      "p90": 245,
      "p95": 310,
      "p99": 450,
      "p99_9": 1200,
      "p99_99": 4500
    },
    "cpu_percent": 45.2,
    "total_bytes_read": 96636764160,
    "total_bytes_written": 41330041728,
    "total_operations": 11125200
  },
  "hardware_snapshot": {
    "cpu_model": "Intel Xeon Gold 6348",
    "cpu_cores": 56,
    "memory_bytes": 257698037760,
    "nic_speed_gbps": 100,
    "rdma_capable": true
  },
  "timestamps": {
    "started_at": "2026-07-30T18:00:00Z",
    "completed_at": "2026-07-30T18:10:00Z",
    "steady_state_reached_at": "2026-07-30T18:01:00Z"
  },
  "raw_engine_output": {
    "path": "runs/550e8400/fio-output.json",
    "format": "fio-json"
  }
}
```

## Field definitions

### `layer` (enum)

| Value | Description |
|-------|-------------|
| `block` | Raw block device (HDD, SSD, NVMe, AFA LUN) |
| `vm-block` | Block device inside virtual machine |
| `file` | File system / parallel FS |
| `object` | S3-compatible object storage |
| `application` | Database, message queue, KV store |

### `engine` (enum)

| Value | Description |
|-------|-------------|
| `stratabench` | Native StrataBench Rust engine |
| `fio` | Flexible I/O Tester |
| `spdk` | SPDK perf |
| `vdbench` | vdbench |
| `warp` | MinIO Warp |
| `gosbench` | GOSBench |
| `elbencho` | elbencho |
| `sbk` | Storage Benchmark Kit (imported) |

### `status` (enum)

| Value | Description |
|-------|-------------|
| `planned` | Test plan created, not yet run |
| `validating` | Validator running |
| `validation_failed` | Validator rejected plan |
| `running` | Benchmark in progress |
| `completed` | Finished successfully |
| `failed` | Engine error or timeout |
| `cancelled` | User cancelled |

### `validation.warnings` (array)

Non-blocking issues the validator flags:

```json
{
  "rule": "runtime_short",
  "message": "Runtime 120s may not reach steady state for this device class",
  "severity": "warning"
}
```

## Comparison object (for cross-run / cross-layer)

```json
{
  "comparison_id": "uuid",
  "runs": ["run-id-1", "run-id-2"],
  "type": "cross_layer",
  "insights": [
    {
      "type": "bottleneck",
      "message": "S3 gateway latency is 562x higher than raw block p99",
      "layer_a": "block",
      "layer_b": "object",
      "metric": "latency_p99",
      "ratio": 562.5
    }
  ]
}
```

## CSV export format (for spreadsheets)

Compatible with sbk-charts-style analysis:

```csv
run_id,profile,layer,engine,host,device,iops,throughput_mbps,latency_p50_us,latency_p99_us,latency_p99_9_us,duration_sec,timestamp
550e8400,nvme-random-oltp,block,fio,node1,/dev/nvme0n1,185420,2897.2,120,450,1200,600,2026-07-30T18:10:00Z
```

## Versioning policy

- Schema version follows semver
- Breaking changes increment major version
- Normalizer must support reading previous major version for regression comparison
