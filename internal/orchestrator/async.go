package orchestrator

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/pratham-vishk/stratabench/internal/metrics"
	"github.com/pratham-vishk/stratabench/internal/runstate"
	"github.com/pratham-vishk/stratabench/internal/schema"
)

// StartAsyncRun validates synchronously, persists a running placeholder, and executes in the background.
func (s *Service) StartAsyncRun(ctx context.Context, opts RunOptions) (string, error) {
	opts.Profile = applyWarpClients(opts.Profile, opts.WarpClients)
	if opts.DataDir == "" {
		return "", fmt.Errorf("data dir required for async runs")
	}
	if opts.WorkDir == "" {
		opts.WorkDir = filepath.Join(opts.DataDir, "work")
	}

	validation := s.Validate(opts)
	if !opts.SkipValidate && !validation.Passed {
		return "", fmt.Errorf("validation failed: %v", validation.Errors)
	}

	runID := uuid.New().String()
	opts.RunID = runID
	started := time.Now().UTC()

	placeholder := &schema.RunResult{
		SchemaVersion: schema.SchemaVersion,
		RunID:         runID,
		Profile:       opts.Profile.Name,
		Layer:         opts.Profile.Layer,
		Engine:        opts.Profile.Engine,
		Status:        "running",
		Mock:          opts.Mock,
		Validation:    validation,
		Timestamps:    schema.Timestamps{StartedAt: started},
	}
	if err := s.Store.Save(placeholder); err != nil {
		return "", err
	}

	progress := runstate.Progress{
		RunID:     runID,
		Phase:     "queued",
		Profile:   opts.Profile.Name,
		StartedAt: started,
	}
	runstate.Set(progress)
	recordRunProgress(progress)

	go func() {
		bgCtx := context.Background()
		run, err := s.Run(bgCtx, opts)
		if err != nil {
			failed := *placeholder
			failed.Status = "failed"
			failed.Timestamps.CompletedAt = time.Now().UTC()
			failed.Validation.Errors = append(failed.Validation.Errors, err.Error())
			_ = s.Store.Save(&failed)
			runstate.Set(runstate.Progress{
				RunID:     runID,
				Phase:     "failed",
				Profile:   opts.Profile.Name,
				StartedAt: started,
				Error:     err.Error(),
			})
			metrics.ClearProgress(runID)
			return
		}
		_ = run
		metrics.ClearProgress(runID)
	}()

	return runID, nil
}
