# Release Guide

## v0.8.0-rc20 (current)

Live interval streaming for gosbench (stdout parsing) and sbk (pgbench `-P 1` progress); mock synthetic intervals for both.

### Tag and publish

```bash
git tag v0.8.0-rc20
git push origin v0.8.0-rc20
gh release create v0.8.0-rc20 --title "v0.8.0-rc20" --notes "Live interval streaming for gosbench and sbk."
```

### Smoke test

```bash
stratabench run --profile s3-gosbench-write --target 10.0.0.1:9000 --mock --watch
stratabench run --profile sbk-postgresql-oltp --target postgres://localhost/bench --mock --watch
```

## v0.8.0-rc19

Kubernetes operator runs benchmarks as Jobs; `stratabench apply --status-out`; `lab validate --smoke-sbk`.

## v0.8.0-rc18

Full 33-profile lab validation matrix generated from `profiles/`; `--smoke-all` and JSON sign-off report.

### Tag and publish

```bash
git tag v0.8.0-rc18
git push origin v0.8.0-rc18
gh release create v0.8.0-rc18 --title "v0.8.0-rc18" --notes "Full lab validate matrix; smoke-all; JSON report."
```

### Smoke test

```bash
stratabench lab validate -f lab.yaml --smoke-all
cat lab-validation.json
```

## v0.8.0-rc17

Docker ships Rust `stratabench-engine`; `s3-gosbench-read` profile; doc sync to 33 profiles; `cargo test` in CI.

### Tag and publish

```bash
git tag v0.8.0-rc17
git push origin v0.8.0-rc17
gh release create v0.8.0-rc17 --title "v0.8.0-rc17" --notes "Rust engine in Docker; s3-gosbench-read; docs + cargo test."
```

### Smoke test

```bash
docker build -t stratabench:rc17 .
docker run --rm stratabench:rc17 /usr/local/bin/stratabench-engine version
stratabench run --profile s3-gosbench-read --target 127.0.0.1:9000 --mock
```

## v0.8.0-rc16

Batched io_uring (QD>1 in-flight per thread) and `stratabench lab validate` for Dell hardware sign-off.

### Tag and publish

```bash
git tag v0.8.0-rc16
git push origin v0.8.0-rc16
gh release create v0.8.0-rc16 --title "v0.8.0-rc16" --notes "Batched io_uring QD>1; lab validate command; Dell lab checklist updates."
```

### Smoke test

```bash
make build-rust
export STRATABENCH_ENGINE_BIN=$PWD/crates/stratabench-engine/target/release/stratabench-engine
stratabench run --profile block-native-io_uring --target /dev/nvme0n1   # Linux + root
stratabench lab validate -f lab.yaml --smoke
```

## v0.8.0-rc15

Rust io_uring block path (`io_engine: io_uring`) with profile `block-native-io_uring`.

### Tag and publish

```bash
git tag v0.8.0-rc15
git push origin v0.8.0-rc15
gh release create v0.8.0-rc15 --title "v0.8.0-rc15" --notes "Rust io_uring block engine path; block-native-io_uring profile."
```

## v0.8.0-rc14

Native engine live progress: `progress_path` JSONL polled by orchestrator for Prometheus/SSE during async runs.

### Tag and publish

```bash
git tag v0.8.0-rc14
git push origin v0.8.0-rc14
gh release create v0.8.0-rc14 --title "v0.8.0-rc14" --notes "Native engine live progress via progress_path JSONL."
```

### Smoke test

```bash
make build-engine
stratabench run --profile block-native-oltp --target /dev/nvme0n1 --async --watch --mock  # use real engine without --mock on Linux
```

## v0.8.0-rc13

Rust native engine v0.2: Linux O_DIRECT block I/O with interval buckets; profile `block-native-oltp`.

### Tag and publish

```bash
git tag v0.8.0-rc13
git push origin v0.8.0-rc13
```

### Smoke test

```bash
make build-rust
export STRATABENCH_ENGINE_BIN=$PWD/crates/stratabench-engine/target/release/stratabench-engine
stratabench run --profile block-native-oltp --target /dev/nvme0n1   # Linux + root
stratabench run --profile nvme-random-oltp --target /dev/null --mock --async --watch
```

## v0.8.0-rc12

Warp live interval streaming via stdout parsing and benchdata analysis; S3 runs get interval time-series in reports.

### Tag and publish

```bash
git tag v0.8.0-rc12
git push origin v0.8.0-rc12
```

### Smoke test

```bash
make build
stratabench run --profile s3-put-throughput --target 127.0.0.1:9000 --async --watch   # requires warp + MinIO
stratabench run --profile nvme-random-oltp --target /dev/null --mock --async --watch
```

## v0.8.0-rc11

fio live interval tailing: real block benchmarks stream IOPS/throughput to Prometheus and SSE while running.

### Tag and publish

```bash
git tag v0.8.0-rc11
git push origin v0.8.0-rc11
```

### Smoke test

```bash
make build
stratabench run --profile ssd-random-4k --target /tmp/fio.dat --async --watch   # requires fio
stratabench run --profile nvme-random-oltp --target /dev/null --mock --async --watch
```

## v0.8.0-rc10

Live interval streaming for mock runs: Prometheus live gauges, SSE `interval` events, watch CLI shows IOPS/MBps/latency mid-run.

### Tag and publish

```bash
git tag v0.8.0-rc10
git push origin v0.8.0-rc10
```

### Smoke test

```bash
make build
./bin/stratabench run --profile nvme-random-oltp --target /dev/null --mock --async --watch
curl -N http://localhost:8080/api/v1/runs/<run_id>/stream   # expect event: interval lines
```

## v0.8.0-rc9

Platform hardening release: multi-node reports, Postgres, GOSBench, mTLS, async runs, native engine stubs, 14 MCP tools, live monitoring.

### Tag and publish

```bash
git tag v0.8.0-rc9
git push origin v0.8.0-rc9
```

### Smoke test

```bash
make build
./bin/stratabench run --profile nvme-random-oltp --target /dev/null --mock --async --watch
./bin/stratabench-engine version
curl -N http://localhost:8080/api/v1/runs/<run_id>/stream   # with API running
```

## v0.7.0-rc1

- [x] Public repository
- [x] Published container image (GHCR)
- [x] Docs site live — https://pratham-vishk.github.io/StrataBench/
- [x] Full Kubernetes operator
- [x] 33 profiles, all engines, all topologies
- [x] Physical + virtual coverage (HDD, NVMe, AFA, S3 RDMA)
- [ ] Dell lab validation on real hardware
- [ ] Native SBK drivers validated on Linux (pgbench, db_bench, kafka)
