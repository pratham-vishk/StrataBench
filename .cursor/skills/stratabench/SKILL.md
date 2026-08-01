---
name: stratabench
description: >-
  Run honest storage benchmarks with StrataBench — plan profiles from natural
  language, validate workload and hardware, execute via fio/warp/vdbench/gosbench/SBK,
  lab bootstrap + profile-aware lab run, and analyze HTML/Excel/PDF reports. Use when
  the user mentions StrataBench, storage benchmarking, IOPS/latency tests, NVMe/AFA/S3
  profiles, lab.yaml, stratabench MCP tools, Claude Code, Devin, or agentic benchmark
  workflows.
---

# StrataBench Agent Skill

## When to use

- User wants to benchmark storage (block, file, object, VM, app)
- User mentions StrataBench CLI, MCP, agent loop, or lab cluster
- User needs profile selection from natural language ("nvme oltp", "s3 rdma cluster")
- User provides client/server IPs for a Dell or multi-node lab

## Preferred workflow

### Lab cluster (real hardware)

1. **Collect IPs** — clients, servers, block devices, S3 endpoints, Postgres DSN as needed
2. **Write `lab.yaml`** — use `examples/lab.yaml.example`; set `targets:` per layer (block, object, sbk, file)
3. **Bootstrap** — `stratabench lab bootstrap -f lab.yaml`
4. **Validate** — `stratabench lab validate -f lab.yaml --check-sbk-tools`
5. **Run** — `stratabench lab run -f lab.yaml <profile>` (target/topology resolved automatically)
6. **Reports** — `~/.stratabench/reports/<run-id>.{html,xlsx,pdf}`

### Single-node / MCP

1. **List profiles** if unsure: MCP `stratabench_list_profiles` or `stratabench profiles`
2. **Plan**: `stratabench plan "<intent>" --llm` or MCP `stratabench_plan`
3. **Guide / discuss** when intent is ambiguous: `stratabench guide "<intent>"` or MCP `stratabench_guide`
4. **Validate**: `stratabench validate --profile <name> --target <target> --check-hardware`
5. **Run**: use `--mock` unless user confirmed real hardware; agent loop blocks on open questions unless `--yes`
6. **Analyze**: `stratabench analyze --run-id <uuid>` or MCP `stratabench_analyze`

## MCP setup

| Platform | Config |
|----------|--------|
| Claude Code | `.mcp.json` (committed) |
| Devin | `.devin/mcp_config.json` |
| Cursor | `examples/mcp-cursor.json` + this skill |

Register `stratabench-mcp` or use committed project configs. Tools are prefixed `stratabench_*` (14 tools).

## Safety

- Default `mock: true` for MCP `stratabench_run` and `stratabench_agent`
- Confirm `/dev/*` paths and production endpoints with user before real runs
- Read `AGENTS.md` and `docs/HARDWARE-VALIDATION.md` for per-use-case requirements

## LLM

Set `STRATABENCH_LLM_PROVIDER=ollama|openai` and model URL/key. `--llm` / `--ollama` enables LLM planner and reporter.

## Key profiles

| Intent | Profile |
|--------|---------|
| HDD sequential | `hdd-sequential-read` |
| SSD/NVMe random | `nvme-random-oltp`, `ssd-random-4k` |
| AFA multi-LUN | `afa-multi-lun` |
| S3 PUT/GET | `s3-put-throughput`, `s3-get-throughput` |
| S3 RDMA cluster | `s3-cluster-rdma` |
| VM guest disk | `vm-disk-random`, `vm-nvme-passthrough` |
| PostgreSQL | `app-postgres-tpc-c` |

Full matrix: `docs/ENGINE-COVERAGE.md`

## SBK native drivers

```bash
stratabench sbk tools
stratabench lab validate -f lab.yaml --check-sbk-tools
```
