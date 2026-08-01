package report

import (
	"fmt"
	"strings"

	"github.com/pratham-vishk/stratabench/internal/schema"
)

// displayLayer is the customer-facing storage layer label.
func displayLayer(layer string) string {
	switch strings.ToLower(strings.TrimSpace(layer)) {
	case "application":
		return "Application"
	case "block":
		return "Block"
	case "object":
		return "Object"
	case "file":
		return "File"
	case "":
		return "Storage"
	default:
		return strings.ToUpper(layer[:1]) + layer[1:]
	}
}

// displayEngine returns a public engine label, or "" when it should stay internal.
func displayEngine(engine string) string {
	switch strings.ToLower(strings.TrimSpace(engine)) {
	case "sbk":
		return ""
	case "fio":
		return "FIO"
	case "warp":
		return "Warp"
	case "":
		return ""
	default:
		return strings.ToUpper(engine)
	}
}

func benchmarkLabel(run *schema.RunResult) string {
	layer := displayLayer(run.Layer)
	if eng := displayEngine(run.Engine); eng != "" {
		return fmt.Sprintf("%s / %s", layer, eng)
	}
	return layer
}
