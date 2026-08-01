package lab

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// BootstrapReport summarizes install actions per host.
type BootstrapReport struct {
	Hosts []HostResult `json:"hosts"`
}

type HostResult struct {
	Host    string `json:"host"`
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

// Bootstrap installs tools + agent on client nodes and optionally MinIO on servers.
func Bootstrap(ctx context.Context, cfg Config, repoRoot string) (*BootstrapReport, error) {
	r := Runner{SSHUser: cfg.SSH.User, SSHKey: cfg.SSH.Key}
	report := &BootstrapReport{}

	if err := EnsureCredentials(&cfg); err != nil {
		return report, err
	}

	binDir := cfg.BinDir
	if !filepath.IsAbs(binDir) {
		binDir = filepath.Join(repoRoot, binDir)
	}
	profilesDir := cfg.ProfilesDir
	if !filepath.IsAbs(profilesDir) {
		profilesDir = filepath.Join(repoRoot, profilesDir)
	}
	agentBin := filepath.Join(binDir, "stratabench-agent")
	if _, err := os.Stat(agentBin); err != nil {
		return report, fmt.Errorf("missing %s — run 'make build' first", agentBin)
	}

	// Clients: warp, fio, agent
	for _, n := range cfg.Clients {
		hr := HostResult{Host: n.Host}
		if err := pushAndInstall(ctx, r, cfg, n.Host, binDir, profilesDir); err != nil {
			hr.Message = err.Error()
		} else {
			hr.OK = true
			hr.Message = "agent + tools installed"
		}
		report.Hosts = append(report.Hosts, hr)
	}

	// Servers: MinIO when S3 deploy is enabled
	if cfg.NeedsMinIO() {
		for _, n := range cfg.Servers {
			already := false
			for _, c := range cfg.Clients {
				if c.Host == n.Host {
					already = true
					break
				}
			}
			if already {
				continue // same host already bootstrapped as client
			}
			hr := HostResult{Host: n.Host}
			if err := DeployMinIO(ctx, r, cfg, n.Host); err != nil {
				hr.Message = "minio: " + err.Error()
			} else {
				hr.OK = true
				hr.Message = "minio deployed"
			}
			report.Hosts = append(report.Hosts, hr)
		}
		// MinIO on client+server same nodes
		for _, n := range cfg.Servers {
			for _, c := range cfg.Clients {
				if c.Host == n.Host {
					hr := HostResult{Host: n.Host + " (s3)"}
					if err := DeployMinIO(ctx, r, cfg, n.Host); err != nil {
						hr.Message = "minio: " + err.Error()
					} else {
						hr.OK = true
						hr.Message = "minio on colocated node"
					}
					report.Hosts = append(report.Hosts, hr)
					break
				}
			}
		}
	}

	if cfg.Firewall.Apply {
		for _, host := range uniqueHosts(cfg.ClientHosts(), cfg.ServerHosts()) {
			_ = ApplyFirewall(ctx, r, host, cfg.Firewall.OpenPorts)
		}
	}

	return report, nil
}

func uniqueHosts(lists ...[]string) []string {
	seen := map[string]bool{}
	var out []string
	for _, list := range lists {
		for _, h := range list {
			if !seen[h] {
				seen[h] = true
				out = append(out, h)
			}
		}
	}
	return out
}

func pushAndInstall(ctx context.Context, r Runner, cfg Config, host, binDir, profilesDir string) error {
	stage := "/tmp/stratabench-staging"
	_, _ = r.RunRemote(ctx, host, fmt.Sprintf("sudo mkdir -p %s/bin %s/profiles", stage, stage))
	files := []string{
		filepath.Join(binDir, "stratabench"),
		filepath.Join(binDir, "stratabench-agent"),
	}
	if err := r.SCP(ctx, host, files, stage+"/bin/"); err != nil {
		return err
	}
	if prof, err := filepath.Glob(filepath.Join(profilesDir, "*.yaml")); err == nil && len(prof) > 0 {
		_ = r.SCP(ctx, host, prof, stage+"/profiles/")
	}
	script := installScript(cfg)
	_, err := r.RunRemote(ctx, host, script)
	return err
}

func installScript(cfg Config) string {
	vdbench := ""
	if cfg.Tools.InstallVdbench && cfg.Tools.VdbenchPath != "" {
		vdbench = fmt.Sprintf("sudo ln -sf %s /usr/local/bin/vdbench", cfg.Tools.VdbenchPath)
	}
	fioInstall := "true"
	if !cfg.Tools.InstallFio {
		fioInstall = "false"
	}
	return fmt.Sprintf(`set -e
INSTALL_DIR=%q
WARP_VERSION=%q
AGENT_PORT=%d
ACCESS=%q
SECRET=%q
INSTALL_FIO=%s
stage=/tmp/stratabench-staging

if command -v apt-get &>/dev/null; then
  sudo DEBIAN_FRONTEND=noninteractive apt-get update -qq
  sudo DEBIAN_FRONTEND=noninteractive apt-get install -y -qq curl ca-certificates fio smartmontools nvme-cli openssh-client docker.io 2>/dev/null || \
  sudo DEBIAN_FRONTEND=noninteractive apt-get install -y -qq curl ca-certificates fio smartmontools nvme-cli openssh-client
elif command -v dnf &>/dev/null; then
  sudo dnf install -y curl ca-certificates fio smartmontools nvme-cli openssh-clients docker 2>/dev/null || \
  sudo dnf install -y curl ca-certificates fio smartmontools nvme-cli openssh-clients
fi

if [ "$INSTALL_FIO" = "false" ]; then true; fi

if ! command -v warp &>/dev/null; then
  tmp=$(mktemp -d)
  curl -fsSL "https://github.com/minio/warp/releases/download/${WARP_VERSION}/warp_Linux_amd64.tar.gz" -o "$tmp/warp.tgz"
  tar -xzf "$tmp/warp.tgz" -C "$tmp"
  sudo install -m 0755 "$tmp/warp" /usr/local/bin/warp
  rm -rf "$tmp"
fi
%s

sudo mkdir -p "$INSTALL_DIR/bin" "$INSTALL_DIR/profiles" "$INSTALL_DIR/data"
sudo cp -a "$stage/bin/"* "$INSTALL_DIR/bin/" 2>/dev/null || true
sudo cp -a "$stage/profiles/"* "$INSTALL_DIR/profiles/" 2>/dev/null || true
sudo chmod +x "$INSTALL_DIR/bin/"*

sudo tee /etc/systemd/system/stratabench-agent.service >/dev/null <<UNIT
[Unit]
Description=StrataBench agent
After=network.target
[Service]
Type=simple
Environment=STRATABENCH_AGENT_LISTEN=:${AGENT_PORT}
Environment=STRATABENCH_ROOT=${INSTALL_DIR}
Environment=STRATABENCH_DATA=${INSTALL_DIR}/data
Environment=WARP_ACCESS_KEY=${ACCESS}
Environment=WARP_SECRET_KEY=${SECRET}
ExecStart=${INSTALL_DIR}/bin/stratabench-agent
Restart=on-failure
[Install]
WantedBy=multi-user.target
UNIT
sudo systemctl daemon-reload
sudo systemctl enable stratabench-agent
sudo systemctl restart stratabench-agent
echo ok`,
		cfg.InstallDir, cfg.Tools.WarpVersion, cfg.AgentPort,
		cfg.S3.AccessKey, cfg.S3.SecretKey, fioInstall, vdbench)
}

// Sync pushes rebuilt binaries to all clients (code-change loop).
func Sync(ctx context.Context, cfg Config, repoRoot string) error {
	r := Runner{SSHUser: cfg.SSH.User, SSHKey: cfg.SSH.Key}
	binDir := cfg.BinDir
	if !filepath.IsAbs(binDir) {
		binDir = filepath.Join(repoRoot, binDir)
	}
	for _, n := range cfg.Clients {
		remote := cfg.InstallDir + "/bin/"
		if err := r.SCP(ctx, n.Host, []string{
			filepath.Join(binDir, "stratabench"),
			filepath.Join(binDir, "stratabench-agent"),
		}, remote); err != nil {
			return err
		}
		_, err := r.RunRemote(ctx, n.Host, "sudo systemctl restart stratabench-agent")
		if err != nil {
			return err
		}
	}
	return nil
}

func PrintBootstrapReport(rep *BootstrapReport) {
	for _, h := range rep.Hosts {
		tag := "OK"
		if !h.OK {
			tag = "FAIL"
		}
		fmt.Printf("[%s] %s: %s\n", tag, h.Host, h.Message)
	}
}
