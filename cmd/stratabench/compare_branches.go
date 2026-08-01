package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/pratham-vishk/stratabench/internal/compare"
	"github.com/pratham-vishk/stratabench/internal/git"
	"github.com/pratham-vishk/stratabench/internal/orchestrator"
	"github.com/pratham-vishk/stratabench/internal/paths"
	"github.com/pratham-vishk/stratabench/internal/report"
	"github.com/pratham-vishk/stratabench/internal/topology"
)

func compareBranchesCmd() *cobra.Command {
	var (
		baseBranch   string
		headBranch   string
		repoDir      string
		buildCmd     string
		allowDirty   bool
		skipBuild    bool
		failOnReg    bool
		compareReport bool
	)
	cmd := &cobra.Command{
		Use:   "branches",
		Short: "Benchmark two git branches and analyze performance impact",
		Long: `Checkout each branch, optionally build, run the same profile, and produce a comparison report.

Example:
  stratabench compare branches --base main --head feature/storage-opt \
    --profile nvme-random-oltp --target /dev/nvme0n1 --mock

  stratabench compare branches --repo /path/to/your/driver --base main --head opt \
    --build-cmd "make -j8" --profile nvme-random-oltp --target /dev/nvme0n1`,
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := loadProfile()
			if err != nil {
				return err
			}
			repo := compare.DefaultRepo(repoDir)
			if baseBranch == "main" && !git.BranchExists(repo, "main") {
				baseBranch = git.DefaultBranch(repo)
			}
			svc, err := newService()
			if err != nil {
				return err
			}
			defer svc.Close()

			targets := topology.ParseCSV(targetsCSV)
			if target != "" && len(targets) == 0 {
				targets = []string{target}
			}

			result, err := compare.CompareBranches(cmd.Context(), svc, compare.BranchOptions{
				RepoDir:    repo,
				BaseBranch: baseBranch,
				HeadBranch: headBranch,
				BuildCmd:   buildCmd,
				AllowDirty: allowDirty,
				SkipBuild:  skipBuild,
				Run: orchestrator.RunOptions{
					Profile:       p,
					Target:        target,
					Targets:       targets,
					Mock:          mock,
					SkipValidate:  skipValidate,
					CheckHardware: checkHardware,
					CacheBytes:    cacheBytes,
					DataDir:       paths.DataDir(),
					GitRepo:       repo,
					BuildCmd:      buildCmd,
				},
			})
			if err != nil {
				return err
			}

			compare.Print(result.BaseRun, result.HeadRun)
			htmlPath, err := report.WriteCompareArtifacts(result.BaseRun, result.HeadRun, result.Diff)
			if err != nil {
				return err
			}
			fmt.Printf("\nCompare report: %s\n", htmlPath)
			fmt.Printf("Base run:  %s\n", result.BaseRun.RunID)
			fmt.Printf("Head run: %s\n", result.HeadRun.RunID)

			if failOnReg && result.Diff.Regressed {
				return fmt.Errorf("head branch regressed vs base — see compare report")
			}
			if compareReport || openReport {
				_ = report.OpenInBrowser(htmlPath)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&baseBranch, "base", "main", "Baseline git branch")
	cmd.Flags().StringVar(&headBranch, "head", "", "Feature git branch to compare (required)")
	cmd.Flags().StringVar(&repoDir, "repo", "", "Git repo to build/benchmark (default: cwd or STRATABENCH_GIT_REPO)")
	cmd.Flags().StringVar(&buildCmd, "build-cmd", "", "Command to run after each checkout (e.g. make -j8)")
	cmd.Flags().BoolVar(&allowDirty, "allow-dirty", false, "Allow dirty working tree")
	cmd.Flags().BoolVar(&skipBuild, "skip-build", false, "Skip build step")
	cmd.Flags().BoolVar(&failOnReg, "fail-on-regression", false, "Exit non-zero if head regressed")
	cmd.Flags().BoolVar(&compareReport, "open-report", false, "Open compare HTML in browser")
	cmd.Flags().StringVar(&profileName, "profile", "", "Workload profile name")
	cmd.Flags().StringVar(&target, "target", "/dev/null", "Block device or path")
	cmd.Flags().StringVar(&targetsCSV, "targets", "", "Comma-separated targets")
	cmd.Flags().BoolVar(&mock, "mock", false, "Use mock engine")
	cmd.Flags().BoolVar(&skipValidate, "skip-validate", false, "Skip validation")
	cmd.Flags().BoolVar(&checkHardware, "check-hardware", false, "Validate hardware (off for mock)")
	cmd.Flags().Int64Var(&cacheBytes, "cache-bytes", 0, "Cache bytes for validation")
	_ = cmd.MarkFlagRequired("head")
	return cmd
}

func initCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize StrataBench data directories (production setup)",
		RunE: func(cmd *cobra.Command, args []string) error {
			dirs := []string{
				paths.DataDir(),
				paths.ReportsDir(),
				filepath.Join(paths.DataDir(), "work"),
			}
			for _, d := range dirs {
				if err := os.MkdirAll(d, 0o755); err != nil {
					return err
				}
				fmt.Println("created:", d)
			}
			fmt.Println("\nStrataBench is ready. Try:")
			fmt.Println("  stratabench run --profile nvme-random-oltp --target /dev/nvme0n1 --mock")
			fmt.Println("  stratabench compare branches --base main --head feature/x --profile nvme-random-oltp --mock")
			return nil
		},
	}
	return cmd
}
