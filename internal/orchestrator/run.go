package orchestrator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/pratham-vishk/stratabench/internal/discovery"
	"github.com/pratham-vishk/stratabench/internal/engine"
	"github.com/pratham-vishk/stratabench/internal/profile"
	"github.com/pratham-vishk/stratabench/internal/schema"
	"github.com/pratham-vishk/stratabench/internal/store"
	"github.com/pratham-vishk/stratabench/internal/validator"
)

type RunOptions struct {
	Profile     *profile.Profile
	Target      string
	Mock        bool
	SkipValidate bool
	CacheBytes  int64
	WorkDir     string
	DataDir     string
}

type Service struct {
	Store *store.Store
}

func NewService(dataDir string) (*Service, error) {
	dbPath := filepath.Join(dataDir, "stratabench.db")
	st, err := store.Open(dbPath)
	if err != nil {
		return nil, err
	}
	return &Service{Store: st}, nil
}

func (s *Service) Close() error { return s.Store.Close() }

func (s *Service) Validate(opts RunOptions) schema.ValidationResult {
	hw := discovery.Snapshot()
	cache := opts.CacheBytes
	if cache == 0 {
		cache = hw.CacheBytes
	}
	return validator.Validate(opts.Profile, validator.Options{CacheBytes: cache})
}

func (s *Service) Run(ctx context.Context, opts RunOptions) (*schema.RunResult, error) {
	if opts.WorkDir == "" {
		opts.WorkDir = filepath.Join(opts.DataDir, "work")
	}
	if err := os.MkdirAll(opts.WorkDir, 0o755); err != nil {
		return nil, err
	}

	hw := discovery.Snapshot()
	validation := s.Validate(opts)
	if !opts.SkipValidate && !validation.Passed {
		return nil, fmt.Errorf("validation failed: %v", validation.Errors)
	}

	runID := uuid.New().String()
	started := time.Now().UTC()

	pattern, blockSize, datasetSize, durationSec, rampSec, qd, threads, rwMix, directIO := opts.Profile.ToWorkload()
	engineName := opts.Profile.Engine
	if opts.Mock {
		engineName = "mock"
	}

	runner := engine.ForProfile(opts.Profile, opts.Mock)
	results, raw, err := runner.Run(ctx, engine.RunInput{
		Profile: opts.Profile,
		Target:  opts.Target,
		Mock:    opts.Mock,
		WorkDir: opts.WorkDir,
	})
	if err != nil {
		return nil, err
	}

	completed := time.Now().UTC()
	steady := started.Add(time.Duration(rampSec) * time.Second)

	targetType := "block"
	if opts.Profile.Layer == "object" {
		targetType = "object"
	}

	run := &schema.RunResult{
		SchemaVersion: schema.SchemaVersion,
		RunID:         runID,
		Profile:       opts.Profile.Name,
		Layer:         opts.Profile.Layer,
		Engine:        engineName,
		Status:        "completed",
		Mock:          opts.Mock,
		Validation:    validation,
		Target: schema.Target{
			Type:     targetType,
			Device:   opts.Target,
			Endpoint: opts.Target,
			VM:       nil,
		},
		Workload: schema.Workload{
			Pattern:      pattern,
			BlockSize:    blockSize,
			ReadWriteMix: rwMix,
			QueueDepth:   qd,
			Threads:      threads,
			DatasetSize:  datasetSize,
			DurationSec:  durationSec,
			RampTimeSec:  rampSec,
			DirectIO:     directIO,
		},
		Results:  *results,
		Hardware: hw,
		Timestamps: schema.Timestamps{
			StartedAt:            started,
			CompletedAt:          completed,
			SteadyStateReachedAt: &steady,
		},
		RawOutput: raw,
	}

	if err := s.Store.Save(run); err != nil {
		return nil, err
	}
	return run, nil
}
