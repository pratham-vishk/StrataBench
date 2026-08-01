package provenance

import (
	"github.com/pratham-vishk/stratabench/internal/git"
	"github.com/pratham-vishk/stratabench/internal/schema"
	"github.com/pratham-vishk/stratabench/internal/version"
)

// Capture records git and tool version for a benchmark run.
func Capture(repoDir, buildCmd, compareRole string) schema.Provenance {
	p := schema.Provenance{
		BuildCmd:    buildCmd,
		ToolVersion: version.Version,
		CompareRole: compareRole,
	}
	if repoDir == "" {
		return p
	}
	info, err := git.Capture(repoDir)
	if err != nil || info.Branch == "" {
		p.GitRepo = repoDir
		return p
	}
	p.GitRepo = info.Repo
	p.GitBranch = info.Branch
	p.GitSHA = info.SHA
	p.GitDirty = info.Dirty
	return p
}

// Label returns a short human-readable provenance string.
func Label(p schema.Provenance) string {
	if p.GitBranch != "" {
		s := p.GitBranch
		if p.GitSHA != "" {
			s += "@" + p.GitSHA
		}
		if p.GitDirty {
			s += " (dirty)"
		}
		return s
	}
	return "—"
}
