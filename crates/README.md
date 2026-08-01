# StrataBench native engine (Rust)

Rust implementation of the `stratabench-engine` binary contract.

## Build

```bash
cargo build --release --manifest-path crates/stratabench-engine/Cargo.toml
# or
make build-rust
```

## Use

```bash
export STRATABENCH_ENGINE_BIN=$PWD/crates/stratabench-engine/target/release/stratabench-engine
stratabench run --profile nvme-random-oltp --target /dev/nvme0n1
```

## Status

**v0.2.0** — Linux block I/O via `O_DIRECT` + `pread`/`pwrite` or `io_uring` (`io_engine: io_uring` in profile params).

```bash
make build-rust
export STRATABENCH_ENGINE_BIN=$PWD/crates/stratabench-engine/target/release/stratabench-engine
stratabench run --profile block-native-oltp --target /dev/nvme0n1
```

Planned: io_uring, libaio, S3 HTTP client for object profiles.

See [docs/NATIVE-ENGINE.md](../docs/NATIVE-ENGINE.md) for the JSON contract.
