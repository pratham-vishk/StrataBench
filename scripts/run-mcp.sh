#!/usr/bin/env bash
# Launch stratabench-mcp for Devin / Claude / Cursor MCP clients.
# Prefers a built binary; falls back to `go run`.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
if [[ -x "$ROOT/bin/stratabench-mcp" ]]; then
  exec "$ROOT/bin/stratabench-mcp"
fi
if [[ -f "$ROOT/bin/stratabench-mcp.exe" ]]; then
  exec "$ROOT/bin/stratabench-mcp.exe"
fi
exec go run ./cmd/stratabench-mcp
