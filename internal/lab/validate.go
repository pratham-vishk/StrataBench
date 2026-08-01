package lab

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/pratham-vishk/stratabench/internal/profile"
	"github.com/pratham-vishk/stratabench/internal/validator"
)

// ValidationItem is one row in the Dell lab hardware sign-off matrix.
type ValidationItem struct {
	Section string `json:"section"`
	Profile string `json:"profile"`
	Engine  string `json:"engine"`
	Target  string `json:"target"`
	Command string `json:"command"`
}

// ValidateReport summarizes lab validation readiness.
type ValidateReport struct {
	Check       *CheckReport     `json:"check,omitempty"`
	Items       []ValidationItem `json:"items"`
	SmokePassed int              `json:"smoke_passed"`
	SmokeFailed int              `json:"smoke_failed"`
	SmokeErrors []string         `json:"smoke_errors,omitempty"`
}

// ValidationMatrix returns the Dell lab sign-off checklist with targets resolved from lab config.
func ValidationMatrix(cfg Config) []ValidationItem {
	block := cfg.BlockTarget()
	s3 := cfg.ServerCSV()
	if s3 == "" {
		s3 = "10.0.1.10:9000"
	}
	client := "10.0.1.1:7777"
	if len(cfg.Clients) > 0 {
		client = fmt.Sprintf("%s:%d", cfg.Clients[0].Host, cfg.AgentPort)
	}
	vm := "root@10.0.1.20:/dev/vdb"
	if len(cfg.Clients) > 0 {
		vm = fmt.Sprintf("root@%s:/dev/vdb", cfg.Clients[0].Host)
	}

	rows := []ValidationItem{
		{Section: "block-fio", Profile: "hdd-sequential-read", Engine: "fio", Target: block, Command: fmt.Sprintf("stratabench run --profile hdd-sequential-read --target %s", block)},
		{Section: "block-fio", Profile: "ssd-random-4k", Engine: "fio", Target: block, Command: fmt.Sprintf("stratabench run --profile ssd-random-4k --target %s", block)},
		{Section: "block-fio", Profile: "nvme-random-oltp", Engine: "fio", Target: block, Command: fmt.Sprintf("stratabench run --profile nvme-random-oltp --target %s", block)},
		{Section: "block-native", Profile: "block-native-oltp", Engine: "stratabench", Target: block, Command: fmt.Sprintf("stratabench run --profile block-native-oltp --target %s", block)},
		{Section: "block-native", Profile: "block-native-io_uring", Engine: "stratabench", Target: block, Command: fmt.Sprintf("stratabench run --profile block-native-io_uring --target %s", block)},
		{Section: "object-warp", Profile: "s3-put-throughput", Engine: "warp", Target: s3, Command: fmt.Sprintf("stratabench run --profile s3-put-throughput --target %s", strings.Split(s3, ",")[0])},
		{Section: "object-warp", Profile: "s3-cluster-rdma", Engine: "warp", Target: s3, Command: fmt.Sprintf("stratabench run --profile s3-cluster-rdma --target %s --clients %s", strings.Split(s3, ",")[0], client)},
		{Section: "object-gosbench", Profile: "s3-gosbench-write", Engine: "gosbench", Target: strings.Split(s3, ",")[0], Command: fmt.Sprintf("stratabench run --profile s3-gosbench-write --target %s", strings.Split(s3, ",")[0])},
		{Section: "vm-block", Profile: "vm-nvme-oltp", Engine: "fio", Target: vm, Command: fmt.Sprintf("stratabench run --profile vm-nvme-oltp --target %s", vm)},
		{Section: "monitoring", Profile: "nvme-random-oltp", Engine: "mock", Target: "/dev/null", Command: "stratabench run --profile nvme-random-oltp --target /dev/null --mock --async --watch"},
	}
	return rows
}

// Validate runs lab check and optional smoke validation.
func Validate(ctx context.Context, cfg Config, profilesDir string, smoke bool) (*ValidateReport, error) {
	check, err := Check(ctx, cfg)
	if err != nil {
		return nil, err
	}
	rep := &ValidateReport{
		Check: check,
		Items: ValidationMatrix(cfg),
	}
	if !smoke {
		return rep, nil
	}
	smokeProfiles := []string{"nvme-random-oltp", "s3-put-throughput", "block-native-oltp"}
	for _, name := range smokeProfiles {
		p, err := profile.LoadByName(profilesDir, name)
		if err != nil {
			rep.SmokeFailed++
			rep.SmokeErrors = append(rep.SmokeErrors, err.Error())
			continue
		}
		v := validator.Validate(p, validator.Options{Mock: true, Target: "/dev/null"})
		if v.Passed {
			rep.SmokePassed++
		} else {
			rep.SmokeFailed++
			rep.SmokeErrors = append(rep.SmokeErrors, fmt.Sprintf("%s: %v", name, v.Errors))
		}
	}
	return rep, nil
}

func PrintValidationReport(rep *ValidateReport) {
	if rep.Check != nil {
		PrintCheckReport(rep.Check, Config{})
		fmt.Println()
	}
	fmt.Println("Dell lab validation matrix:")
	cur := ""
	for _, it := range rep.Items {
		if it.Section != cur {
			cur = it.Section
			fmt.Printf("\n[%s]\n", cur)
		}
		fmt.Printf("  %-28s %-12s %s\n", it.Profile, it.Engine, it.Command)
	}
	fmt.Println("\nSign-off (mark in docs/DELL-LAB-VALIDATION.md):")
	fmt.Println("  [ ] All engines: fio, warp, gosbench, stratabench, sbk, mock")
	fmt.Println("  [ ] Live monitoring: async + watch + SSE during mock run")
	fmt.Println("  [ ] Native engine: block-native-oltp + block-native-io_uring on NVMe")
	fmt.Println("  [ ] Baseline regression on nvme-random-oltp")
	if rep.SmokePassed+rep.SmokeFailed > 0 {
		fmt.Printf("\nSmoke validation: %d passed, %d failed\n", rep.SmokePassed, rep.SmokeFailed)
		for _, e := range rep.SmokeErrors {
			fmt.Printf("  ! %s\n", e)
		}
	}
	if rep.Check != nil && rep.Check.Ready {
		fmt.Println("\nlab-validate: infrastructure READY — execute matrix on hardware")
	} else {
		fmt.Println("\nlab-validate: infrastructure NOT READY — fix lab check first")
	}
}

// BlockTarget returns configured block device or default.
func (c Config) BlockTarget() string {
	if v := os.Getenv("LAB_BLOCK_TARGET"); v != "" {
		return v
	}
	return "/dev/nvme0n1"
}
