#!/usr/bin/env bash
# Install StrataBench agent + benchmark tools on ONE Linux node.
# Run locally on the node, or via: ssh user@host 'bash -s' < scripts/lab-install-node.sh
set -euo pipefail

INSTALL_DIR="${INSTALL_DIR:-/opt/stratabench}"
BIN_DIR="${INSTALL_DIR}/bin"
PROFILES_DIR="${INSTALL_DIR}/profiles"
WARP_VERSION="${WARP_VERSION:-v1.1.0}"
ARCH="${ARCH:-amd64}"
STRATABENCH_AGENT_LISTEN="${STRATABENCH_AGENT_LISTEN:-:7777}"
INSTALL_WARP="${INSTALL_WARP:-true}"
INSTALL_FIO="${INSTALL_FIO:-true}"
START_AGENT="${START_AGENT:-true}"

log() { echo "[lab-install] $*"; }

install_os_packages() {
	if ! command -v curl &>/dev/null; then
		if command -v apt-get &>/dev/null; then
			sudo apt-get update -qq
			sudo DEBIAN_FRONTEND=noninteractive apt-get install -y -qq curl ca-certificates
		elif command -v dnf &>/dev/null; then
			sudo dnf install -y curl ca-certificates
		fi
	fi
	if [[ "${INSTALL_FIO}" == "true" ]] && ! command -v fio &>/dev/null; then
		log "installing fio..."
		if command -v apt-get &>/dev/null; then
			sudo DEBIAN_FRONTEND=noninteractive apt-get install -y -qq fio smartmontools nvme-cli openssh-client
		elif command -v dnf &>/dev/null; then
			sudo dnf install -y fio smartmontools nvme-cli openssh-clients
		else
			log "WARN: could not install fio automatically — install manually"
		fi
	fi
}

install_warp() {
	if [[ "${INSTALL_WARP}" != "true" ]]; then
		return 0
	fi
	if command -v warp &>/dev/null; then
		log "warp already in PATH: $(command -v warp)"
		return 0
	fi
	local tmp
	tmp=$(mktemp -d)
	trap 'rm -rf "$tmp"' EXIT
	local url="https://github.com/minio/warp/releases/download/${WARP_VERSION}/warp_Linux_${ARCH}.tar.gz"
	log "downloading warp ${WARP_VERSION}..."
	curl -fsSL "$url" -o "${tmp}/warp.tgz"
	tar -xzf "${tmp}/warp.tgz" -C "${tmp}"
	sudo install -m 0755 "${tmp}/warp" /usr/local/bin/warp
	log "warp installed: $(warp --version 2>/dev/null || echo ok)"
}

install_stratabench_binaries() {
	sudo mkdir -p "${BIN_DIR}" "${PROFILES_DIR}" "${INSTALL_DIR}/data"
	# Coordinator may have staged files in /tmp/stratabench-staging
	if [[ -d /tmp/stratabench-staging/bin ]]; then
		sudo cp -a /tmp/stratabench-staging/bin/* "${BIN_DIR}/"
		[[ -d /tmp/stratabench-staging/profiles ]] && sudo cp -a /tmp/stratabench-staging/profiles/* "${PROFILES_DIR}/"
	elif [[ -f ./bin/stratabench-agent ]]; then
		sudo cp -a ./bin/stratabench ./bin/stratabench-agent "${BIN_DIR}/" 2>/dev/null || true
		[[ -d ./profiles ]] && sudo cp -a ./profiles/* "${PROFILES_DIR}/"
	else
		log "WARN: no binaries in ./bin or /tmp/stratabench-staging — run lab-sync.sh from coordinator first"
	fi
	sudo chmod +x "${BIN_DIR}/"* 2>/dev/null || true
}

install_systemd_agent() {
	local unit=/etc/systemd/system/stratabench-agent.service
	if [[ ! -f "${BIN_DIR}/stratabench-agent" ]]; then
		log "WARN: stratabench-agent missing — skip systemd"
		return 0
	fi
	sudo tee "$unit" >/dev/null <<EOF
[Unit]
Description=StrataBench node agent
After=network.target

[Service]
Type=simple
Environment=STRATABENCH_AGENT_LISTEN=${STRATABENCH_AGENT_LISTEN}
Environment=STRATABENCH_ROOT=${INSTALL_DIR}
Environment=STRATABENCH_DATA=${INSTALL_DIR}/data
Environment=WARP_ACCESS_KEY=${WARP_ACCESS_KEY:-minioadmin}
Environment=WARP_SECRET_KEY=${WARP_SECRET_KEY:-minioadmin}
ExecStart=${BIN_DIR}/stratabench-agent
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF
	sudo systemctl daemon-reload
	sudo systemctl enable stratabench-agent
	if [[ "${START_AGENT}" == "true" ]]; then
		sudo systemctl restart stratabench-agent
		sleep 1
		sudo systemctl is-active stratabench-agent && log "agent running on ${STRATABENCH_AGENT_LISTEN}"
	fi
}

log "install dir: ${INSTALL_DIR}"
install_os_packages
install_warp
install_stratabench_binaries
install_systemd_agent
log "done — verify: curl -s http://127.0.0.1:${STRATABENCH_AGENT_LISTEN#:}/v1/health"
