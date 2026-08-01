package manifest

import (
	"encoding/json"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// WriteApplyResult persists apply output for the operator Job to read from shared storage.
func WriteApplyResult(path string, result *ApplyResult) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// ReadApplyResult loads apply output written by a benchmark Job.
func ReadApplyResult(path string) (*ApplyResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var result ApplyResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ToYAML serializes a benchmark manifest for ConfigMap mounting.
func (b *Benchmark) ToYAML() ([]byte, error) {
	out := *b
	if out.APIVersion == "" {
		out.APIVersion = "stratabench.io/v1alpha1"
	}
	if out.Kind == "" {
		out.Kind = "Benchmark"
	}
	return yaml.Marshal(&out)
}
