package export

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/pratham-vishk/stratabench/internal/schema"
)

func WriteJSON(run *schema.RunResult, path string) error {
	data, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return err
	}
	fmt.Printf("exported JSON: %s\n", path)
	return nil
}
