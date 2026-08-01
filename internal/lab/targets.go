package lab

import (
	"fmt"
	"strings"

	"github.com/pratham-vishk/stratabench/internal/profile"
)

// TargetsConfig holds per-layer benchmark targets for a lab cluster.
// Env vars (LAB_BLOCK_TARGET, etc.) override when a field is empty.
type TargetsConfig struct {
	Block       string `yaml:"block"`
	AFALuns     string `yaml:"afa_luns"`
	File        string `yaml:"file"`
	SPDKPCI     string `yaml:"spdk_pci"`
	PostgresDSN string `yaml:"postgres_dsn"`
	Kafka       string `yaml:"kafka"`
	RocksDBPath string `yaml:"rocksdb_path"`
	VMBlock     string `yaml:"vm_block"`
	VMFile      string `yaml:"vm_file"`
}

// RunPlan is the resolved execution plan for lab run.
type RunPlan struct {
	Profile       string
	Layer         string
	Engine        string
	Target        string
	ServerTargets []string
	ClientURLs    []string
	Topology      string
	NeedsS3       bool
	NeedsClients  bool
}

type labTargets struct {
	block      string
	afa        string
	spdk       string
	file       string
	s3         string
	vm         string
	vmS3       string
	postgres   string
	kafka      string
	rocksdb    string
	client     string
	serverList string
}

// ResolveRun builds a profile-aware run plan from lab config.
func (c Config) ResolveRun(p *profile.Profile) (RunPlan, error) {
	if len(c.Clients) == 0 {
		return RunPlan{}, fmt.Errorf("lab config has no clients — add clients: in lab.yaml")
	}
	tgts := c.resolveTargets()
	target := profileTarget(tgts, p)
	if target == "" {
		return RunPlan{}, fmt.Errorf("no target resolved for profile %q (layer=%s engine=%s) — set targets.%s in lab.yaml or LAB_* env",
			p.Name, p.Layer, p.Engine, targetHintKey(p))
	}

	plan := RunPlan{
		Profile:      p.Name,
		Layer:        p.Layer,
		Engine:       p.Engine,
		Target:       target,
		Topology:     inferTopology(p, c),
		NeedsS3:      profileNeedsS3(p),
		NeedsClients: profileNeedsClients(p) || len(c.Clients) > 0,
	}
	for _, cl := range c.Clients {
		port := cl.Port
		if port == 0 {
			port = c.AgentPort
		}
		plan.ClientURLs = append(plan.ClientURLs, fmt.Sprintf("%s:%d", cl.Host, port))
	}
	if profileNeedsObjectServers(p) && len(c.Servers) > 0 {
		for _, s := range c.Servers {
			port := s.Port
			if port == 0 {
				port = 9000
			}
			plan.ServerTargets = append(plan.ServerTargets, fmt.Sprintf("%s:%d", s.Host, port))
		}
	}
	if plan.NeedsS3 && len(c.Servers) == 0 && c.S3.Deploy != "external" {
		return RunPlan{}, fmt.Errorf("profile %q needs S3 — add servers: to lab.yaml or set s3.deploy: external", p.Name)
	}
	return plan, nil
}

func (c Config) resolveTargets() labTargets {
	s3 := c.ServerCSV()
	if s3 == "" {
		s3 = "10.0.1.10:9000"
	}
	firstS3 := strings.Split(s3, ",")[0]
	client := "10.0.1.1:7777"
	vmHostIP := "10.0.1.20"
	if len(c.Clients) > 0 {
		client = fmt.Sprintf("%s:%d", c.Clients[0].Host, c.AgentPort)
		vmHostIP = c.Clients[0].Host
	}
	vmBlock := c.Targets.VMBlock
	if vmBlock == "" {
		vmBlock = fmt.Sprintf("root@%s:/dev/vdb", vmHostIP)
	}
	return labTargets{
		block:      firstNonEmpty(c.Targets.Block, c.BlockTarget(), envOr("LAB_BLOCK_TARGET", "/dev/nvme0n1")),
		afa:        firstNonEmpty(c.Targets.AFALuns, envOr("LAB_AFA_LUNS", "/dev/sdb,/dev/sdc,/dev/sdd")),
		spdk:       firstNonEmpty(c.Targets.SPDKPCI, envOr("LAB_SPDK_PCI", "0000:01:00.0")),
		file:       firstNonEmpty(c.Targets.File, envOr("LAB_FILE_TARGET", "/mnt/nfs/share")),
		s3:         firstS3,
		vm:         vmBlock,
		vmS3:       envOr("LAB_VM_S3_TARGET", firstS3),
		postgres:   firstNonEmpty(c.Targets.PostgresDSN, envOr("LAB_POSTGRES_DSN", "postgres://bench@localhost/stratabench")),
		kafka:      firstNonEmpty(c.Targets.Kafka, envOr("LAB_KAFKA_TARGET", "10.0.1.30:9092")),
		rocksdb:    firstNonEmpty(c.Targets.RocksDBPath, envOr("LAB_ROCKSDB_PATH", "/data/rocksdb")),
		client:     client,
		serverList: s3,
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func profileNeedsS3(p *profile.Profile) bool {
	return p.Layer == "object" || p.Layer == "vm-object" || p.Engine == "warp" || p.Engine == "gosbench"
}

func profileNeedsObjectServers(p *profile.Profile) bool {
	if p.Engine != "warp" {
		return false
	}
	return strings.Contains(p.Name, "cluster") ||
		p.ParamInt("warp_clients", 0) > 0 ||
		len(p.ParamStringSlice("warp_clients")) > 0
}

func inferTopology(p *profile.Profile, cfg Config) string {
	if cfg.DefaultRun.Topology != "" && cfg.DefaultRun.Profile == p.Name {
		return cfg.DefaultRun.Topology
	}
	switch p.Layer {
	case "object":
		if strings.Contains(p.Name, "cluster") {
			return "shard"
		}
		return "single"
	case "block", "file", "application":
		if len(cfg.Clients) > 1 {
			return "pool"
		}
		return "single"
	default:
		return "single"
	}
}

func targetHintKey(p *profile.Profile) string {
	switch p.Layer {
	case "block":
		if p.Engine == "vdbench" {
			return "afa_luns"
		}
		if p.Engine == "spdk" {
			return "spdk_pci"
		}
		return "block"
	case "file":
		return "file"
	case "object":
		return "servers (S3)"
	case "application":
		return "postgres_dsn / kafka / rocksdb_path"
	default:
		return "block"
	}
}

// NeedsMinIO returns true when bootstrap should deploy MinIO on server nodes.
func (c Config) NeedsMinIO() bool {
	if c.S3.Deploy == "skip" || c.S3.Deploy == "external" {
		return false
	}
	return len(c.Servers) > 0
}

func profileTarget(tgts labTargets, p *profile.Profile) string {
	switch p.Layer {
	case "block":
		if p.Engine == "vdbench" || p.Name == "afa-multi-lun" {
			return tgts.afa
		}
		if p.Engine == "spdk" {
			return tgts.spdk
		}
		return tgts.block
	case "file":
		return tgts.file
	case "object":
		return tgts.s3
	case "application":
		return appTarget(tgts, p)
	case "vm-block":
		return vmBlockTarget(tgts, p)
	case "vm-afa":
		return fmt.Sprintf("root@%s:/dev/sdb,/dev/sdc,/dev/sdd", vmHost(tgts))
	case "vm-file":
		return vmFileTarget(tgts)
	case "vm-object":
		return tgts.vmS3
	case "vm-application":
		return vmAppTarget(tgts, p)
	default:
		return tgts.block
	}
}

func vmHost(tgts labTargets) string {
	vm := tgts.vm
	if i := strings.Index(vm, "@"); i >= 0 {
		vm = vm[i+1:]
	}
	if j := strings.Index(vm, ":"); j >= 0 {
		return vm[:j]
	}
	if j := strings.Index(vm, "/"); j >= 0 {
		return vm[:j]
	}
	return vm
}

func vmBlockTarget(tgts labTargets, p *profile.Profile) string {
	host := vmHost(tgts)
	if strings.Contains(p.Name, "passthrough") {
		return fmt.Sprintf("root@%s:/dev/nvme0n1", host)
	}
	return fmt.Sprintf("root@%s:/dev/vdb", host)
}

func vmFileTarget(tgts labTargets) string {
	if strings.Contains(tgts.vm, "/mnt/") {
		return tgts.vm
	}
	return fmt.Sprintf("root@%s:/mnt/data", vmHost(tgts))
}

func appTarget(tgts labTargets, p *profile.Profile) string {
	switch {
	case strings.Contains(p.Name, "postgres"):
		return tgts.postgres
	case strings.Contains(p.Name, "kafka"):
		return tgts.kafka
	case strings.Contains(p.Name, "rocksdb"):
		return tgts.rocksdb
	default:
		return tgts.postgres
	}
}

func vmAppTarget(tgts labTargets, p *profile.Profile) string {
	if strings.Contains(p.Name, "postgres") {
		return "postgres://bench@localhost/db"
	}
	return "localhost:9092"
}
