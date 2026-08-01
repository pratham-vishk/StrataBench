# Contributing to StrataBench

Thank you for contributing. StrataBench is an open-source storage benchmarking platform — profiles, engines, validator rules, and agent prompts are all welcome.

## Development setup

```bash
git clone https://github.com/pratham-vishk/StrataBench.git
cd StrataBench
make build
go test ./...
```

On Windows, use `go build -o bin/stratabench.exe ./cmd/stratabench`. Mock mode (`--mock`) works without Linux tools.

### Optional tools (Linux / Dell lab)

| Tool | Purpose |
|------|---------|
| `fio` | Block / VM profiles |
| `warp` | S3 cluster profiles |
| `vdbench` | AFA multi-LUN profiles |
| `nvme` CLI | Hardware inventory NVMe details |
| `ollama` | LLM planner and NL report summaries |

## Ways to contribute

- **Workload profiles** — YAML profiles in `profiles/`
- **Engine wrappers** — `internal/engine/`
- **Validator rules** — `internal/validator/`
- **Agent prompts** — `agents/planner.prompt`, `agents/reporter.prompt`
- **Documentation** — `docs/`

## Code style

- **Go:** `gofmt`, standard conventions, tests for non-trivial logic
- **YAML profiles:** include `validation` block, realistic descriptions
- **Docs:** explain *why*, not just *what*

## Pull request process

1. Fork and create a feature branch
2. Run `go test ./...`
3. Update docs if behavior changes
4. Open a PR with description and test plan

## Profile contributions

New profiles must:
- Include a `validation` block with appropriate honesty rules
- Have a clear `description` of workload intent
- Pass `stratabench validate --profile your-profile`
- Avoid hero-number settings without justification

## License

Contributions are licensed under Apache License 2.0.
