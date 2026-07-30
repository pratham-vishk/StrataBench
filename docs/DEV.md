# StrataBench Development Guide

## Prerequisites

- Go 1.22+
- Linux or WSL2 for real `fio` benchmarks
- `fio` (optional): `sudo apt install fio`

## Build

```bash
make build
# or
go build -o bin/stratabench ./cmd/stratabench
```

## Commands

```bash
# List profiles
./bin/stratabench profiles

# Plan from natural language
./bin/stratabench plan "benchmark nvme for oltp database"

# Validate (honest test rules)
./bin/stratabench validate --profile nvme-random-oltp --cache-bytes 10737418240

# Run mock benchmark (no hardware needed)
./bin/stratabench run --profile ssd-random-4k --target /tmp/test --mock

# Run real fio (Linux/WSL, needs target path)
./bin/stratabench run --profile hdd-sequential-read --target /tmp/stratabench.dat

# Report
./bin/stratabench report --run-id <uuid>
```

## Data layout

```
.stratabench/
  stratabench.db    # SQLite run history
  reports/          # HTML reports
  work/             # fio job files, temp data
```

## Mock mode

Use `--mock` on Windows or when hardware is unavailable. Validator and reports work the same; results are synthetic.

## Environment variables

| Variable | Purpose |
|----------|---------|
| `STRATABENCH_ROOT` | Repo root (auto-detected) |
| `STRATABENCH_MOCK_CACHE_BYTES` | Override cache size for validation |

## Tests

```bash
go test ./...
```

## Dell lab (later)

Deploy `stratabench-agent` on client VMs (Phase 2). Phase 1 is single-node CLI only.
