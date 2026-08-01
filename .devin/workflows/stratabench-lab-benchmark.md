# StrataBench lab benchmark

Run an end-to-end storage benchmark on a multi-node lab cluster: collect IPs → write `lab.yaml` → bootstrap → validate → profile-aware `lab run` → return HTML/Excel/PDF report paths.

**Invoke:** `/stratabench-lab-benchmark`

**Prerequisites:** Linux jump host with SSH to lab nodes; `stratabench` MCP server connected (`.devin/mcp_config.json`); read `AGENTS.md` and `CLAUDE.md` first.

---

## Step 1 — Understand the request

Ask the user (if not already provided):

- Workload intent (e.g. HDD sequential, NVMe OLTP, AFA multi-LUN, S3 RDMA, Postgres TPC-C)
- Client node IPs (for `stratabench-agent`)
- Server / S3 node IPs (only for object profiles)
- Block device paths (`/dev/sdb`, `/dev/nvme0n1`, comma-separated LUNs for AFA)
- Optional: Postgres DSN, Kafka broker, NFS path, VM SSH target
- SSH user/key if not default

Map intent to a profile using MCP `stratabench_plan` or `stratabench_list_profiles`. Show the chosen profile and resolved target plan before proceeding.

## Step 2 — Write or update `lab.yaml`

Start from `examples/lab.yaml.example`. Set at minimum:

```yaml
clients:
  - host: <client-ip>
targets:
  block: <device>          # HDD/NVMe/fio profiles
  afa_luns: <luns>         # vdbench AFA profiles
  file: <nfs-path>         # elbencho file profiles
  postgres_dsn: <dsn>      # SBK app profiles
servers:                   # only when profile needs S3
  - host: <s3-ip>
    port: 9000
    role: s3
s3:
  deploy: docker           # use skip when S3 not needed; external for existing MinIO/ECS
```

Confirm destructive paths (`/dev/*`) and production endpoints with the user before real runs.

## Step 3 — Build and bootstrap (jump host)

On the Linux jump host inside the repo:

```bash
make build
stratabench lab bootstrap -f lab.yaml --write-env lab.env
stratabench lab check -f lab.yaml
stratabench lab validate -f lab.yaml --check-sbk-tools   # when SBK/app profile
stratabench sbk tools                                     # probe pgbench, db_bench, kafka tools
```

If bootstrap fails, diagnose SSH, firewall (`lab firewall`), or missing tools before continuing.

## Step 4 — Run the benchmark

Profile-aware run (target/topology/engine resolved automatically):

```bash
stratabench lab run -f lab.yaml <profile-name>
```

For single-node or mock exploration without lab hardware, use MCP instead:

- `stratabench_guide` — discuss ambiguous intent
- `stratabench_validate` — pre-flight
- `stratabench_agent` with `mock: true` unless user confirmed real hardware

## Step 5 — Reports and summary

After a successful run, collect and return to the user:

- `run_id`
- `~/.stratabench/reports/<run-id>.html`
- `~/.stratabench/reports/<run-id>.xlsx`
- `~/.stratabench/reports/<run-id>.pdf` (executive summary)

Use MCP `stratabench_analyze` for insights. Optionally `stratabench_compare_runs` against a baseline.

## Step 6 — Daily code-change loop

When the user changes StrataBench code and wants to re-benchmark:

```bash
make build
stratabench lab sync -f lab.yaml
stratabench lab run -f lab.yaml <profile-name>
```

## Safety (required)

- Default to **mock** for MCP runs unless the user explicitly names real lab hardware
- Never commit secrets (S3 keys, DSN passwords, SSH keys) — use `lab.env` locally
- Do not run destructive profiles on production volumes without explicit user approval
- For distributed runs, ensure `stratabench-agent` is running on clients and `STRATABENCH_AGENT_TOKEN` matches

## Delegation

For long research (profile matrix, hardware docs), spawn the **stratabench-benchmarker** explore subagent. For hands-on lab execution with shell + MCP, use **stratabench-benchmarker** general subagent or run this workflow directly.
