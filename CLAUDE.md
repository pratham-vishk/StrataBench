# StrataBench — Claude Code / Devin project guide

This file is read automatically by **Claude Code** and **Devin** (along with `AGENTS.md`).

## What this repo does

Agentic storage benchmarking: plan workloads from natural language, validate honesty rules + hardware, run via native engines (fio, warp, vdbench, SPDK, elbencho, gosbench, SBK) or mock mode, analyze results. Lab workflow: one `lab.yaml` → bootstrap → profile-aware `lab run` → HTML/Excel/PDF reports.

## MCP tools (preferred)

Project MCP is configured in **`.mcp.json`** (Claude Code) and **`.devin/mcp_config.json`** (Devin).

After clone, open Claude Code in this directory and approve the `stratabench` MCP server when prompted (`/mcp` to check status).

| Tool | Use when |
|------|----------|
| `stratabench_list_profiles` | User asks what benchmarks exist |
| `stratabench_plan` | Map intent → profile name |
| `stratabench_guide` | Discuss ambiguous intent before run |
| `stratabench_validate` | Pre-flight before real hardware |
| `stratabench_run` | Execute one profile (`mock: true` unless user confirms hardware) |
| `stratabench_agent` | Full plan → validate → run → analyze → report |
| `stratabench_analyze` | Explain a completed `run_id` |
| `stratabench_list_runs` | Show recent results |
| `stratabench_compare_runs` | Compare two runs |
| `stratabench_report` | HTML report for a run |
| `stratabench_baseline_check` | Regression vs baseline |
| `stratabench_export_json` / `stratabench_import_json` | JSON export/import |
| `stratabench_run_progress` | Poll in-flight distributed run |

## Lab cluster (real hardware)

```bash
# User gives IPs once → agent writes lab.yaml with targets:
stratabench lab bootstrap -f lab.yaml
stratabench lab validate -f lab.yaml --check-sbk-tools
stratabench lab run -f lab.yaml nvme-random-oltp   # picks /dev/nvme* from targets
stratabench lab run -f lab.yaml s3-cluster-rdma      # picks S3 endpoints + topology
```

Reports: `~/.stratabench/reports/<run-id>.{html,xlsx,pdf}`

## Build

```bash
make build          # all binaries
make build-mcp      # MCP server only
make test
```

## CLI shortcuts

```bash
./bin/stratabench profiles
./bin/stratabench plan "nvme oltp" --llm
./bin/stratabench agent "ssd random 4k" --target /tmp/test --mock
```

## Safety (required)

- Default **mock mode** for MCP runs unless the user explicitly names a real target
- Run `stratabench_validate` with `--check-hardware` before production storage
- Never commit secrets (`.env`, API keys, WARP credentials)

## More docs

- `AGENTS.md` — full agent contract
- `docs/AGENTIC.md` — Claude, Devin, Cursor setup
- `.devin/workflows/stratabench-lab-benchmark.md` — Devin `/stratabench-lab-benchmark` workflow
- `docs/HARDWARE-VALIDATION.md` — per-use-case hardware checks
