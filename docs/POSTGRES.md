# PostgreSQL store

StrataBench persists runs, baselines, and inventory in **SQLite** by default (`~/.stratabench/stratabench.db`). For shared lab deployments, use **PostgreSQL** instead.

## Configuration

```bash
export STRATABENCH_DATABASE_URL=postgres://user:pass@host:5432/stratabench?sslmode=require
stratabench run --profile nvme-random-oltp --target /dev/nvme0n1 --mock
```

The API, MCP server, and agents use the same variable when they call `store.OpenDefault`.

## Schema

Tables mirror SQLite:

| Table | Purpose |
|-------|---------|
| `runs` | JSON `RunResult` blobs + indexed columns |
| `baselines` | Profile + target key → reference run |
| `hardware_inventory` | Hardware snapshots |
| `smart_history` | SMART reading history |

Migrations run automatically on first connect.

## Testing

```bash
STRATABENCH_DATABASE_URL=postgres://... go test ./internal/store -run Postgres
```

## Fallback

Without `STRATABENCH_DATABASE_URL`, behavior is unchanged (local SQLite). Export runs via `stratabench export` or MCP `stratabench_export_json` for external aggregation.
