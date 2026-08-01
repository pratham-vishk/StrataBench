# SBK bridge example

Reference Python script for `STRATABENCH_SBK_BRIDGE`. StrataBench invokes:

```bash
sbk_bridge.py run --config <json> --output <json>
```

## Setup

```bash
# Linux / macOS
export STRATABENCH_SBK_BRIDGE="$(pwd)/examples/sbk-bridge/sbk_bridge.py"
chmod +x examples/sbk-bridge/sbk_bridge.py

# Windows (PowerShell)
$env:STRATABENCH_SBK_BRIDGE = "python C:\path\to\StrataBench\examples\sbk-bridge\sbk_bridge.py"
```

## Run

```bash
stratabench run --profile app-postgres-tpc-c --target "postgres://localhost/bench" --mock=false
```

When native `pgbench` is unavailable, StrataBench falls back to the bridge if `STRATABENCH_SBK_BRIDGE` is set.

## Config JSON

```json
{
  "target": "postgres://localhost/bench",
  "profile": "app-postgres-tpc-c",
  "driver": "postgresql",
  "params": { "duration_sec": 60, "threads": 4 }
}
```

## Output JSON

Must match StrataBench `schema.Results` fields (`iops`, `throughput_mbps`, `latency_us`, etc.).

Replace `synthesize()` in `sbk_bridge.py` with your SBK Python API calls.
