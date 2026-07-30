# Contributing to StrataBench

Thank you for your interest in contributing. StrataBench is designed for open-source release — your contributions help build the platform storage teams actually need.

## Ways to contribute

- **Workload profiles** — add or improve YAML profiles for real-world workloads
- **Engine wrappers** — fio, Warp, SPDK, vdbench adapters
- **Validator rules** — new honesty checks for benchmark design
- **Documentation** — guides, examples, architecture notes
- **Rust engine** — block, file, S3 I/O improvements
- **Agent prompts** — planner/analyst agent quality

## Development setup

> Setup instructions will be added in Phase 1 when code scaffolding lands.

```bash
git clone https://github.com/<org>/StrataBench.git
cd StrataBench
# Rust engine
cd crates/stratabench-engine && cargo build
# Go CLI
cd cmd/stratabench && go build
```

## Code style

- **Rust:** `rustfmt`, `clippy` clean
- **Go:** `gofmt`, standard Go conventions
- **YAML profiles:** validated against profile schema before PR
- **Docs:** clear, practical — explain *why*, not just *what*

## Pull request process

1. Fork the repository
2. Create a feature branch (`feature/add-warp-wrapper`)
3. Write tests where applicable
4. Update docs if behavior changes
5. Open PR with clear description and test plan

## Profile contributions

New profiles must:
- Include `validation` block with appropriate rules
- Have a realistic `description` explaining the workload intent
- Not use hero-number settings (tiny block size + extreme QD without justification)
- Pass `stratabench validate --profile your-profile.yaml`

## License

By contributing, you agree your contributions are licensed under Apache License 2.0.
