package engine

import (
	"context"
	"fmt"

	"github.com/pratham-vishk/stratabench/internal/schema"
)

// errRunner fails runs for unsupported engines (honest failure instead of silent mock).
type errRunner struct {
	name string
	err  error
}

func (e *errRunner) Name() string { return e.name }

func (e *errRunner) Run(ctx context.Context, _ RunInput) (*schema.Results, *schema.RawEngineOutput, error) {
	if ctx.Err() != nil {
		return nil, nil, ctx.Err()
	}
	return nil, nil, e.err
}

func unsupportedEngine(name string) Runner {
	return &errRunner{
		name: name,
		err:  fmt.Errorf("engine %q is not available (install the tool or use --mock)", name),
	}
}

func nativeEnginePending(name string) Runner {
	return &errRunner{
		name: name,
		err:  fmt.Errorf("engine %q native implementation is not yet available; use an external engine profile or --mock", name),
	}
}
