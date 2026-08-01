package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// Info is the current git working tree state.
type Info struct {
	Repo   string
	Branch string
	SHA    string
	Dirty  bool
}

// Capture reads git metadata from repoDir. Returns empty Info when not a git repo.
func Capture(repoDir string) (Info, error) {
	if !IsRepo(repoDir) {
		return Info{Repo: repoDir}, nil
	}
	branch, err := run(repoDir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return Info{Repo: repoDir}, err
	}
	sha, err := run(repoDir, "rev-parse", "--short", "HEAD")
	if err != nil {
		return Info{Repo: repoDir, Branch: branch}, err
	}
	dirty, _ := isDirty(repoDir)
	return Info{Repo: repoDir, Branch: branch, SHA: sha, Dirty: dirty}, nil
}

// DefaultBranch returns main, master, or the current branch.
func DefaultBranch(repoDir string) string {
	for _, name := range []string{"main", "master"} {
		if BranchExists(repoDir, name) {
			return name
		}
	}
	b, _ := run(repoDir, "rev-parse", "--abbrev-ref", "HEAD")
	return b
}

// BranchExists reports whether a local or remote branch exists.
func BranchExists(repoDir, branch string) bool {
	return exec.Command("git", "-C", repoDir, "rev-parse", "--verify", branch).Run() == nil ||
		exec.Command("git", "-C", repoDir, "rev-parse", "--verify", "origin/"+branch).Run() == nil
}

// IsRepo reports whether dir is inside a git repository.
func IsRepo(dir string) bool {
	err := exec.Command("git", "-C", dir, "rev-parse", "--git-dir").Run()
	return err == nil
}

// Checkout switches branches, creating a tracking branch when needed.
func Checkout(repoDir, branch string) error {
	if err := exec.Command("git", "-C", repoDir, "checkout", branch).Run(); err == nil {
		return nil
	}
	return exec.Command("git", "-C", repoDir, "checkout", "-B", branch, "origin/"+branch).Run()
}

func isDirty(repoDir string) (bool, error) {
	out, err := run(repoDir, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

func run(repoDir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", repoDir}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}
