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

**v0.1.0-stub** — synthetic results from profile parameters (same as Go reference stub in `cmd/stratabench-engine`).

Planned: `O_DIRECT` block I/O, libaio/io_uring, S3 HTTP client for object profiles.

See [docs/NATIVE-ENGINE.md](../docs/NATIVE-ENGINE.md) for the JSON contract.
