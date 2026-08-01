---
name: stratabench
description: >-
  Run honest storage benchmarks with StrataBench — plan profiles from natural
  language, validate workload and hardware, lab bootstrap + profile-aware lab run,
  and analyze HTML/Excel/PDF reports. Use when the user mentions StrataBench,
  storage benchmarking, IOPS/latency tests, NVMe/AFA/S3 profiles, lab.yaml,
  stratabench MCP tools, or agentic benchmark workflows.
---

# StrataBench (Devin skill)

Same workflow as `.cursor/skills/stratabench/SKILL.md`. For a full step-by-step lab runbook, invoke workflow `/stratabench-lab-benchmark` or delegate to the **stratabench-benchmarker** subagent.

## Quick commands

```bash
stratabench lab bootstrap -f lab.yaml
stratabench lab run -f lab.yaml nvme-random-oltp
stratabench agent "nvme oltp" --target /dev/nvme0n1 --clients 10.0.1.1:7777 --yes
stratabench sbk tools
```

Reports: `~/.stratabench/reports/<run-id>.{html,xlsx,pdf}`

Read `AGENTS.md` for the full MCP tool list and safety defaults.
