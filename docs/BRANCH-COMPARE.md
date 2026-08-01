# Branch benchmark comparison

Compare two git branches of **your code** and see storage performance impact — no manual run-ID juggling.

## Quick start

```bash
# One-time setup
stratabench init

# Compare branches (mock — works everywhere)
stratabench compare branches \
  --base main \
  --head feature/storage-opt \
  --profile nvme-random-oltp \
  --target /dev/null \
  --mock \
  --open-report
```

## Production workflow (real hardware)

Benchmark your driver/repo on Linux with NVMe:

```bash
export STRATABENCH_GIT_REPO=/path/to/your/storage-driver

stratabench compare branches \
  --repo $STRATABENCH_GIT_REPO \
  --base main \
  --head feature/zero-copy-read \
  --build-cmd "make -j$(nproc)" \
  --profile nvme-random-oltp \
  --target /dev/nvme0n1 \
  --fail-on-regression \
  --open-report
```

### What happens

1. Saves current git branch
2. **Base branch** → checkout → build → benchmark → store run
3. **Head branch** → checkout → build → benchmark → store run
4. Restores original branch
5. Writes:
   - `.stratabench/reports/compare-<base>-vs-<head>.html` — impact summary
   - Full HTML + Excel for each run
6. Prints IOPS / latency delta table + verdict (improved / regressed / neutral)

## Compare existing runs

```bash
stratabench compare runs --run-id <base-uuid> --run-id-b <head-uuid> --open-report
```

## Git provenance

Every `stratabench run` now records:

- `git_branch`, `git_sha`, dirty flag
- StrataBench version
- Optional `build_cmd`

Shown in reports and used in compare labels.

## CI gate (example)

```yaml
- name: Benchmark regression
  run: |
    stratabench compare branches \
      --base origin/main \
      --head HEAD \
      --profile nvme-random-oltp \
      --target /dev/nvme0n1 \
      --build-cmd "make" \
      --fail-on-regression
```

## Flags

| Flag | Description |
|------|-------------|
| `--base` | Baseline branch (default: `main`) |
| `--head` | Feature branch to test (required) |
| `--repo` | Git repo to build (default: cwd) |
| `--build-cmd` | Build after each checkout |
| `--skip-build` | Skip build step |
| `--allow-dirty` | Allow uncommitted changes |
| `--fail-on-regression` | Exit 1 if head regressed |
| `--mock` | Synthetic I/O (dev/CI) |
