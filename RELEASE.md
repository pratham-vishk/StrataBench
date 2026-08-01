# Release Guide

## v0.8.0-rc13 (current)

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
- [x] 30 profiles, all engines, all topologies
- [x] Physical + virtual coverage (HDD, NVMe, AFA, S3 RDMA)
- [ ] Dell lab validation on real hardware
- [ ] Native SBK drivers validated on Linux (pgbench, db_bench, kafka)
