package main

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/pratham-vishk/stratabench/internal/samples"
)

func sampleCompareCmd() *cobra.Command {
	var outputDir string
	cmd := &cobra.Command{
		Use:   "sample-compare",
		Short: "Run base benchmark, then candidate, then write compare HTML (sequential, not parallel)",
		Long: `Sequential benchmark workflow for comparison:

  1. Run base benchmark (mock) → base-benchmark.html
  2. Run candidate benchmark → candidate-benchmark.html
  3. Generate compare-sample.html with Grafana overlay charts

Cannot compare in parallel — candidate relies on completed base run.

  stratabench sample-compare
  stratabench sample-compare --open-report`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = initDirs()
			svc, err := newService()
			if err != nil {
				return err
			}
			defer svc.Close()

			fmt.Println("Step 1/3: Running base benchmark...")
			base, candidate, err := samples.RunSequentialBenchmark(cmd.Context(), svc, outputDir)
			if err != nil {
				return err
			}
			printRunSummary(base)
			fmt.Println("\nStep 2/3: Candidate benchmark completed.")
			printRunSummary(candidate)

			cmpPath := filepath.Join(outputDir, "compare-sample.html")
			fmt.Println("\nStep 3/3: Compare report written.")
			fmt.Println("  Base:      ", filepath.Join(outputDir, "base-benchmark.html"))
			fmt.Println("  Candidate: ", filepath.Join(outputDir, "candidate-benchmark.html"))
			fmt.Println("  Compare:   ", cmpPath)

			if openReport {
				_ = reportOpen(cmpPath)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&outputDir, "output", filepath.Join("examples", "sample-report", "output"), "Output directory")
	cmd.Flags().BoolVar(&openReport, "open-report", false, "Open compare HTML in browser")
	return cmd
}
