# Release Guide

## v0.4.0-rc1 (current)

Release candidate for public OSS launch.

### Tag and publish

```bash
git tag v0.4.0-rc1
git push origin v0.4.0-rc1
```

This triggers:
- **Docker** workflow → `ghcr.io/pratham-vishk/stratabench:0.4.0-rc1`
- **Pages** workflow → docs site

### Enable GitHub Pages

Settings → Pages → Build and deployment → **GitHub Actions**

### Make repository public

When ready for external contributors:
1. Settings → General → Change visibility → Public
2. Verify no secrets in git history
3. Announce with README and docs link

### Smoke test after release

```bash
docker pull ghcr.io/pratham-vishk/stratabench:0.4.0-rc1
docker run --rm ghcr.io/pratham-vishk/stratabench:0.4.0-rc1 version
docker run --rm ghcr.io/pratham-vishk/stratabench:0.4.0-rc1 agent "ssd random 4k" --target test --mock
```

### v1.0.0 criteria

- [ ] Public repository
- [ ] Published container image
- [ ] Docs site live
- [ ] Dell lab validation on real hardware
- [ ] Native SBK drivers validated on Linux (pgbench, db_bench)
- [ ] Full Kubernetes operator (optional — CRD + apply shipped in rc1)
