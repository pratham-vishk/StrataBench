package report

import (
	"github.com/pratham-vishk/stratabench/internal/schema"
)

// WriteHTMLOnly writes a single HTML report (no Excel/JSON).
func WriteHTMLOnly(run *schema.RunResult, opts Options, outPath string) error {
	return WriteHTMLWithOptions(run, opts, outPath)
}
