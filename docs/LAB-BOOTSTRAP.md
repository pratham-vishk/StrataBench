# Lab bootstrap — one cluster, full stack

StrataBench can **own the full lab loop**: discover nodes → install tools → deploy MinIO → open firewall → benchmark → re-sync after code changes.

## What the repo handles now

| Task | Command |
|------|---------|
| Profile / YAML / params | Built-in `profiles/` + NLP `guide` / `agent` |
| Install warp, fio on nodes | `stratabench lab bootstrap` |
| vdbench | Detect + optional `tools.vdbench_path` symlink (manual tarball) |
| Multi-node topology | `topology: shard` in lab.yaml + `lab run` |
| Deploy MinIO / S3 | `stratabench lab deploy-s3` or `bootstrap` (docker) |
| Remote agent runs | `stratabench-agent` via bootstrap + `lab run --clients` |
| Auto-discover cluster | `stratabench lab discover --hosts 10.0.1.1,10.0.1.2,...` |
| Warp download | Automatic on each node during bootstrap |
| Credentials | `lab.yaml` `s3.access_key` / `secret_key` (+ env export on run) |
| Firewall | `stratabench lab firewall` or `bootstrap --firewall` |
| Hardware / tool check | `stratabench lab check` |
| Code-change loop | `make build && stratabench lab sync` |
| Results store | SQLite via `lab run` / `stratabench run` |

## Quick start (new cluster)

On your **Linux jump host** (SSH access to all nodes):

```bash
git clone https://github.com/pratham-vishk/StrataBench.git
cd StrataBench
cp examples/lab.yaml.example lab.yaml
# edit clients + servers IPs

make build
stratabench lab bootstrap -f lab.yaml --write-env lab.env
stratabench lab check -f lab.yaml
stratabench lab run -f lab.yaml s3-cluster-rdma
```

Or with **auto-discovery** when you only have a host list:

```bash
stratabench lab init
stratabench lab discover -f lab.yaml --hosts 10.0.1.1,10.0.1.2,10.0.1.10,10.0.1.11
make build
stratabench lab bootstrap -f lab.yaml --firewall
stratabench lab check -f lab.yaml
```

## After code changes (your daily loop)

```bash
make build
stratabench lab sync -f lab.yaml
stratabench lab run -f lab.yaml
# or agentic:
stratabench agent "s3 rdma object 3kb-100kb 1 hour" --clients $(grep LAB_CLIENT lab.env) --yes
```

## Config files

| File | Format |
|------|--------|
| `lab.yaml` | Preferred — clients, servers, tools, firewall |
| `lab.env` | Shell-style — see `examples/lab.env.example` |

## External S3 (no MinIO deploy)

If the cluster already has PowerScale / ECS / MinIO:

```yaml
s3:
  deploy: external
servers:
  - host: 10.0.1.10
    port: 9000
```

Only install agents on clients: `bootstrap` skips MinIO when `deploy: external`.

## vdbench (AFA)

vdbench is not redistributable — install on nodes manually, then:

```yaml
tools:
  install_vdbench: true
  vdbench_path: /opt/vdbench/vdbench
```

## Prerequisites

- **Coordinator**: Linux, `ssh`, `scp`, `curl`, `make`, Go 1.25+
- **Nodes**: passwordless SSH as `ssh.user` (default `root`)
- **Docker** on S3 server nodes (if `s3.deploy: docker`)

## Shell scripts (optional)

Scripts in `scripts/lab-*.sh` wrap the same flow for CI; prefer `stratabench lab` CLI.

## Troubleshooting

| Issue | Fix |
|-------|-----|
| `warp not found` | `stratabench lab bootstrap` |
| Agent unreachable | `ufw` / firewall: `stratabench lab firewall --firewall` |
| Warp auth error | Match `WARP_ACCESS_KEY` in lab.yaml and MinIO |
| vdbench missing | Install tarball; set `vdbench_path` |

See also [DELL-LAB.md](DELL-LAB.md) and [DELL-LAB-VALIDATION.md](DELL-LAB-VALIDATION.md).
