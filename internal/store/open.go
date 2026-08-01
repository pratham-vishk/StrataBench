package store

import (
	"os"
	"path/filepath"
)

// OpenDefault opens PostgreSQL when STRATABENCH_DATABASE_URL is set,
// otherwise SQLite at dataDir/stratabench.db.
func OpenDefault(dataDir string) (*Store, error) {
	if dsn := os.Getenv("STRATABENCH_DATABASE_URL"); dsn != "" {
		return OpenPostgres(dsn)
	}
	return Open(filepath.Join(dataDir, "stratabench.db"))
}
