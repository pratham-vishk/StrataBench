# Agentic StrataBench — CLI Models & MCP

Use StrataBench with **Cursor**, **Claude Code**, **Claude Desktop**, **Devin**, or any agent that supports **MCP** or shell tools.

## Architecture

```
┌──────────────────────────┐     MCP (stdio)      ┌──────────────────┐
│ Cursor / Claude / Devin  │ ◄──────────────────► │ stratabench-mcp  │
│ CLI model                │                      │  (7 tools)       │
└────────┬─────────────────┘                      └────────┬─────────┘
         │ shell                                           │
         ▼                                                   ▼
┌─────────────────┐                               ┌──────────────────┐
│ stratabench CLI │                               │ agentloop        │
│ plan/agent/run  │                               │ planner/validator│
└─────────────────┘                               │ orchestrator     │
                                                  └──────────────────┘
```

## Platform setup

| Platform | Committed config | First-time steps |
|----------|------------------|------------------|
| **Claude Code** | `.mcp.json` | Open repo → approve MCP server (`/mcp`) |
| **Devin** | `.devin/mcp_config.json`, `AGENTS.md`, `CLAUDE.md` | Clone repo → MCP loads automatically |
| **Claude Desktop** | `examples/mcp-claude-desktop.json` | Merge into Desktop config |
| **Cursor** | `examples/mcp-cursor.json` | Add to Cursor MCP settings |

---

## Claude Code

Claude Code reads **`.mcp.json`** at the project root (already committed in this repo).

```bash
cd StrataBench
claude    # or: claude mcp list
```

On first session, approve the `stratabench` server when prompted. Check status with `/mcp`.

**Alternative (user scope):**

```bash
claude mcp add --scope project --transport stdio stratabench -- \
  go run ./cmd/stratabench-mcp
```

**Windows:** use `powershell -File scripts/run-mcp.ps1` instead of `go run` if Go is not on PATH.

**Files Claude reads:** `CLAUDE.md`, `AGENTS.md`, `.mcp.json`

---

## Devin

Devin reads **AGENTS.md** and **CLAUDE.md** automatically, plus project MCP from **`.devin/mcp_config.json`**.

```bash
# On Devin's Ubuntu VM after repo clone
make build-mcp
bash scripts/run-mcp.sh   # test MCP server starts
```

**Committed Devin files:**

| File | Purpose |
|------|---------|
| `.devin/mcp_config.json` | Project MCP servers |
| `.devin/config.json` | Import rules from Claude/Cursor/AGENTS.md |
| `AGENTS.md` | Agent contract (agents.md standard) |
| `CLAUDE.md` | Short project guide |

**Local overrides (gitignored):** `.devin/mcp_config.local.json`, `.devin/config.local.json`

**Example Devin session prompts:**

- *"Use the stratabench MCP to list NVMe profiles and plan an oltp benchmark"*
- *"Run a mock stratabench agent loop for ssd random 4k on /tmp/test"*
- *"Validate afa-multi-lun against /dev/sdb,/dev/sdc with hardware checks"*

---

## Claude Desktop

Merge `examples/mcp-claude-desktop.json` into your Desktop config:

| OS | Config path |
|----|-------------|
| macOS | `~/Library/Application Support/Claude/claude_desktop_config.json` |
| Windows | `%APPDATA%\Claude\claude_desktop_config.json` |

After `make build-mcp`, set `command` to the absolute path of `bin/stratabench-mcp`.

---

## Cursor

See `examples/mcp-cursor.json`. Add under Cursor → Settings → MCP.

---

## REST API

```bash
stratabench-api   # :8080

curl -X POST http://localhost:8080/api/v1/plan \
  -H 'Content-Type: application/json' \
  -d '{"intent":"nvme oltp database","use_llm":true}'

curl -X POST http://localhost:8080/api/v1/agent \
  -H 'Content-Type: application/json' \
  -d '{"intent":"ssd random 4k","target":"/tmp/test","mock":true}'
```

## CLI for shell agents

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

- `AGENTS.md` — machine-oriented onboarding (agents.md standard; Devin + Cursor)
- `CLAUDE.md` — Claude Code + Devin project guide
- `llms.txt` — concise index for LLM crawlers
- `.mcp.json` — Claude Code project MCP (committed)
- `.devin/mcp_config.json` — Devin project MCP (committed)
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
