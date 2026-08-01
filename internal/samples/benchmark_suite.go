package samples

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pratham-vishk/stratabench/internal/analyst"
	"github.com/pratham-vishk/stratabench/internal/compare"
	"github.com/pratham-vishk/stratabench/internal/importsbk"
	"github.com/pratham-vishk/stratabench/internal/orchestrator"
	"github.com/pratham-vishk/stratabench/internal/paths"
	"github.com/pratham-vishk/stratabench/internal/profile"
	"github.com/pratham-vishk/stratabench/internal/report"
	"github.com/pratham-vishk/stratabench/internal/schema"
)

// SuiteOutput paths for HTML-only benchmark samples.
type SuiteOutput struct {
	BaseHTML      string
	CandidateHTML string
	CompareHTML   string
	SBKHTML       string
	S3HTML        string
}

// RunSequentialBenchmark runs base then candidate (never parallel) and writes compare HTML.
func RunSequentialBenchmark(ctx context.Context, svc *orchestrator.Service, outDir string) (base, candidate *schema.RunResult, err error) {
	prof, err := profile.LoadByName(paths.ProfilesDir(), "nvme-random-oltp")
	if err != nil {
		return nil, nil, err
	}
	opts := orchestrator.RunOptions{
		Profile: prof, Target: "/dev/null", Mock: true, SkipValidate: true,
		DataDir: paths.DataDir(),
	}

	// Step 1: base benchmark (must complete before candidate)
	base, err = svc.Run(ctx, opts)
	if err != nil {
		return nil, nil, fmt.Errorf("base benchmark: %w", err)
	}
	base.Provenance.CompareRole = "base"

	// Step 2: candidate benchmark (sequential — relies on base being done)
	candidate, err = svc.Run(ctx, opts)
	if err != nil {
		return nil, nil, fmt.Errorf("candidate benchmark: %w", err)
	}
	candidate.Provenance.CompareRole = "head"

	if err := WriteBenchmarkSuite(outDir, base, candidate); err != nil {
		return base, candidate, err
	}
	return base, candidate, nil
}

// WriteBenchmarkSuite writes HTML-only base, candidate, and compare reports.
func WriteBenchmarkSuite(outDir string, base, candidate *schema.RunResult) error {
	if err := mkdir(outDir); err != nil {
		return err
	}
	basePath := filepath.Join(outDir, "base-benchmark.html")
	candPath := filepath.Join(outDir, "candidate-benchmark.html")
	cmpPath := filepath.Join(outDir, "compare-sample.html")

	baseSummary := fmt.Sprintf("Base benchmark — %s on %s. Run this first; candidate compares against this baseline.",
		base.Profile, base.Target.Device)
	candSummary := fmt.Sprintf("Candidate benchmark — %s (sequential run after base). Compared in compare-sample.html.",
		candidate.Profile)

	if err := report.WriteHTMLOnly(base, report.Options{
		Summary:  baseSummary,
		Insights: analyst.Analyze(base, nil),
	}, basePath); err != nil {
		return err
	}
	if err := report.WriteHTMLOnly(candidate, report.Options{
		Summary:  candSummary,
		Insights: analyst.Analyze(candidate, nil),
	}, candPath); err != nil {
		return err
	}

	diff := compare.Diff(base, candidate)
	return report.WriteCompareHTML(base, candidate, diff, cmpPath, "base-benchmark.html", "candidate-benchmark.html")
}

// WriteSBKSample imports SBK CSV when available.
func WriteSBKSample(outDir string) error {
	run, summary, err := loadSBKRun()
	if err != nil {
		return err
	}
	if err := mkdir(outDir); err != nil {
		return err
	}
	return report.WriteHTMLOnly(run, report.Options{Summary: summary}, filepath.Join(outDir, "sample-result.html"))
}

// WriteS3Sample runs S3 mock benchmark sequentially.
func WriteS3Sample(ctx context.Context, svc *orchestrator.Service, outDir string) error {
	if err := mkdir(outDir); err != nil {
		return err
	}
	prof, err := profile.LoadByName(paths.ProfilesDir(), "s3-put-throughput")
	if err != nil {
		return err
	}
	run, err := svc.Run(ctx, orchestrator.RunOptions{
		Profile: prof, Target: "127.0.0.1:9000", Mock: true, SkipValidate: true,
		DataDir: paths.DataDir(),
	})
	if err != nil {
		return err
	}
	return report.WriteHTMLOnly(run, report.Options{
		Summary: "S3 PUT benchmark — Grafana operations dashboard with dynamic Ops/s and PUT charts.",
	}, filepath.Join(outDir, "s3-put-sample.html"))
}

// GenerateAll creates all HTML sample reports.
func GenerateAll(ctx context.Context, svc *orchestrator.Service, outDir string) error {
	if _, _, err := RunSequentialBenchmark(ctx, svc, outDir); err != nil {
		return err
	}
	if err := WriteSBKSample(outDir); err != nil {
		fmt.Fprintf(os.Stderr, "sbk sample skipped: %v\n", err)
	}
	if err := WriteS3Sample(ctx, svc, outDir); err != nil {
		return err
	}
	if err := WriteMultiNodeSample(outDir); err != nil {
		return fmt.Errorf("multi-node sample: %w", err)
	}
	return nil
}

func mkdir(dir string) error {
	return os.MkdirAll(dir, 0o755)
}

func loadSBKRun() (*schema.RunResult, string, error) {
	sbkPath := filepath.Join(".tmp", "sbk-charts", "samples", "charts", "sbk-file-read.csv")
	runs, err := importsbk.ParseCSV(sbkPath)
	if err != nil || len(runs) == 0 {
		return nil, "", fmt.Errorf("SBK CSV not found (%s)", sbkPath)
	}
	run := runs[0]
	run.RunID = "sample-file-read"
	return run, "SBK file-read — Grafana operations dashboard, intervals, totals, full percentile suite.", nil
}
