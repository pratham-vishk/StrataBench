package manifest

import (
	"context"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/pratham-vishk/stratabench/internal/agentloop"
	"github.com/pratham-vishk/stratabench/internal/orchestrator"
	"github.com/pratham-vishk/stratabench/internal/paths"
	"github.com/pratham-vishk/stratabench/internal/profile"
)

// Benchmark is a declarative benchmark manifest (CRD-compatible shape).
type Benchmark struct {
	APIVersion string          `yaml:"apiVersion"`
	Kind       string          `yaml:"kind"`
	Metadata   Metadata        `yaml:"metadata"`
	Spec       BenchmarkSpec   `yaml:"spec"`
	Status     BenchmarkStatus `yaml:"status,omitempty"`
}

type Metadata struct {
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace,omitempty"`
}

type BenchmarkSpec struct {
	Profile       string   `yaml:"profile"`
	Target        string   `yaml:"target"`
	Targets       []string `yaml:"targets,omitempty"`
	Clients       []string `yaml:"clients,omitempty"`
	Topology      string   `yaml:"topology,omitempty"`
	Mock          bool     `yaml:"mock,omitempty"`
	SkipValidate  bool     `yaml:"skipValidate,omitempty"`
	CheckBaseline bool     `yaml:"checkBaseline,omitempty"`
	CheckHardware *bool    `yaml:"checkHardware,omitempty"`
	Intent        string   `yaml:"intent,omitempty"`
	UseOllama     bool     `yaml:"useOllama,omitempty"`
}

func Load(path string) (*Benchmark, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var b Benchmark
	if err := yaml.Unmarshal(data, &b); err != nil {
		return nil, err
	}
	if b.Spec.Profile == "" && b.Spec.Intent == "" {
		return nil, fmt.Errorf("spec.profile or spec.intent is required")
	}
	return &b, nil
}

func Apply(ctx context.Context, svc *orchestrator.Service, b *Benchmark) (*ApplyResult, error) {
	checkHW := effectiveCheckHardware(b.Spec)
	if b.Spec.Intent != "" {
		result, err := agentloop.Run(ctx, agentloop.Options{
			Intent:        b.Spec.Intent,
			Target:        b.Spec.Target,
			Targets:       b.Spec.Targets,
			Clients:       b.Spec.Clients,
			Topology:      b.Spec.Topology,
			Mock:          b.Spec.Mock,
			SkipValidate:  b.Spec.SkipValidate,
			CheckBaseline: b.Spec.CheckBaseline,
			CheckHardware: checkHW,
			UseOllama:     b.Spec.UseOllama,
			DataDir:       paths.DataDir(),
		})
		if err != nil {
			return nil, err
		}
		return &ApplyResult{
			RunID:   result.Run.RunID,
			Profile: result.Run.Profile,
			Status:  result.Run.Status,
		}, nil
	}
	if b.Spec.Profile == "" {
		return nil, fmt.Errorf("spec.profile or spec.intent is required")
	}
	p, err := profile.LoadByName(paths.ProfilesDir(), b.Spec.Profile)
	if err != nil {
		return nil, err
	}
	run, err := svc.Run(ctx, orchestrator.RunOptions{
		Profile:       p,
		Target:        b.Spec.Target,
		Targets:       b.Spec.Targets,
		Clients:       b.Spec.Clients,
		Topology:      b.Spec.Topology,
		Mock:          b.Spec.Mock,
		SkipValidate:  b.Spec.SkipValidate,
		CheckBaseline: b.Spec.CheckBaseline,
		CheckHardware: checkHW,
		DataDir:       paths.DataDir(),
	})
	if err != nil {
		return nil, err
	}
	return &ApplyResult{RunID: run.RunID, Profile: run.Profile, Status: run.Status}, nil
}

// effectiveCheckHardware defaults to true for real runs; mock skips checks.
func effectiveCheckHardware(spec BenchmarkSpec) bool {
	if spec.Mock {
		return false
	}
	if spec.CheckHardware != nil {
		return *spec.CheckHardware
	}
	return true
}

type ApplyResult struct {
	RunID   string `json:"run_id"`
	Profile string `json:"profile"`
	Status  string `json:"status"`
}
