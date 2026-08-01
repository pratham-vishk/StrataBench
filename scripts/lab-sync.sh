#!/usr/bin/env bash
exec "$(cd "$(dirname "$0")/.." && pwd)/bin/stratabench" lab sync -f "${1:-lab.yaml}" "${@:2}"
