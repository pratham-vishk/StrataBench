# Release Guide

## v0.8.0-rc9 (current)

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
