package compare

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pratham-vishk/stratabench/internal/git"
	"github.com/pratham-vishk/stratabench/internal/orchestrator"
	"github.com/pratham-vishk/stratabench/internal/provenance"
	"github.com/pratham-vishk/stratabench/internal/schema"
)

// BranchOptions configures an automated base vs head benchmark.
type BranchOptions struct {
	RepoDir      string
	BaseBranch   string
	HeadBranch   string
	BuildCmd     string
	AllowDirty   bool
	SkipBuild    bool
	Run          orchestrator.RunOptions
}

// BranchResult holds both runs and the computed diff.
type BranchResult struct {
	BaseRun *schema.RunResult
	HeadRun *schema.RunResult
	Diff    DiffResult
}

// CompareBranches checks out each branch, builds, benchmarks, and restores git state.
func CompareBranches(ctx context.Context, svc *orchestrator.Service, opts BranchOptions) (*BranchResult, error) {
	repo := opts.RepoDir
	if repo == "" {
		return nil, fmt.Errorf("repo directory is required")
	}
	if !git.IsRepo(repo) {
		return nil, fmt.Errorf("%s is not a git repository", repo)
	}
	if opts.BaseBranch == "" || opts.HeadBranch == "" {
		return nil, fmt.Errorf("base and head branches are required")
	}

	orig, err := git.Capture(repo)
	if err != nil {
		return nil, err
	}
	defer func() { _ = git.Checkout(repo, orig.Branch) }()

	if !opts.AllowDirty && orig.Dirty {
		return nil, fmt.Errorf("working tree is dirty — commit/stash or pass --allow-dirty")
	}

	baseRun, err := benchBranch(ctx, svc, opts, opts.BaseBranch, "base")
	if err != nil {
		return nil, fmt.Errorf("base branch %s: %w", opts.BaseBranch, err)
	}
	headRun, err := benchBranch(ctx, svc, opts, opts.HeadBranch, "head")
	if err != nil {
		return nil, fmt.Errorf("head branch %s: %w", opts.HeadBranch, err)
	}

	diff := Diff(baseRun, headRun)
	return &BranchResult{BaseRun: baseRun, HeadRun: headRun, Diff: diff}, nil
}

func benchBranch(ctx context.Context, svc *orchestrator.Service, opts BranchOptions, branch, role string) (*schema.RunResult, error) {
	if err := git.Checkout(opts.RepoDir, branch); err != nil {
		return nil, fmt.Errorf("git checkout: %w", err)
	}
	if !opts.SkipBuild && strings.TrimSpace(opts.BuildCmd) != "" {
		if err := runBuild(opts.RepoDir, opts.BuildCmd); err != nil {
			return nil, err
		}
	}
	runOpts := opts.Run
	runOpts.Provenance = provenance.Capture(opts.RepoDir, opts.BuildCmd, role)
	runOpts.Provenance.GitBranch = branch
	if info, err := git.Capture(opts.RepoDir); err == nil {
		runOpts.Provenance.GitSHA = info.SHA
		runOpts.Provenance.GitDirty = info.Dirty
	}
	return svc.Run(ctx, runOpts)
}

func runBuild(repoDir, buildCmd string) error {
	parts := strings.Fields(buildCmd)
	if len(parts) == 0 {
		return nil
	}
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Dir = repoDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("build failed (%s in %s): %w", buildCmd, repoDir, err)
	}
	return nil
}

// DefaultRepo returns the directory to use for git provenance.
func DefaultRepo(explicit string) string {
	if explicit != "" {
		return explicit
	}
	if v := os.Getenv("STRATABENCH_GIT_REPO"); v != "" {
		return v
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	if git.IsRepo(cwd) {
		return cwd
	}
	return cwd
}

// CompareReportPath returns the default HTML path for a branch comparison.
func CompareReportPath(baseID, headID string) string {
	return filepath.Join(".stratabench", "reports", fmt.Sprintf("compare-%s-vs-%s.html", shortID(baseID), shortID(headID)))
}

func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}
