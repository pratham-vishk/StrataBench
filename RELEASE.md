# Release Guide

## v0.7.0-rc1 (current)

Full platform release: 29 profiles, all topologies, physical + virtual engines, public OSS.

### Tag and publish

```bash
git tag v0.7.0-rc1
git push origin v0.7.0-rc1
```

Triggers: Docker → `ghcr.io/pratham-vishk/stratabench:0.7.0-rc1` · Pages → docs site

### Smoke test

```bash
docker pull ghcr.io/pratham-vishk/stratabench:0.7.0-rc1
docker run --rm ghcr.io/pratham-vishk/stratabench:0.7.0-rc1 version
stratabench run --profile nvme-random-oltp --target /dev/null --mock
stratabench run --profile s3-put-throughput --targets 10.0.1.10:9000,10.0.1.11:9000 --mock --topology sweep
kubectl apply -k deploy/k8s/
```

## v1.0.0 criteria

- [x] Public repository
- [x] Published container image (GHCR)
- [x] Docs site live — https://pratham-vishk.github.io/StrataBench/
- [x] Full Kubernetes operator
- [x] 29 profiles, all engines, all topologies
- [x] Physical + virtual coverage (HDD, NVMe, AFA, S3 RDMA)
- [ ] Dell lab validation on real hardware
- [ ] Native SBK drivers validated on Linux (pgbench, db_bench, kafka)
