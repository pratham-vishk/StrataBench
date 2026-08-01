---
name: stratabench
description: >-
  Run honest storage benchmarks with StrataBench — plan profiles from natural
  language, validate workload and hardware, execute via fio/warp/vdbench, and
  analyze results. Use when the user mentions StrataBench, storage benchmarking,
  IOPS/latency tests, NVMe/AFA/S3 profiles, stratabench MCP tools, Claude Code,
  Devin, or agentic benchmark workflows.
---

# StrataBench Agent Skill

## When to use

- User wants to benchmark storage (block, file, object, VM, app)
- User mentions StrataBench CLI, MCP, or agent loop
- User needs profile selection from natural language ("nvme oltp", "s3 rdma cluster")

## Preferred workflow

1. **List profiles** if unsure: MCP `stratabench_list_profiles` or `stratabench profiles`
2. **Plan**: `stratabench plan "<intent>" --llm` or MCP `stratabench_plan`
3. **Validate**: `stratabench validate --profile <name> --target <target> --check-hardware`
4. **Run**: use `--mock` unless user confirmed real hardware
5. **Analyze**: `stratabench analyze --run-id <uuid>` or MCP `stratabench_analyze`

## MCP setup

| Platform | Config |
|----------|--------|
| Claude Code | `.mcp.json` (committed) |
| Devin | `.devin/mcp_config.json` |
| Cursor | `examples/mcp-cursor.json` |

Register `stratabench-mcp` or use committed project configs. Tools are prefixed `stratabench_*`.

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
