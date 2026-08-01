# AGENTS.md — StrataBench for AI Agents

This repository is **agent-ready**. External CLI models (**Cursor**, **Claude Code**, **Claude Desktop**, **Devin**, Windsurf, etc.) can drive StrataBench via **MCP tools**, **REST API**, or the **CLI**.

## Agent platform quick reference

| Platform | Config file | Docs |
|----------|-------------|------|
| **Claude Code** | `.mcp.json` (project root, committed) | [docs/AGENTIC.md](docs/AGENTIC.md#claude-code) |
| **Claude Desktop** | `~/Library/Application Support/Claude/claude_desktop_config.json` | `examples/mcp-claude-desktop.json` |
| **Devin** | `.devin/mcp_config.json` + `AGENTS.md` / `CLAUDE.md` | [docs/AGENTIC.md](docs/AGENTIC.md#devin) |
| **Cursor** | Cursor MCP settings | `examples/mcp-cursor.json` |

Clone the repo → Claude Code and Devin pick up MCP automatically. Approve the `stratabench` server on first use.

## Quick start for agents

1. **Discover profiles** — `stratabench_list_profiles` (MCP) or `GET /api/v1/profiles`
2. **Plan from intent** — `stratabench_plan` with `intent: "nvme oltp database"`
3. **Discuss / guide** — `stratabench_guide` when intent is ambiguous (questions, warnings, recommendations)
4. **Validate** — `stratabench_validate` before real hardware runs
5. **Run** — `stratabench_run` (default `mock: true` via MCP for safety)
6. **Full loop** — `stratabench_agent` with `intent` + `target` (blocks on open questions unless `yes: true`)

Prefer **mock mode** unless the user explicitly requests real I/O on known hardware.

## MCP server

Build and register in your MCP client config:

```json
{
  "mcpServers": {
    "stratabench": {
      "command": "stratabench-mcp",
      "args": [],
      "env": {
        "STRATABENCH_DATA": "~/.stratabench"
      }
    }
  }
}
```

See `examples/mcp-cursor.json`, `examples/mcp-claude-desktop.json`, `examples/mcp-claude-code.json`, and `examples/mcp-devin.json`. Project-scoped configs are committed at `.mcp.json` and `.devin/mcp_config.json`.

### MCP tools

| Tool | Purpose |
|------|---------|
| `stratabench_list_profiles` | Catalog of 29 workload profiles |
| `stratabench_plan` | NL intent → profile name (+ guidance summary) |
| `stratabench_guide` | Discuss intent before run — questions, warnings, engine params |
| `stratabench_validate` | Workload + hardware pre-check |
| `stratabench_run` | Execute a profile |
| `stratabench_agent` | Full plan → validate → run → analyze → report |
| `stratabench_analyze` | Post-run insights for a `run_id` |
| `stratabench_list_runs` | Recent runs from SQLite store |

## CLI (for shell-based agents)

```bash
# Discuss before running (all engines)
stratabench guide "nvme object size 3kb-100kb clients 10.0.1.1 servers 10.0.1.10"

# Plan
stratabench plan "nvme oltp" --llm

# Validate + run
stratabench validate --profile nvme-random-oltp --target /dev/nvme0n1
stratabench run --profile nvme-random-oltp --target /dev/nvme0n1

# Agentic loop (pauses on open questions; --yes to accept recommendations)
stratabench agent "ssd random 4k" --target /tmp/test --mock
stratabench agent "nvme database oltp" --target /dev/nvme0n1 --llm --mock=false
stratabench agent "s3 put 3kb-100kb duration 1h" --yes --mock
```

## REST API (`stratabench-api` on `:8080`)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/health` | GET | Liveness |
| `/api/v1/profiles` | GET | List profiles |
| `/api/v1/plan` | POST | `{"intent":"...","use_llm":true}` |
| `/api/v1/runs` | GET/POST | List or create runs |
| `/api/v1/agent` | POST | Full agentic loop |
| `/api/v1/analyze/{run_id}` | GET | Insights |
| `/metrics` | GET | Prometheus |

## LLM configuration

StrataBench uses an LLM for **planning** and **reporting** (not for validation — that is rule-based).

| Variable | Description |
|----------|-------------|
| `STRATABENCH_LLM_PROVIDER` | `ollama`, `openai`, or `auto` (default) |
| `STRATABENCH_LLM_URL` | Base URL (Ollama or OpenAI-compatible) |
| `STRATABENCH_LLM_API_KEY` | API key for OpenAI-compatible endpoints |
| `STRATABENCH_LLM_MODEL` | Model name |
| `OLLAMA_URL` / `OLLAMA_MODEL` | Ollama fallbacks |
| `OPENAI_API_KEY` / `OPENAI_BASE_URL` | OpenAI-compatible fallbacks |

Works with Ollama, OpenAI, LiteLLM, vLLM, LocalAI, and any `/v1/chat/completions` proxy.

## Repository map (for coding agents)

| Path | Role |
|------|------|
| `cmd/stratabench` | Main CLI |
| `cmd/stratabench-mcp` | MCP stdio server for external agents |
| `cmd/stratabench-api` | REST API |
| `cmd/stratabench-agent` | Distributed execution daemon |
| `internal/agentloop` | Plan → validate → run → analyze → report |
| `internal/planner` | Keyword + LLM profile selection |
| `internal/validator` | Honest workload + hardware rules |
| `internal/analyst` | Post-run insight detection |
| `agents/*.prompt` | LLM system prompts (planner, reporter) |
| `profiles/*.yaml` | Workload definitions |
| `docs/AGENTIC.md` | User guide for agentic usage |
| `CLAUDE.md` | Claude Code + Devin project instructions |

## Safety rules for agents

- Default to `--mock` or `mock: true` when hardware is unknown
- Always **validate** before real runs on production storage
- Never destructive: profiles use read-heavy or isolated test patterns; still confirm target with user
- Do not commit secrets (API keys, WARP keys, Postgres DSNs)
- For distributed runs, start `stratabench-agent` on client nodes first

## Docs

- [AGENTIC.md](docs/AGENTIC.md) — setup for Cursor, Claude, Devin, MCP
- [HARDWARE-VALIDATION.md](docs/HARDWARE-VALIDATION.md) — per-use-case hardware checks
- [ENGINE-COVERAGE.md](docs/ENGINE-COVERAGE.md) — profile matrix
- [DEV.md](docs/DEV.md) — build and development
