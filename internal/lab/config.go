package lab

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config describes a benchmark lab cluster (clients, S3 servers, SSH, tools).
type Config struct {
	SSH         SSHConfig         `yaml:"ssh"`
	InstallDir  string            `yaml:"install_dir"`
	AgentPort   int               `yaml:"agent_port"`
	Clients     []Node            `yaml:"clients"`
	Servers     []Node            `yaml:"servers"`
	S3          S3Config          `yaml:"s3"`
	Tools       ToolsConfig       `yaml:"tools"`
	Firewall    FirewallConfig    `yaml:"firewall"`
	DefaultRun  DefaultRunConfig  `yaml:"default_run"`
	BinDir      string            `yaml:"bin_dir"` // local coordinator binaries
	ProfilesDir string            `yaml:"profiles_dir"`
}

type SSHConfig struct {
	User string `yaml:"user"`
	Key  string `yaml:"key,omitempty"`
}

type Node struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port,omitempty"`
	Role string `yaml:"role,omitempty"` // client, s3, block
}

type S3Config struct {
	Deploy     string `yaml:"deploy"` // docker, external, skip
	AccessKey  string `yaml:"access_key"`
	SecretKey  string `yaml:"secret_key"`
	Bucket     string `yaml:"bucket"`
}

type ToolsConfig struct {
	WarpVersion    string `yaml:"warp_version"`
	InstallFio     bool   `yaml:"install_fio"`
	InstallVdbench bool   `yaml:"install_vdbench"`
	VdbenchPath    string `yaml:"vdbench_path,omitempty"`
}

type FirewallConfig struct {
	Apply     bool  `yaml:"apply"`
	OpenPorts []int `yaml:"open_ports"`
}

type DefaultRunConfig struct {
	Profile  string `yaml:"profile"`
	Topology string `yaml:"topology"`
}

func DefaultConfig() Config {
	return Config{
		SSH:        SSHConfig{User: "root"},
		InstallDir: "/opt/stratabench",
		AgentPort:  7777,
		S3: S3Config{
			Deploy:    "docker",
			AccessKey: "minioadmin",
			SecretKey: "minioadmin",
			Bucket:    "stratabench-test",
		},
		Tools: ToolsConfig{
			WarpVersion: "v1.1.0",
			InstallFio:  true,
		},
		Firewall: FirewallConfig{
			OpenPorts: []int{7777, 9000, 9001},
		},
		DefaultRun: DefaultRunConfig{
			Profile:  "s3-cluster-rdma",
			Topology: "shard",
		},
		BinDir:      "bin",
		ProfilesDir: "profiles",
	}
}

func LoadConfig(path string) (Config, error) {
	cfg := DefaultConfig()
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return cfg, err
		}
	case ".env":
		if err := applyEnvFile(&cfg, string(data)); err != nil {
			return cfg, err
		}
	default:
		return cfg, fmt.Errorf("unsupported config format %q (use .yaml or .env)", ext)
	}
	cfg.normalize()
	return cfg, nil
}

func applyEnvFile(cfg *Config, content string) error {
	vars := map[string]string{}
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		vars[strings.TrimSpace(k)] = strings.Trim(strings.TrimSpace(v), `"'`)
	}
	if u := vars["LAB_SSH_USER"]; u != "" {
		cfg.SSH.User = u
	}
	if k := vars["LAB_SSH_KEY"]; k != "" {
		cfg.SSH.Key = k
	}
	if d := vars["INSTALL_DIR"]; d != "" {
		cfg.InstallDir = d
	}
	if p := vars["LAB_AGENT_PORT"]; p != "" {
		fmt.Sscanf(p, "%d", &cfg.AgentPort)
	}
	if h := vars["LAB_CLIENT_HOSTS"]; h != "" {
		cfg.Clients = hostsToNodes(h, cfg.AgentPort, "client")
	}
	if s := vars["LAB_S3_ENDPOINTS"]; s != "" {
		cfg.Servers = endpointsToNodes(s, 9000, "s3")
	}
	if ak := vars["WARP_ACCESS_KEY"]; ak != "" {
		cfg.S3.AccessKey = ak
	}
	if sk := vars["WARP_SECRET_KEY"]; sk != "" {
		cfg.S3.SecretKey = sk
	}
	if wv := vars["WARP_VERSION"]; wv != "" {
		cfg.Tools.WarpVersion = wv
	}
	if pr := vars["LAB_PROFILE"]; pr != "" {
		cfg.DefaultRun.Profile = pr
	}
	if tp := vars["LAB_TOPOLOGY"]; tp != "" {
		cfg.DefaultRun.Topology = tp
	}
	if bd := vars["STRATABENCH_BIN"]; bd != "" {
		cfg.BinDir = filepath.Dir(bd)
	}
	if cfg.S3.Deploy == "" {
		cfg.S3.Deploy = "docker"
	}
	return nil
}

func hostsToNodes(csv string, port int, role string) []Node {
	var out []Node
	for _, h := range strings.Split(csv, ",") {
		h = strings.TrimSpace(h)
		if h == "" {
			continue
		}
		host, p := splitHostPort(h, port)
		out = append(out, Node{Host: host, Port: p, Role: role})
	}
	return out
}

func endpointsToNodes(csv string, defaultPort int, role string) []Node {
	return hostsToNodes(csv, defaultPort, role)
}

func splitHostPort(addr string, defaultPort int) (string, int) {
	if i := strings.LastIndex(addr, ":"); i > 0 {
		var p int
		fmt.Sscanf(addr[i+1:], "%d", &p)
		if p > 0 {
			return addr[:i], p
		}
	}
	return addr, defaultPort
}

func (c *Config) normalize() {
	if c.SSH.User == "" {
		c.SSH.User = "root"
	}
	if c.InstallDir == "" {
		c.InstallDir = "/opt/stratabench"
	}
	if c.AgentPort == 0 {
		c.AgentPort = 7777
	}
	for i := range c.Clients {
		if c.Clients[i].Port == 0 {
			c.Clients[i].Port = c.AgentPort
		}
	}
	for i := range c.Servers {
		if c.Servers[i].Port == 0 {
			c.Servers[i].Port = 9000
		}
	}
	if c.S3.AccessKey == "" {
		c.S3.AccessKey = "minioadmin"
	}
	if c.S3.SecretKey == "" {
		c.S3.SecretKey = "minioadmin"
	}
	if c.S3.Deploy == "" {
		c.S3.Deploy = "docker"
	}
	if c.Tools.WarpVersion == "" {
		c.Tools.WarpVersion = "v1.1.0"
	}
	if len(c.Firewall.OpenPorts) == 0 {
		c.Firewall.OpenPorts = []int{c.AgentPort, 9000, 9001}
	}
}

func (c Config) ClientCSV() string {
	return nodesCSV(c.Clients)
}

func (c Config) ServerCSV() string {
	return nodesCSV(c.Servers)
}

func nodesCSV(nodes []Node) string {
	var parts []string
	for _, n := range nodes {
		parts = append(parts, fmt.Sprintf("%s:%d", n.Host, n.Port))
	}
	return strings.Join(parts, ",")
}

func (c Config) ClientHosts() []string {
	var h []string
	for _, n := range c.Clients {
		h = append(h, n.Host)
	}
	return h
}

func (c Config) ServerHosts() []string {
	var h []string
	for _, n := range c.Servers {
		h = append(h, n.Host)
	}
	return h
}

func (c Config) PrimaryTarget() string {
	if len(c.Servers) > 0 {
		s := c.Servers[0]
		return fmt.Sprintf("%s:%d", s.Host, s.Port)
	}
	return ""
}

func SaveConfig(path string, cfg Config) error {
	cfg.normalize()
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
