package lab

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pratham-vishk/stratabench/internal/engine"
	"github.com/pratham-vishk/stratabench/internal/profile"
	"github.com/pratham-vishk/stratabench/internal/validator"
)

// ValidationItem is one row in the Dell lab hardware sign-off matrix.
type ValidationItem struct {
	Section string `json:"section"`
	Profile string `json:"profile"`
	Engine  string `json:"engine"`
	Layer   string `json:"layer"`
	Target  string `json:"target"`
	Command string `json:"command"`
}

// ValidateReport summarizes lab validation readiness.
type ValidateReport struct {
	Check       *CheckReport         `json:"check,omitempty"`
	SBKTools    *engine.SBKToolReport `json:"sbk_tools,omitempty"`
	Items       []ValidationItem     `json:"items"`
	ProfileCount int             `json:"profile_count"`
	SmokePassed int              `json:"smoke_passed"`
	SmokeFailed int              `json:"smoke_failed"`
	SmokeErrors []string         `json:"smoke_errors,omitempty"`
}

// ValidateOptions configures lab validation.
type ValidateOptions struct {
	ProfilesDir   string
	Smoke         bool
	SmokeAll      bool // validate every profile (mock)
	SmokeSBK      bool // validate SBK app profiles (mock)
	CheckSBKTools bool // probe native SBK drivers on local PATH
}

// ValidationMatrix returns the full Dell lab sign-off checklist generated from profiles/.
func ValidationMatrix(cfg Config, profilesDir string) ([]ValidationItem, error) {
	profiles, err := profile.List(profilesDir)
	if err != nil {
		return nil, err
	}
	sort.Slice(profiles, func(i, j int) bool {
		if profiles[i].Layer != profiles[j].Layer {
			return profiles[i].Layer < profiles[j].Layer
		}
		return profiles[i].Name < profiles[j].Name
	})
	tgts := resolveTargets(cfg)
	var rows []ValidationItem
	for _, p := range profiles {
		target := profileTarget(tgts, p)
		rows = append(rows, ValidationItem{
			Section: validationSection(p),
			Profile: p.Name,
			Engine:  p.Engine,
			Layer:   p.Layer,
			Target:  target,
			Command: profileCommand(cfg, p, target),
		})
	}
	// Live monitoring smoke row (not a profile file)
	rows = append(rows, ValidationItem{
		Section: "monitoring",
		Profile: "nvme-random-oltp",
		Engine:  "mock",
		Layer:   "block",
		Target:  "/dev/null",
		Command: "stratabench run --profile nvme-random-oltp --target /dev/null --mock --async --watch",
	})
	return rows, nil
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

func resolveTargets(cfg Config) labTargets {
	s3 := cfg.ServerCSV()
	if s3 == "" {
		s3 = "10.0.1.10:9000"
	}
	firstS3 := strings.Split(s3, ",")[0]
	client := "10.0.1.1:7777"
	if len(cfg.Clients) > 0 {
		client = fmt.Sprintf("%s:%d", cfg.Clients[0].Host, cfg.AgentPort)
	}
	vmHost := "10.0.1.20"
	if len(cfg.Clients) > 0 {
		vmHost = cfg.Clients[0].Host
	}
	vm := fmt.Sprintf("root@%s:/dev/vdb", vmHost)
	tgts := labTargets{
		block:      cfg.BlockTarget(),
		afa:        envOr("LAB_AFA_LUNS", "/dev/sdb,/dev/sdc,/dev/sdd"),
		spdk:       envOr("LAB_SPDK_PCI", "0000:01:00.0"),
		file:       envOr("LAB_FILE_TARGET", "/mnt/nfs/share"),
		s3:         firstS3,
		vm:         vm,
		vmS3:       envOr("LAB_VM_S3_TARGET", firstS3),
		postgres:   envOr("LAB_POSTGRES_DSN", "postgres://bench@localhost/stratabench"),
		kafka:      envOr("LAB_KAFKA_TARGET", "10.0.1.30:9092"),
		rocksdb:    envOr("LAB_ROCKSDB_PATH", "/data/rocksdb"),
		client:     client,
		serverList: s3,
	}
	return tgts
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
		return `"postgres://bench@localhost/db"`
	}
	return "localhost:9092"
}

func profileCommand(cfg Config, p *profile.Profile, target string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "stratabench run --profile %s --target %s", p.Name, target)
	if profileNeedsClients(p) {
		fmt.Fprintf(&b, " --clients %s", resolveTargets(cfg).client)
	}
	if p.Engine == "warp" && (strings.Contains(p.Name, "cluster") || p.ParamInt("warp_clients", 0) > 0 || len(p.ParamStringSlice("warp_clients")) > 0) {
		tgts := resolveTargets(cfg)
		if tgts.serverList != "" {
			fmt.Fprintf(&b, " --targets %s", tgts.serverList)
		}
	}
	return b.String()
}

func profileNeedsClients(p *profile.Profile) bool {
	if p.Layer == "vm-application" {
		return true
	}
	if p.Engine == "warp" && strings.Contains(p.Name, "cluster") {
		return true
	}
	return false
}

func validationSection(p *profile.Profile) string {
	if p.Engine == "stratabench" {
		return p.Layer + "-native"
	}
	return p.Layer + "-" + p.Engine
}

// Validate runs lab check and optional smoke validation.
func Validate(ctx context.Context, cfg Config, opts ValidateOptions) (*ValidateReport, error) {
	check, err := Check(ctx, cfg)
	if err != nil {
		return nil, err
	}
	items, err := ValidationMatrix(cfg, opts.ProfilesDir)
	if err != nil {
		return nil, err
	}
	rep := &ValidateReport{
		Check:        check,
		Items:        items,
		ProfileCount: len(items) - 1, // exclude monitoring row
	}
	if opts.CheckSBKTools {
		sbk := engine.ProbeSBKDrivers()
		rep.SBKTools = &sbk
	}
	if !opts.Smoke && !opts.SmokeSBK {
		return rep, nil
	}
	profiles, err := profile.List(opts.ProfilesDir)
	if err != nil {
		return nil, err
	}
	if opts.SmokeSBK {
		for _, p := range profiles {
			if p.Engine == "sbk" {
				runSmokeValidate(rep, p)
			}
		}
	}
	if !opts.Smoke {
		return rep, nil
	}
	if opts.SmokeAll {
		for _, p := range profiles {
			runSmokeValidate(rep, p)
		}
	} else {
		for _, name := range []string{"nvme-random-oltp", "s3-put-throughput", "block-native-oltp", "s3-gosbench-read", "app-postgres-tpc-c"} {
			p, err := profile.LoadByName(opts.ProfilesDir, name)
			if err != nil {
				rep.SmokeFailed++
				rep.SmokeErrors = append(rep.SmokeErrors, err.Error())
				continue
			}
			runSmokeValidate(rep, p)
		}
	}
	return rep, nil
}

func runSmokeValidate(rep *ValidateReport, p *profile.Profile) {
	v := validator.Validate(p, validator.Options{Mock: true, Target: "/dev/null"})
	if v.Passed {
		rep.SmokePassed++
	} else {
		rep.SmokeFailed++
		rep.SmokeErrors = append(rep.SmokeErrors, fmt.Sprintf("%s: %v", p.Name, v.Errors))
	}
}

func PrintValidationReport(rep *ValidateReport) {
	if rep.Check != nil {
		PrintCheckReport(rep.Check, Config{})
		fmt.Println()
	}
	fmt.Printf("Dell lab validation matrix (%d profiles + monitoring):\n", rep.ProfileCount)
	cur := ""
	for _, it := range rep.Items {
		if it.Section != cur {
			cur = it.Section
			fmt.Printf("\n[%s]\n", cur)
		}
		fmt.Printf("  %-28s %-12s %s\n", it.Profile, it.Engine, it.Command)
	}
	fmt.Println("\nSign-off (mark in docs/DELL-LAB-VALIDATION.md):")
	fmt.Println("  [ ] All engines: fio, warp, gosbench, stratabench, sbk, vdbench, spdk, elbencho, mock")
	fmt.Println("  [ ] Live monitoring: async + watch + SSE during mock run")
	fmt.Println("  [ ] Native engine: block-native-oltp + block-native-io_uring on NVMe")
	fmt.Println("  [ ] SBK: app-postgres-tpc-c, app-kafka-producer, app-rocksdb-read")
	fmt.Println("  [ ] Baseline regression on nvme-random-oltp")
	if rep.SmokePassed+rep.SmokeFailed > 0 {
		fmt.Printf("\nSmoke validation: %d passed, %d failed\n", rep.SmokePassed, rep.SmokeFailed)
		for _, e := range rep.SmokeErrors {
			fmt.Printf("  ! %s\n", e)
		}
	}
	if rep.SBKTools != nil {
		fmt.Println("\nSBK native drivers (local PATH):")
		for _, d := range rep.SBKTools.Drivers {
			status := "missing"
			if d.Available {
				status = "ok (" + d.Path + ")"
			}
			fmt.Printf("  %-12s %-28s %s\n", d.Driver, d.Tool, status)
		}
		if rep.SBKTools.AllAvailable {
			fmt.Println("  sbk-tools: all native drivers available")
		} else {
			fmt.Println("  sbk-tools: install missing drivers before application-layer hardware runs")
		}
	}
	if rep.Check != nil && rep.Check.Ready {
		fmt.Println("\nlab-validate: infrastructure READY — execute matrix on hardware")
	} else {
		fmt.Println("\nlab-validate: infrastructure NOT READY — fix lab check first")
	}
}

// WriteValidationReportJSON writes the validation report for sign-off tracking.
func WriteValidationReportJSON(path string, rep *ValidateReport) error {
	data, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// BlockTarget returns configured block device or default.
func (c Config) BlockTarget() string {
	if v := os.Getenv("LAB_BLOCK_TARGET"); v != "" {
		return v
	}
	return "/dev/nvme0n1"
}

// DefaultValidationOutput returns a sensible report path beside lab config.
func DefaultValidationOutput(configPath string) string {
	base := strings.TrimSuffix(filepath.Base(configPath), filepath.Ext(configPath))
	if base == "" || base == "." {
		base = "lab"
	}
	return base + "-validation.json"
}
