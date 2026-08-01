package manifest

// BenchmarkStatus is written to CR .status by the operator.
type BenchmarkStatus struct {
	Phase   string `json:"phase,omitempty"`
	RunID   string `json:"runId,omitempty"`
	Message string `json:"message,omitempty"`
}

const (
	PhasePending   = "Pending"
	PhaseRunning   = "Running"
	PhaseCompleted = "Completed"
	PhaseFailed    = "Failed"
)
