package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/pratham-vishk/stratabench/internal/analyst"
	"github.com/pratham-vishk/stratabench/internal/lab"
	"github.com/pratham-vishk/stratabench/internal/orchestrator"
	"github.com/pratham-vishk/stratabench/internal/paths"
	"github.com/pratham-vishk/stratabench/internal/profile"
	"github.com/pratham-vishk/stratabench/internal/report"
)

func labCmd() *cobra.Command {
	var configPath string
	var hostsCSV string
	var applyFirewall bool
	var writeEnv string

	cmd := &cobra.Command{
		Use:   "lab",
		Short: "Discover, bootstrap, and run benchmarks on a lab cluster",
		Long: `End-to-end lab workflow: discover nodes, install warp/fio/agent, deploy MinIO,
open firewall ports, sync after code changes, and run benchmarks.

  cp examples/lab.yaml.example lab.yaml   # edit hosts
  make build
  stratabench lab bootstrap -f lab.yaml
  stratabench lab check -f lab.yaml
  stratabench lab validate -f lab.yaml
  stratabench lab run -f lab.yaml`,
	}

	cmd.PersistentFlags().StringVarP(&configPath, "file", "f", "lab.yaml", "lab config (yaml or .env)")

	validateCmd := &cobra.Command{
		Use:   "validate",
		Short: "Print Dell lab sign-off matrix and verify readiness",
		Long: `Runs lab check, prints the hardware validation matrix with resolved targets,
and optional profile smoke validation (mock). Execute the printed commands on lab hardware.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := lab.LoadConfig(configPath)
			if err != nil {
				return err
			}
			smoke, _ := cmd.Flags().GetBool("smoke")
			rep, err := lab.Validate(cmd.Context(), cfg, paths.ProfilesDir(), smoke)
			if err != nil {
				return err
			}
			lab.PrintValidationReport(rep)
			if rep.Check != nil && !rep.Check.Ready {
				return fmt.Errorf("lab not ready")
			}
			if smoke && rep.SmokeFailed > 0 {
				return fmt.Errorf("smoke validation failed")
			}
			return nil
		},
	}
	validateCmd.Flags().Bool("smoke", false, "run mock profile validation smoke tests")

	cmd.AddCommand(
		&cobra.Command{
			Use:   "init",
			Short: "Write default lab.yaml",
			RunE: func(cmd *cobra.Command, args []string) error {
				out := "lab.yaml"
				if len(args) > 0 {
					out = args[0]
				}
				if _, err := os.Stat(out); err == nil {
					return fmt.Errorf("%s already exists", out)
				}
				return lab.SaveConfig(out, lab.DefaultConfig())
			},
		},
		&cobra.Command{
			Use:   "discover",
			Short: "Probe hosts and classify client vs S3 roles",
			RunE: func(cmd *cobra.Command, args []string) error {
				cfg, err := lab.LoadConfig(configPath)
				if err != nil {
					return err
				}
				hosts := splitCSV(hostsCSV)
				if len(hosts) == 0 {
					hosts = uniqueLabHosts(cfg)
				}
				if len(hosts) == 0 {
					return fmt.Errorf("pass --hosts or set clients/servers in config")
				}
				r := lab.Runner{SSHUser: cfg.SSH.User, SSHKey: cfg.SSH.Key}
				extra := []string{}
				if cfg.Tools.InstallVdbench {
					extra = append(extra, "vdbench")
				}
				statuses, err := lab.DiscoverHosts(cmd.Context(), r, hosts, cfg.AgentPort, 9000, extra)
				if err != nil {
					return err
				}
				for _, st := range statuses {
					fmt.Printf("%s  agent=%v s3=%v tools=%v → %s\n",
						st.Host, st.AgentOK, st.S3OK, st.Tools, st.Suggested)
				}
				lab.ApplyDiscovery(&cfg, statuses)
				return lab.SaveConfig(configPath, cfg)
			},
		},
		&cobra.Command{
			Use:   "bootstrap",
			Short: "Install warp, fio, agent; deploy MinIO; optional firewall",
			RunE: func(cmd *cobra.Command, args []string) error {
				cfg, err := lab.LoadConfig(configPath)
				if err != nil {
					return err
				}
				cfg.Firewall.Apply = applyFirewall
				rep, err := lab.Bootstrap(cmd.Context(), cfg, paths.RepoRoot())
				if err != nil {
					return err
				}
				lab.PrintBootstrapReport(rep)
				if writeEnv != "" {
					_ = lab.WriteEnvFile(writeEnv, cfg)
				}
				return nil
			},
		},
		&cobra.Command{
			Use:   "deploy-s3",
			Short: "Deploy MinIO (Docker) on server nodes",
			RunE: func(cmd *cobra.Command, args []string) error {
				cfg, err := lab.LoadConfig(configPath)
				if err != nil {
					return err
				}
				r := lab.Runner{SSHUser: cfg.SSH.User, SSHKey: cfg.SSH.Key}
				for _, s := range cfg.Servers {
					if err := lab.DeployMinIO(cmd.Context(), r, cfg, s.Host); err != nil {
						return fmt.Errorf("%s: %w", s.Host, err)
					}
					fmt.Printf("OK minio on %s\n", s.Host)
				}
				return nil
			},
		},
		validateCmd,
		&cobra.Command{
			Use:   "check",
			Short: "Verify agents, warp/fio, and S3 endpoints",
			RunE: func(cmd *cobra.Command, args []string) error {
				cfg, err := lab.LoadConfig(configPath)
				if err != nil {
					return err
				}
				rep, err := lab.Check(cmd.Context(), cfg)
				if err != nil {
					return err
				}
				lab.PrintCheckReport(rep, cfg)
				if !rep.Ready {
					return fmt.Errorf("lab not ready")
				}
				return nil
			},
		},
		&cobra.Command{
			Use:   "sync",
			Short: "Push rebuilt binaries to clients (after code changes)",
			RunE: func(cmd *cobra.Command, args []string) error {
				cfg, err := lab.LoadConfig(configPath)
				if err != nil {
					return err
				}
				return lab.Sync(cmd.Context(), cfg, paths.RepoRoot())
			},
		},
		&cobra.Command{
			Use:   "firewall",
			Short: "Print or apply ufw rules for lab ports",
			RunE: func(cmd *cobra.Command, args []string) error {
				cfg, err := lab.LoadConfig(configPath)
				if err != nil {
					return err
				}
				script := lab.FirewallScript(cfg.Firewall.OpenPorts)
				if !applyFirewall {
					fmt.Print(script)
					return nil
				}
				r := lab.Runner{SSHUser: cfg.SSH.User, SSHKey: cfg.SSH.Key}
				for _, h := range uniqueLabHosts(cfg) {
					if err := lab.ApplyFirewall(cmd.Context(), r, h, cfg.Firewall.OpenPorts); err != nil {
						fmt.Printf("WARN %s: %v\n", h, err)
					} else {
						fmt.Printf("OK firewall %s\n", h)
					}
				}
				return nil
			},
		},
		&cobra.Command{
			Use:   "run",
			Short: "Run benchmark on lab cluster",
			RunE: func(cmd *cobra.Command, args []string) error {
				cfg, err := lab.LoadConfig(configPath)
				if err != nil {
					return err
				}
				profName := cfg.DefaultRun.Profile
				if len(args) > 0 {
					profName = args[0]
				}
				p, err := profile.LoadByName(paths.ProfilesDir(), profName)
				if err != nil {
					return err
				}
				target := cfg.PrimaryTarget()
				if target == "" {
					return fmt.Errorf("no server targets in lab config")
				}
				os.Setenv("WARP_ACCESS_KEY", cfg.S3.AccessKey)
				os.Setenv("WARP_SECRET_KEY", cfg.S3.SecretKey)

				svc, err := orchestrator.NewService(paths.DataDir())
				if err != nil {
					return err
				}
				defer svc.Close()

				topo := cfg.DefaultRun.Topology
				if topo == "" {
					topo = "shard"
				}
				var clientHosts []string
				for _, c := range cfg.Clients {
					clientHosts = append(clientHosts, fmt.Sprintf("%s:%d", c.Host, c.Port))
				}
				var serverHosts []string
				for _, s := range cfg.Servers {
					serverHosts = append(serverHosts, fmt.Sprintf("%s:%d", s.Host, s.Port))
				}

				run, err := svc.Run(cmd.Context(), orchestrator.RunOptions{
					Profile:       p,
					Target:        target,
					Targets:       serverHosts,
					Clients:       clientHosts,
					Topology:      topo,
					Mock:          mock,
					CheckHardware: checkHardware && !mock,
					DataDir:       paths.DataDir(),
				})
				if err != nil {
					return err
				}
				alerts := svc.CheckRegression(run)
				insights := analyst.Analyze(run, alerts)
				arts, err := report.WriteRunArtifacts(run, report.OptionsFromAnalysis(insights, "", alerts))
				if err != nil {
					return err
				}
				fmt.Printf("run_id=%s profile=%s iops=%.0f throughput=%.1f MB/s\n",
					run.RunID, run.Profile, run.Results.IOPS, run.Results.ThroughputMBps)
				fmt.Printf("Report: %s\nExcel: %s\n", arts.HTML, arts.Excel)
				if openReport {
					_ = report.OpenInBrowser(arts.HTML)
				}
				return nil
			},
		},
	)

	cmd.PersistentFlags().StringVar(&hostsCSV, "hosts", "", "comma-separated hosts to discover")
	cmd.PersistentFlags().BoolVar(&applyFirewall, "firewall", false, "apply firewall rules during bootstrap")
	cmd.PersistentFlags().StringVar(&writeEnv, "write-env", "", "write lab.env after bootstrap")

	return cmd
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func uniqueLabHosts(cfg lab.Config) []string {
	seen := map[string]bool{}
	var out []string
	for _, h := range cfg.ClientHosts() {
		if !seen[h] {
			seen[h] = true
			out = append(out, h)
		}
	}
	for _, h := range cfg.ServerHosts() {
		if !seen[h] {
			seen[h] = true
			out = append(out, h)
		}
	}
	return out
}
