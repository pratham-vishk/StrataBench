package engine

// SBKRunner runs SBK-style application workloads with native driver detection.
type SBKRunner struct{}

func (s *SBKRunner) Name() string { return "sbk" }
