#!/usr/bin/env bash
# Full agentic loop on lab cluster (plan → guide → validate → run).
#   ./scripts/lab-agent.sh lab.env "s3 rdma object 3kb-100kb duration 1 hour"
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ENV_FILE="${1:-${ROOT}/lab.env}"
shift || true
INTENT="${*:-physical s3 rdma cluster duration 30 minutes}"
[[ -f "$ENV_FILE" ]] || { echo "missing $ENV_FILE"; exit 1; }
# shellcheck disable=SC1090
source "$ENV_FILE"

LAB_AGENT_PORT="${LAB_AGENT_PORT:-7777}"
STRATABENCH_BIN="${STRATABENCH_BIN:-${ROOT}/bin/stratabench}"

clients_csv() {
	local out=""
	IFS=',' read -ra HOSTS <<< "${LAB_CLIENT_HOSTS:?}"
	for host in "${HOSTS[@]}"; do
		host=$(echo "$host" | tr -d ' ')
		[[ -n "$out" ]] && out+=","
		out+="${host}:${LAB_AGENT_PORT}"
	done
	echo "$out"
}

CLIENTS=$(clients_csv)
TARGET="${LAB_BLOCK_TARGET:-$(echo "${LAB_S3_ENDPOINTS:-}" | cut -d, -f1 | tr -d ' ')}"
TOPO="${LAB_TOPOLOGY:-shard}"

export WARP_ACCESS_KEY="${WARP_ACCESS_KEY:-minioadmin}"
export WARP_SECRET_KEY="${WARP_SECRET_KEY:-minioadmin}"

INTENT_FULL="$INTENT clients $CLIENTS"
[[ -n "${LAB_S3_ENDPOINTS:-}" ]] && INTENT_FULL="$INTENT_FULL servers ${LAB_S3_ENDPOINTS} topology $TOPO"

exec "$STRATABENCH_BIN" agent "$INTENT_FULL" \
	${TARGET:+--target "$TARGET"} \
	--clients "$CLIENTS" \
	--topology "$TOPO" \
	--check-hardware \
	"$@"
