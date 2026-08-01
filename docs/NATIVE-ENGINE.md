# Native StrataBench engine (external binary)

The `stratabench` engine profile type invokes an external **`stratabench-engine`** binary (planned Rust implementation). Go orchestration handles validation, storage, and reporting; the binary performs raw I/O.

## Discovery

1. `STRATABENCH_ENGINE_BIN` environment variable
2. `stratabench-engine` on `PATH`

Without a binary, non-mock runs fail honestly. Use `--mock` or an external engine profile (`fio`, `warp`, etc.).

## Contract

```bash
stratabench-engine run --config /path/to/config.json --output /path/to/results.json
```

### Config JSON (`native-engine-config.json`)

Generated from the workload profile:

```json
{
  "target": "/dev/nvme0n1",
  "profile": "block-native-oltp",
  "layer": "block",
  "pattern": "randread",
  "block_size": "4k",
  "duration_sec": 60,
  "queue_depth": 32,
  "threads": 4,
  "direct_io": true,
  "progress_path": "/path/to/native-engine-progress.jsonl"
}
```

When `progress_path` is set, the engine appends one JSON interval sample per line (JSONL) during the run. The Go orchestrator polls this file for live Prometheus/SSE metrics.

### Output JSON (`native-engine-results.json`)

Must match `schema.Results`:

```json
{
  "iops": 125000,
  "throughput_mbps": 488,
  "latency_us": {"p50": 120, "p99": 450}
}
```

## Rust engine

```bash
make build-rust
export STRATABENCH_ENGINE_BIN=$PWD/crates/stratabench-engine/target/release/stratabench-engine
```

On **Linux**, the Rust binary performs real block I/O with `O_DIRECT` when `direct_io` is true. Set `io_engine` to `io_uring` for the io_uring submission path (falls back to `pread` on failure). On other platforms it falls back to synthetic results matching the Go stub.

The Rust crate in `crates/stratabench-engine/` implements the same CLI contract as the Go reference stub in `cmd/stratabench-engine/`.

## Go reference stub

Until real I/O lands in Rust, the repo includes a **Go reference stub** at `cmd/stratabench-engine` (built via `make build-engine`). It implements the same CLI contract and produces deterministic synthetic I/O results from profile parameters.

```bash
export STRATABENCH_ENGINE_BIN=/path/to/stratabench-engine
stratabench run --profile ssd-random-4k --target /dev/nvme0n1
```
