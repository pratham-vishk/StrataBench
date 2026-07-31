package agentapi

import "github.com/pratham-vishk/stratabench/internal/schema"

const Version = "v1"

type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
	Host    string `json:"host"`
}

type RunRequest struct {
	ProfileYAML  string `json:"profile_yaml"`
	Target       string `json:"target"`
	Mock         bool   `json:"mock"`
	SkipValidate bool   `json:"skip_validate"`
	CacheBytes   int64  `json:"cache_bytes"`
	WorkDir      string `json:"work_dir,omitempty"`
}

type RunResponse struct {
	OK    bool              `json:"ok"`
	Error string            `json:"error,omitempty"`
	Run   *schema.RunResult `json:"run,omitempty"`
}
