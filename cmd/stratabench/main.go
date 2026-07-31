package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/pratham-vishk/stratabench/internal/compare"
	"github.com/pratham-vishk/stratabench/internal/export"
	"github.com/pratham-vishk/stratabench/internal/orchestrator"
	"github.com/pratham-vishk/stratabench/internal/paths"
	"github.com/pratham-vishk/stratabench/internal/planner"
	"github.com/pratham-vishk/stratabench/internal/profile"
	"github.com/pratham-vishk/stratabench/internal/remote"
	"github.com/pratham-vishk/stratabench/internal/report"
	"github.com/pratham-vishk/stratabench/internal/schema"
)

var (
	profileName  string
	target       string
	clientsCSV   string
	mock         bool
	runID        string
	runIDB       string
	cacheBytes   int64
	skipValidate bool
)

func main() {
	root := &cobra.Command{
		Use:   "stratabench",
		Short: "StrataBench — agentic, honest storage benchmarking",
	}

	root.AddCommand(
		validateCmd(),
		runCmd(),
		reportCmd(),
		exportCmd(),
		runsCmd(),
		compareCmd(),
		planCmd(),
		profilesCmd(),
	)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func loadProfile() (*profile.Profile, error) {
	if profileName == "" {
		return nil, fmt.Errorf("--profile is required")
	}
	return profile.LoadByName(paths.ProfilesDir(), profileName)
}

func newService() (*orchestrator.Service, error) {
	return orchestrator.NewService(paths.DataDir())
}

func validateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate a workload profile before running",
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := loadProfile()
			if err != nil {
				return err
			}
			svc, err := newService()
			if err != nil {
				return err
			}
			defer svc.Close()

			res := svc.Validate(orchestrator.RunOptions{Profile: p, CacheBytes: cacheBytes})
			printValidation(p, res)
			if !res.Passed {
				return fmt.Errorf("validation failed")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&profileName, "profile", "", "Workload profile name")
	cmd.Flags().Int64Var(&cacheBytes, "cache-bytes", 0, "Assumed array cache size in bytes")
	_ = cmd.MarkFlagRequired("profile")
	return cmd
}

func runCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run a benchmark profile (local or distributed via agents)",
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := loadProfile()
			if err != nil {
				return err
			}
			svc, err := newService()
			if err != nil {
				return err
			}
			defer svc.Close()

			clients := remote.ParseHosts(clientsCSV)
			run, err := svc.Run(cmd.Context(), orchestrator.RunOptions{
				Profile:      p,
				Target:       target,
				Clients:      clients,
				Mock:         mock,
				SkipValidate: skipValidate,
				CacheBytes:   cacheBytes,
				DataDir:      paths.DataDir(),
			})
			if err != nil {
				return err
			}

			out := filepath.Join(paths.ReportsDir(), run.RunID+".html")
			if err := report.WriteHTML(run, out); err != nil {
				return err
			}
			_ = export.WriteJSON(run, filepath.Join(paths.ReportsDir(), run.RunID+".json"))

			printRunSummary(run, len(clients))
			return nil
		},
	}
	cmd.Flags().StringVar(&profileName, "profile", "", "Workload profile name")
	cmd.Flags().StringVar(&target, "target", "", "Block device, file path, or S3 endpoint")
	cmd.Flags().StringVar(&clientsCSV, "clients", "", "Comma-separated agent URLs (host:7777)")
	cmd.Flags().BoolVar(&mock, "mock", false, "Use mock engine (no real I/O)")
	cmd.Flags().BoolVar(&skipValidate, "skip-validate", false, "Skip pre-run validation")
	cmd.Flags().Int64Var(&cacheBytes, "cache-bytes", 0, "Assumed cache bytes for validation")
	_ = cmd.MarkFlagRequired("profile")
	return cmd
}

func reportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "report",
		Short: "Generate HTML report for a completed run",
		RunE: func(cmd *cobra.Command, args []string) error {
			if runID == "" {
				return fmt.Errorf("--run-id is required")
			}
			svc, err := newService()
			if err != nil {
				return err
			}
			defer svc.Close()
			run, err := svc.Store.Get(runID)
			if err != nil {
				return err
			}
			return report.WriteHTML(run, filepath.Join(paths.ReportsDir(), run.RunID+".html"))
		},
	}
	cmd.Flags().StringVar(&runID, "run-id", "", "Run UUID")
	return cmd
}

func exportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export run result as JSON",
		RunE: func(cmd *cobra.Command, args []string) error {
			if runID == "" {
				return fmt.Errorf("--run-id is required")
			}
			svc, err := newService()
			if err != nil {
				return err
			}
			defer svc.Close()
			run, err := svc.Store.Get(runID)
			if err != nil {
				return err
			}
			return export.WriteJSON(run, filepath.Join(paths.ReportsDir(), run.RunID+".json"))
		},
	}
	cmd.Flags().StringVar(&runID, "run-id", "", "Run UUID")
	return cmd
}

func runsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "runs",
		Short: "List recent benchmark runs",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := newService()
			if err != nil {
				return err
			}
			defer svc.Close()
			runs, err := svc.Store.List(20)
			if err != nil {
				return err
			}
			fmt.Printf("%-38s %-22s %-10s %8s %10s\n", "RUN_ID", "PROFILE", "ENGINE", "MOCK", "IOPS")
			for _, r := range runs {
				mockTag := "no"
				if r.Mock {
					mockTag = "yes"
				}
				fmt.Printf("%-38s %-22s %-10s %8s %10.0f\n", r.RunID, r.Profile, r.Engine, mockTag, r.Results.IOPS)
			}
			return nil
		},
	}
}

func compareCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "compare",
		Short: "Compare two completed runs",
		RunE: func(cmd *cobra.Command, args []string) error {
			if runID == "" || runIDB == "" {
				return fmt.Errorf("--run-id and --run-id-b are required")
			}
			svc, err := newService()
			if err != nil {
				return err
			}
			defer svc.Close()
			a, err := svc.Store.Get(runID)
			if err != nil {
				return err
			}
			b, err := svc.Store.Get(runIDB)
			if err != nil {
				return err
			}
			compare.Print(a, b)
			return nil
		},
	}
	cmd.Flags().StringVar(&runID, "run-id", "", "First run UUID")
	cmd.Flags().StringVar(&runIDB, "run-id-b", "", "Second run UUID")
	return cmd
}

func planCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "plan [intent]",
		Short: "Suggest a profile from natural language (keyword planner v0.1)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			text := strings.Join(args, " ")
			profiles, err := profile.List(paths.ProfilesDir())
			if err != nil {
				return err
			}
			name := planner.SuggestProfile(text, profiles)
			p, err := profile.LoadByName(paths.ProfilesDir(), name)
			if err != nil {
				return err
			}
			fmt.Printf("Suggested profile: %s\n", p.Name)
			fmt.Printf("  %s\n", p.Description)
			fmt.Printf("  engine=%s layer=%s load=%s\n", p.Engine, p.Layer, p.Load)
			fmt.Printf("\nNext:\n  stratabench validate --profile %s\n", p.Name)
			fmt.Printf("  stratabench run --profile %s --target <target> --mock\n", p.Name)
			return nil
		},
	}
}

func profilesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "profiles",
		Short: "List built-in workload profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			profiles, err := profile.List(paths.ProfilesDir())
			if err != nil {
				return err
			}
			for _, p := range profiles {
				fmt.Printf("%-22s %-10s %-12s %s\n", p.Name, p.Layer, p.Engine, p.Description)
			}
			return nil
		},
	}
}

func printValidation(p *profile.Profile, res schema.ValidationResult) {
	fmt.Printf("Profile: %s (%s)\n", p.Name, p.Description)
	fmt.Printf("Engine:  %s | Layer: %s | Load: %s\n", p.Engine, p.Layer, p.Load)
	if res.Passed {
		fmt.Println("Validation: PASSED")
	} else {
		fmt.Println("Validation: FAILED")
	}
	for _, e := range res.Errors {
		fmt.Printf("  ERROR: %s\n", e)
	}
	for _, w := range res.Warnings {
		fmt.Printf("  WARN [%s]: %s\n", w.Rule, w.Message)
	}
}

func printRunSummary(run *schema.RunResult, clientCount int) {
	fmt.Printf("Run completed: %s\n", run.RunID)
	if clientCount > 0 {
		fmt.Printf("  Clients: %d (aggregated)\n", clientCount)
	}
	fmt.Printf("  IOPS: %.0f  Throughput: %.2f MB/s  p99: %.1f µs\n",
		run.Results.IOPS, run.Results.ThroughputMBps, run.Results.LatencyUS.P99)
	if run.Mock {
		fmt.Println("  (mock mode — not real storage I/O)")
	}
}
