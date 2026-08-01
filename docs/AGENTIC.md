# Agentic StrataBench — CLI Models & MCP

Use StrataBench with **Cursor**, **Claude Desktop**, **Claude Code**, or any agent that supports **MCP** or shell tools.

## Architecture

```
┌─────────────────┐     MCP (stdio)      ┌──────────────────┐
│  Cursor / Claude │ ◄──────────────────► │ stratabench-mcp  │
│  CLI model       │                      │  (7 tools)       │
└────────┬────────┘                      └────────┬─────────┘
         │ shell                                  │
         ▼                                        ▼
┌─────────────────┐                      ┌──────────────────┐
│ stratabench CLI │                      │ agentloop        │
│ plan/agent/run  │                      │ planner/validator│
└─────────────────┘                      │ orchestrator     │
                                         └──────────────────┘
```

## 1. MCP (recommended for IDE agents)

### Build

```bash
make build-mcp
# binary: bin/stratabench-mcp
```

### Cursor

Copy `examples/mcp-cursor.json` into Cursor MCP settings, or add:

```json
{
  "mcpServers": {
    "stratabench": {
      "command": "C:/path/to/StrataBench/bin/stratabench-mcp.exe",
      "env": { "STRATABENCH_DATA": "C:/Users/you/.stratabench" }
    }
  }
}
```

### Claude Desktop

See `examples/mcp-claude-desktop.json` — same pattern under `mcpServers`.

### Example agent prompts

- *"Use stratabench to plan a benchmark for nvme database oltp"*
- *"List StrataBench profiles for S3 RDMA"*
- *"Run a mock stratabench agent loop for ssd random 4k on /tmp/test"*
- *"Validate nvme-random-oltp against /dev/nvme0n1 with hardware checks"*

## 2. REST API

```bash
stratabench-api   # :8080

curl -X POST http://localhost:8080/api/v1/plan \
  -H 'Content-Type: application/json' \
  -d '{"intent":"nvme oltp database","use_llm":true}'

curl -X POST http://localhost:8080/api/v1/agent \
  -H 'Content-Type: application/json' \
  -d '{"intent":"ssd random 4k","target":"/tmp/test","mock":true}'
```

## 3. CLI for shell agents

```bash
stratabench plan "s3 cluster rdma" --llm
stratabench agent "afa multi lun flash array" --target /dev/sdb --mock
stratabench agent "nvme stress test" --target /dev/nvme0n1 --llm --mock=false
```

## LLM providers

| Provider | Setup |
|----------|-------|
| **Ollama** (local) | `ollama serve` + `export OLLAMA_MODEL=llama3.2` |
| **OpenAI** | `export OPENAI_API_KEY=sk-...` |
| **OpenAI-compatible** | `export STRATABENCH_LLM_URL=https://your-proxy/v1` + API key |
| **Auto** | Uses OpenAI if API key set, else Ollama |

Flags: `--llm` or `--ollama` on `plan` and `agent` commands.

## Kubernetes agentic mode

```yaml
spec:
  intent: "nvme oltp database workload"
  target: /dev/nvme0n1
  useOllama: true
```

See `examples/benchmark-intent.yaml`.

## Repo conventions for coding agents

- `AGENTS.md` — machine-oriented onboarding (this repo's agent contract)
- `llms.txt` — concise index for LLM crawlers
- `.cursor/skills/stratabench/SKILL.md` — Cursor project skill
- `agents/planner.prompt` / `agents/reporter.prompt` — edit to tune LLM behavior

## Safety defaults

| Surface | Default |
|---------|---------|
| MCP `stratabench_run` | `mock: true` |
| MCP `stratabench_agent` | `mock: true` |
| CLI `agent` | `--mock` default true |
| CLI `run` | `--mock` default false |

Always validate before real hardware runs. See [HARDWARE-VALIDATION.md](HARDWARE-VALIDATION.md).
