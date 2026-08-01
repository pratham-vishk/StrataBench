package lab

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEnvConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lab.env")
	content := `LAB_SSH_USER=bench
LAB_CLIENT_HOSTS=10.0.0.1,10.0.0.2
LAB_S3_ENDPOINTS=10.0.0.10:9000
LAB_AGENT_PORT=7777
WARP_ACCESS_KEY=ak
WARP_SECRET_KEY=sk
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Clients) != 2 || cfg.Clients[0].Host != "10.0.0.1" {
		t.Fatalf("clients=%+v", cfg.Clients)
	}
	if cfg.ServerCSV() != "10.0.0.10:9000" {
		t.Fatalf("servers=%s", cfg.ServerCSV())
	}
	if cfg.S3.AccessKey != "ak" {
		t.Fatal(cfg.S3.AccessKey)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.AgentPort != 7777 || cfg.S3.Deploy != "docker" {
		t.Fatalf("%+v", cfg)
	}
}
