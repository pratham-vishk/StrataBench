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
	case "stratabench":
		return &MockRunner{} // native Rust engine lands in a later phase
	default:
		return &MockRunner{}
	}
}
