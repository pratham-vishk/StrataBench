# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| master  | yes       |

## Reporting a Vulnerability

If you discover a security vulnerability in StrataBench, please report it responsibly:

1. **Do not** open a public GitHub issue for security vulnerabilities.
2. Email the maintainer with a description of the issue, steps to reproduce, and potential impact.
3. Allow reasonable time for a fix before public disclosure.

We will acknowledge receipt within 5 business days and aim to provide a fix or mitigation plan within 30 days for confirmed issues.

## Scope

In scope:
- StrataBench CLI, agent, and API server
- Credential handling (Warp, SSH, database DSNs in profiles)
- Remote agent HTTP API (`stratabench-agent`)

Out of scope:
- Underlying benchmark tools (fio, vdbench, Warp, etc.) — report to those projects directly
- Misconfiguration of storage systems under test
