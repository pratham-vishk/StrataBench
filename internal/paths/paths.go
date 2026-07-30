package paths

import (
	"os"
	"path/filepath"
)

func RepoRoot() string {
	if v := os.Getenv("STRATABENCH_ROOT"); v != "" {
		return v
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	dir := cwd
	for i := 0; i < 6; i++ {
		if _, err := os.Stat(filepath.Join(dir, "profiles")); err == nil {
			if _, err2 := os.Stat(filepath.Join(dir, "go.mod")); err2 == nil {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return cwd
}

func ProfilesDir() string  { return filepath.Join(RepoRoot(), "profiles") }
func DataDir() string     { return filepath.Join(RepoRoot(), ".stratabench") }
func ReportsDir() string  { return filepath.Join(DataDir(), "reports") }
