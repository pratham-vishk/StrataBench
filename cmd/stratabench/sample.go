package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/pratham-vishk/stratabench/internal/analyst"
	"github.com/pratham-vishk/stratabench/internal/compare"
	"github.com/pratham-vishk/stratabench/internal/orchestrator"
	"github.com/pratham-vishk/stratabench/internal/paths"
	"github.com/pratham-vishk/stratabench/internal/profile"
	"github.com/pratham-vishk/stratabench/internal/report"
	"github.com/pratham-vishk/stratabench/internal/topology"
)

func sampleCmd() *cobra.Command {
	var outputDir string
	var useMock bool
	cmd := &cobra.Command{
		Use:   "sample",
		Short: "Run a normal benchmark demo and save sample HTML",
		Long: `Runs a standard stratabench run (same as 'run') and writes report artifacts.

Default is mock mode so it works on any machine without NVMe/fio.

  stratabench sample
  stratabench sample --open-report
  stratabench sample --profile nvme-random-oltp --target /dev/nvme0n1   # real hardware (Linux)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = initDirs()

			if profileName == "" {
				profileName = "nvme-random-oltp"
			}
			if target == "" {
				target = "/dev/null"
			}
			p, err := profile.LoadByName(paths.ProfilesDir(), profileName)
			if err != nil {
				return err
			}

			svc, err := newService()
			if err != nil {
				return err
			}
			defer svc.Close()

			targets := topology.ParseCSV(targetsCSV)
			run, err := svc.Run(cmd.Context(), orchestrator.RunOptions{
				Profile:       p,
				Target:        target,
				Targets:       targets,
				Mock:          useMock,
				SkipValidate:  useMock || skipValidate,
				CheckHardware: !useMock && checkHardware,
				CacheBytes:    cacheBytes,
				DataDir:       paths.DataDir(),
				GitRepo:       compare.DefaultRepo(""),
			})
			if err != nil {
				return err
			}

			insights := analyst.Analyze(run, nil)
			summary := fmt.Sprintf(
				"Sample benchmark — %s on %s (%s engine, %ds). Use the same flow with: stratabench run --profile %s --target <device>",
				run.Profile, target, run.Engine, run.Workload.DurationSec, run.Profile,
			)
			_ = os.MkdirAll(outputDir, 0o755)
			if err := report.WriteHTMLOnly(run, report.OptionsFromAnalysis(insights, summary, nil),
				filepath.Join(outputDir, "benchmark-sample.html")); err != nil {
				return err
			}

			printRunSummary(run)
			fmt.Println()
			fmt.Println("Sample benchmark report (HTML only):")
			fmt.Println("  ", filepath.Join(outputDir, "benchmark-sample.html"))
			fmt.Println()
			fmt.Println("For base + candidate compare:")
			fmt.Println("  stratabench sample-compare")

			if openReport {
				_ = report.OpenInBrowser(filepath.Join(outputDir, "benchmark-sample.html"))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&profileName, "profile", "nvme-random-oltp", "Workload profile")
	cmd.Flags().StringVar(&target, "target", "/dev/null", "Block device or path")
	cmd.Flags().StringVar(&targetsCSV, "targets", "", "Comma-separated targets")
	cmd.Flags().BoolVar(&useMock, "mock", true, "Use mock engine (recommended for sample)")
	cmd.Flags().BoolVar(&skipValidate, "skip-validate", false, "Skip pre-run validation")
	cmd.Flags().BoolVar(&checkHardware, "check-hardware", false, "Validate hardware before run")
	cmd.Flags().Int64Var(&cacheBytes, "cache-bytes", 10*1024*1024*1024, "Cache bytes for validation")
	cmd.Flags().StringVar(&outputDir, "output", filepath.Join("examples", "sample-report", "output"), "Directory for benchmark-sample.* files")
	cmd.Flags().BoolVar(&openReport, "open-report", false, "Open HTML report in browser")
	return cmd
}

func initDirs() error {
	for _, d := range []string{paths.DataDir(), paths.ReportsDir(), filepath.Join(paths.DataDir(), "work")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func reportOpen(path string) error {
	return report.OpenInBrowser(path)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
