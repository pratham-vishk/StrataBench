#!/usr/bin/env bash
# Wrapper — prefer: stratabench lab bootstrap -f lab.yaml
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ENV_FILE="${1:-${ROOT}/lab.yaml}"
[[ -f "$ENV_FILE" ]] || ENV_FILE="${ROOT}/lab.env"
exec "${ROOT}/bin/stratabench" lab bootstrap -f "$ENV_FILE" "${@:2}"
