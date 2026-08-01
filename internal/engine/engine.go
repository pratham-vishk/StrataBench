package engine

import (
	"context"

	"github.com/pratham-vishk/stratabench/internal/profile"
	"github.com/pratham-vishk/stratabench/internal/schema"
)

type RunInput struct {
	Profile *profile.Profile
	Target  string
	Mock    bool
	WorkDir string
}

type Runner interface {
	Name() string
	Run(ctx context.Context, in RunInput) (*schema.Results, *schema.RawEngineOutput, error)
}

func ForProfile(p *profile.Profile, mock bool) Runner {
	if mock || p.Engine == "mock" {
		return &MockRunner{}
	}
	switch p.Engine {
	case "fio":
		return &FioRunner{}
	case "warp":
		return &WarpRunner{}
	case "elbencho":
		return &ElbenchoRunner{}
	case "spdk":
		return &SPDKRunner{}
	case "vdbench":
		return &VdbenchRunner{}
	case "sbk":
		return &SBKRunner{}
	case "gosbench":
		return unsupportedEngine("gosbench")
	case "stratabench":
		if mock {
			return &MockRunner{}
		}
		return nativeEnginePending("stratabench")
	default:
		if mock {
			return &MockRunner{}
		}
		return unsupportedEngine(p.Engine)
	}
}
