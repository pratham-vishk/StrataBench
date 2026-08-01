# Release Guide

## v0.5.0-rc1 (current)

Release candidate with in-cluster Kubernetes operator.

### Tag and publish

```bash
git tag v0.5.0-rc1
git push origin v0.5.0-rc1
```

This triggers:
- **Docker** workflow → `ghcr.io/pratham-vishk/stratabench:0.5.0-rc1`
- **Pages** workflow → docs site

### Smoke test after release

```bash
docker pull ghcr.io/pratham-vishk/stratabench:0.5.0-rc1
docker run --rm ghcr.io/pratham-vishk/stratabench:0.5.0-rc1 version
kubectl apply -k deploy/k8s/
kubectl apply -f examples/benchmark-mock.yaml
kubectl get benchmarks -n stratabench -w
```

## v0.4.0-rc1

Initial OSS release candidate — K8s manifests, Docker/GHCR CI, GitHub Pages docs.

### Enable GitHub Pages

Settings → Pages → Build and deployment → **GitHub Actions**

### Make repository public

When ready for external contributors:
1. Settings → General → Change visibility → Public
2. Verify no secrets in git history
3. Announce with README and docs link

### v1.0.0 criteria

- [ ] Public repository
- [ ] Published container image
- [ ] Docs site live
- [ ] Dell lab validation on real hardware
- [ ] Native SBK drivers validated on Linux (pgbench, db_bench)
- [x] Full Kubernetes operator (`stratabench-operator` reconciles Benchmark CRs)
