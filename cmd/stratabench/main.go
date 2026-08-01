package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/pratham-vishk/stratabench/internal/agentloop"
	"github.com/pratham-vishk/stratabench/internal/analyst"
	"github.com/pratham-vishk/stratabench/internal/baseline"
	"github.com/pratham-vishk/stratabench/internal/compare"
	"github.com/pratham-vishk/stratabench/internal/crosslayer"
	"github.com/pratham-vishk/stratabench/internal/discovery"
	"github.com/pratham-vishk/stratabench/internal/export"
	"github.com/pratham-vishk/stratabench/internal/importsbk"
	"github.com/pratham-vishk/stratabench/internal/inventory"
	"github.com/pratham-vishk/stratabench/internal/manifest"
	"github.com/pratham-vishk/stratabench/internal/orchestrator"
	"github.com/pratham-vishk/stratabench/internal/paths"
	"github.com/pratham-vishk/stratabench/internal/planner"
	"github.com/pratham-vishk/stratabench/internal/profile"
	"github.com/pratham-vishk/stratabench/internal/remote"
	"github.com/pratham-vishk/stratabench/internal/report"
	"github.com/pratham-vishk/stratabench/internal/smarthistory"
	"github.com/pratham-vishk/stratabench/internal/schema"
	"github.com/pratham-vishk/stratabench/internal/topology"
	"github.com/pratham-vishk/stratabench/internal/version"
)

var (
	profileName  string
	target       string
	targetsCSV   string
	clientsCSV   string
	topologyMode string
	mock         bool
	runID        string
	runIDB       string
	cacheBytes   int64
	skipValidate   bool
	checkBaseline  bool
	checkHardware  bool
	profilesCSV   string
	useOllama     bool
	ollamaURL     string
	ollamaModel   string
)

func main() {
	root := &cobra.Command{
		Use:   "stratabench",
		Short: "StrataBench — agentic, honest storage benchmarking",
	}

	root.AddCommand(
		validateCmd(),
		applyCmd(),
		runCmd(),
		reportCmd(),
		exportCmd(),
		runsCmd(),
		compareCmd(),
		crossLayerCmd(),
		importCmd(),
		baselineCmd(),
		inventoryCmd(),
		smartCmd(),
		analyzeCmd(),
		agentCmd(),
		planCmd(),
		profilesCmd(),
		versionCmd(),
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

			res := svc.Validate(orchestrator.RunOptions{
				Profile:       p,
				CacheBytes:    cacheBytes,
				CheckHardware: checkHardware,
				Target:        target,
				Mock:          mock,
			})
			printValidation(p, res)
			if !res.Passed {
				return fmt.Errorf("validation failed")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&profileName, "profile", "", "Workload profile name")
	cmd.Flags().StringVar(&target, "target", "", "Block device, path, or endpoint for hardware checks")
	cmd.Flags().Int64Var(&cacheBytes, "cache-bytes", 0, "Assumed array cache size in bytes")
	cmd.Flags().BoolVar(&mock, "mock", false, "Skip hardware checks (mock mode)")
	cmd.Flags().BoolVar(&checkHardware, "check-hardware", true, "Validate host tools and devices for profile")
	_ = cmd.MarkFlagRequired("profile")
	return cmd
}

func applyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply a declarative benchmark manifest (YAML)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("manifest file path required")
			}
			b, err := manifest.Load(args[0])
			if err != nil {
				return err
			}
			svc, err := newService()
			if err != nil {
				return err
			}
			defer svc.Close()
			result, err := manifest.Apply(cmd.Context(), svc, b)
			if err != nil {
				return err
			}
			fmt.Printf("benchmark applied: run_id=%s profile=%s status=%s\n", result.RunID, result.Profile, result.Status)
			return nil
		},
	}
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
			targets := topology.ParseCSV(targetsCSV)
			if target != "" && len(targets) == 0 {
				targets = []string{target}
			}
			run, err := svc.Run(cmd.Context(), orchestrator.RunOptions{
				Profile:       p,
				Target:        target,
				Targets:       targets,
				Clients:       clients,
				Topology:      topologyMode,
				Mock:          mock,
				SkipValidate:  skipValidate,
				CheckBaseline: checkBaseline,
				CheckHardware: checkHardware,
				CacheBytes:    cacheBytes,
				DataDir:       paths.DataDir(),
			})
			if err != nil {
				return err
			}

			out := filepath.Join(paths.ReportsDir(), run.RunID+".html")
			if err := report.WriteHTML(run, out); err != nil {
				return err
			}
			_ = export.WriteJSON(run, filepath.Join(paths.ReportsDir(), run.RunID+".json"))

			printRunSummary(run)
			if checkBaseline {
				baseline.PrintAlerts(svc.CheckRegression(run))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&profileName, "profile", "", "Workload profile name")
	cmd.Flags().StringVar(&target, "target", "", "Single block device, path, or S3 endpoint")
	cmd.Flags().StringVar(&targetsCSV, "targets", "", "Comma-separated server targets (multi-server)")
	cmd.Flags().StringVar(&clientsCSV, "clients", "", "Comma-separated agent URLs (host:7777)")
	cmd.Flags().StringVar(&topologyMode, "topology", "auto", "Topology: auto, single, pool, sweep, shard, matrix")
	cmd.Flags().BoolVar(&mock, "mock", false, "Use mock engine (no real I/O)")
	cmd.Flags().BoolVar(&skipValidate, "skip-validate", false, "Skip pre-run validation")
	cmd.Flags().BoolVar(&checkBaseline, "check-baseline", false, "Compare results against stored baseline after run")
	cmd.Flags().BoolVar(&checkHardware, "check-hardware", true, "Validate host tools and devices before run")
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

func crossLayerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cross-layer",
		Short: "Run multiple profiles and analyze cross-layer bottlenecks",
		RunE: func(cmd *cobra.Command, args []string) error {
			names := crosslayer.ParseProfilesCSV(profilesCSV)
			if len(names) < 2 {
				return fmt.Errorf("--profiles requires at least 2 comma-separated profile names")
			}
			svc, err := newService()
			if err != nil {
				return err
			}
			defer svc.Close()

			var runs []*schema.RunResult
			for _, name := range names {
				profileName = name
				p, err := loadProfile()
				if err != nil {
					return err
				}
				run, err := svc.Run(cmd.Context(), orchestrator.RunOptions{
					Profile:      p,
					Target:       target,
					Mock:         mock,
					SkipValidate: skipValidate,
					CacheBytes:   cacheBytes,
					DataDir:      paths.DataDir(),
				})
				if err != nil {
					return fmt.Errorf("%s: %w", name, err)
				}
				_ = report.WriteHTML(run, filepath.Join(paths.ReportsDir(), run.RunID+".html"))
				runs = append(runs, run)
				fmt.Printf("  %s (%s): IOPS=%.0f p99=%.0fµs\n", name, p.Layer, run.Results.IOPS, run.Results.LatencyUS.P99)
			}
			crosslayer.PrintInsights(crosslayer.Analyze(runs))
			return nil
		},
	}
	cmd.Flags().StringVar(&profilesCSV, "profiles", "", "Comma-separated profiles (e.g. nvme-random-oltp,s3-put-throughput)")
	cmd.Flags().StringVar(&target, "target", "", "Target path or endpoint")
	cmd.Flags().BoolVar(&mock, "mock", true, "Use mock engines")
	cmd.Flags().BoolVar(&skipValidate, "skip-validate", true, "Skip validation (default true for multi-profile runs)")
	_ = cmd.MarkFlagRequired("profiles")
	return cmd
}

func importCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import external benchmark results",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "sbk [csv-file]",
		Short: "Import SBK CSV results into StrataBench store",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := newService()
			if err != nil {
				return err
			}
			defer svc.Close()
			runs, err := importsbk.ParseCSV(args[0])
			if err != nil {
				return err
			}
			for _, run := range runs {
				if err := svc.Store.Save(run); err != nil {
					return err
				}
				fmt.Printf("imported run %s profile=%s IOPS=%.0f\n", run.RunID, run.Profile, run.Results.IOPS)
			}
			return nil
		},
	})
	return cmd
}

func baselineCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "baseline",
		Short: "Manage regression baselines per profile+target",
	}
	cmd.AddCommand(
		&cobra.Command{
			Use:   "set",
			Short: "Set baseline from an existing run",
			RunE: func(cmd *cobra.Command, args []string) error {
				if runID == "" {
					return fmt.Errorf("--run-id is required")
				}
				svc, err := newService()
				if err != nil {
					return err
				}
				defer svc.Close()
				rec, err := svc.SetBaselineFromRun(runID)
				if err != nil {
					return err
				}
				fmt.Printf("baseline set: profile=%s target=%s run_id=%s\n", rec.Profile, rec.TargetKey, rec.RunID)
				return nil
			},
		},
		&cobra.Command{
			Use:   "show",
			Short: "List stored baselines",
			RunE: func(cmd *cobra.Command, args []string) error {
				svc, err := newService()
				if err != nil {
					return err
				}
				defer svc.Close()
				recs, err := svc.Store.ListBaselines()
				if err != nil {
					return err
				}
				if len(recs) == 0 {
					fmt.Println("No baselines stored.")
					return nil
				}
				fmt.Printf("%-22s %-24s %-38s %s\n", "PROFILE", "TARGET", "RUN_ID", "SET_AT")
				for _, r := range recs {
					fmt.Printf("%-22s %-24s %-38s %s\n", r.Profile, r.TargetKey, r.RunID, r.SetAt)
				}
				return nil
			},
		},
		&cobra.Command{
			Use:   "check",
			Short: "Check a run against its baseline",
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
				baseline.PrintAlerts(svc.CheckRegression(run))
				return nil
			},
		},
	)
	cmd.PersistentFlags().StringVar(&runID, "run-id", "", "Run ID for set/check")
	return cmd
}

func planCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plan [intent]",
		Short: "Suggest a profile from natural language (keyword or Ollama)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			text := strings.Join(args, " ")
			profiles, err := profile.List(paths.ProfilesDir())
			if err != nil {
				return err
			}
			result := planner.Plan(cmd.Context(), planner.PlanOptions{
				Intent:      text,
				Profiles:    profiles,
				Hardware:    discovery.Snapshot(),
				UseOllama:   useOllama,
				OllamaURL:   ollamaURL,
				OllamaModel: ollamaModel,
			})
			p, err := profile.LoadByName(paths.ProfilesDir(), result.Profile)
			if err != nil {
				return err
			}
			fmt.Printf("Suggested profile: %s (source: %s)\n", p.Name, result.Source)
			fmt.Printf("  %s\n", result.Rationale)
			fmt.Printf("  engine=%s layer=%s load=%s\n", p.Engine, p.Layer, p.Load)
			fmt.Printf("\nNext:\n  stratabench validate --profile %s\n", p.Name)
			fmt.Printf("  stratabench run --profile %s --target <target> --mock\n", p.Name)
			fmt.Printf("  stratabench agent %q --target <target> --mock\n", text)
			return nil
		},
	}
	cmd.Flags().BoolVar(&useOllama, "ollama", false, "Use Ollama LLM planner (falls back to keywords)")
	cmd.Flags().StringVar(&ollamaURL, "ollama-url", "", "Ollama API URL (default http://localhost:11434)")
	cmd.Flags().StringVar(&ollamaModel, "model", "", "Ollama model name (default llama3.2)")
	return cmd
}

func inventoryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inventory",
		Short: "Hardware inventory database",
	}
	cmd.AddCommand(
		&cobra.Command{
			Use:   "collect",
			Short: "Collect and store current host hardware snapshot",
			RunE: func(cmd *cobra.Command, args []string) error {
				svc, err := newService()
				if err != nil {
					return err
				}
				defer svc.Close()
				snap := inventory.Collect()
				if err := inventory.Save(svc.Store, snap); err != nil {
					return err
				}
				fmt.Printf("inventory saved for host %s (%d NVMe, %d block devices)\n",
					inventory.HostID(snap), len(snap.NVMe), len(snap.BlockDevices))
				return nil
			},
		},
		&cobra.Command{
			Use:   "list",
			Short: "List stored hardware inventory",
			RunE: func(cmd *cobra.Command, args []string) error {
				svc, err := newService()
				if err != nil {
					return err
				}
				defer svc.Close()
				recs, err := inventory.List(svc.Store)
				if err != nil {
					return err
				}
				inventory.Print(recs)
				return nil
			},
		},
	)
	return cmd
}

func smartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "smart",
		Short: "SMART disk health history",
	}
	cmd.AddCommand(
		&cobra.Command{
			Use:   "collect",
			Short: "Collect SMART readings via smartctl (Linux)",
			RunE: func(cmd *cobra.Command, args []string) error {
				svc, err := newService()
				if err != nil {
					return err
				}
				defer svc.Close()
				n, err := smarthistory.CollectAndSave(svc.Store)
				if err != nil {
					return err
				}
				fmt.Printf("SMART: saved %d device readings\n", n)
				return nil
			},
		},
		&cobra.Command{
			Use:   "list",
			Short: "List SMART history",
			RunE: func(cmd *cobra.Command, args []string) error {
				svc, err := newService()
				if err != nil {
					return err
				}
				defer svc.Close()
				recs, err := smarthistory.List(svc.Store, 50)
				if err != nil {
					return err
				}
				smarthistory.Print(recs)
				return nil
			},
		},
	)
	return cmd
}

func analyzeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "analyze",
		Short: "Run analyst on a completed benchmark",
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
			regression := svc.CheckRegression(run)
			insights := analyst.Analyze(run, regression)
			analyst.PrintInsights(insights)
			fmt.Println(analyst.SummaryText(run, insights))
			return nil
		},
	}
	cmd.Flags().StringVar(&runID, "run-id", "", "Run ID to analyze")
	return cmd
}

func agentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent [intent]",
		Short: "Agentic loop: plan → validate → run → analyze → report",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targets := topology.MergeTargets(target, topology.ParseCSV(targetsCSV))
			if len(targets) == 0 {
				return fmt.Errorf("--target or --targets is required")
			}
			_, err := agentloop.Run(cmd.Context(), agentloop.Options{
				Intent:        strings.Join(args, " "),
				Target:        targets[0],
				Targets:       targets,
				Clients:       remote.ParseHosts(clientsCSV),
				Topology:      topologyMode,
				Mock:          mock,
				SkipValidate:  skipValidate,
				CheckBaseline: checkBaseline,
				CheckHardware: checkHardware,
				CacheBytes:    cacheBytes,
				UseOllama:     useOllama,
				OllamaURL:     ollamaURL,
				OllamaModel:   ollamaModel,
			})
			return err
		},
	}
	cmd.Flags().StringVar(&target, "target", "", "Block device, file path, or S3 endpoint")
	cmd.Flags().StringVar(&targetsCSV, "targets", "", "Comma-separated server targets")
	cmd.Flags().StringVar(&clientsCSV, "clients", "", "Comma-separated agent URLs (host:7777)")
	cmd.Flags().StringVar(&topologyMode, "topology", "auto", "Topology: auto, single, pool, sweep, shard, matrix")
	cmd.Flags().BoolVar(&mock, "mock", true, "Use mock engine (default true for agent)")
	cmd.Flags().BoolVar(&skipValidate, "skip-validate", false, "Skip pre-run validation")
	cmd.Flags().BoolVar(&checkBaseline, "check-baseline", true, "Compare against stored baseline")
	cmd.Flags().BoolVar(&checkHardware, "check-hardware", true, "Validate host tools and devices before run")
	cmd.Flags().Int64Var(&cacheBytes, "cache-bytes", 0, "Assumed cache bytes for validation")
	cmd.Flags().BoolVar(&useOllama, "ollama", false, "Use Ollama LLM planner")
	cmd.Flags().StringVar(&ollamaURL, "ollama-url", "", "Ollama API URL")
	cmd.Flags().StringVar(&ollamaModel, "model", "", "Ollama model name")
	_ = cmd.MarkFlagRequired("target") // target or targets
	return cmd
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

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print StrataBench version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("%s %s\n", version.Name, version.Version)
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

func printRunSummary(run *schema.RunResult) {
	fmt.Printf("Run completed: %s\n", run.RunID)
	if run.Topology != "" {
		fmt.Printf("  Topology: %s\n", run.Topology)
	}
	if len(run.Clients) > 0 {
		fmt.Printf("  Client assignments: %d\n", len(run.Clients))
	}
	if len(run.Targets) > 1 {
		fmt.Printf("  Server targets: %d (aggregated)\n", len(run.Targets))
	}
	fmt.Printf("  IOPS: %.0f  Throughput: %.2f MB/s  p99: %.1f µs\n",
		run.Results.IOPS, run.Results.ThroughputMBps, run.Results.LatencyUS.P99)
	if run.Mock {
		fmt.Println("  (mock mode — not real storage I/O)")
	}
}
