package orchestrator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/pratham-vishk/stratabench/internal/aggregate"
	"github.com/pratham-vishk/stratabench/internal/discovery"
	"github.com/pratham-vishk/stratabench/internal/engine"
	"github.com/pratham-vishk/stratabench/internal/profile"
	"github.com/pratham-vishk/stratabench/internal/remote"
	"github.com/pratham-vishk/stratabench/internal/schema"
	"github.com/pratham-vishk/stratabench/internal/store"
	"github.com/pratham-vishk/stratabench/internal/validator"
)

type RunOptions struct {
	Profile      *profile.Profile
	Target       string
	Clients      []string
	Mock         bool
	SkipValidate bool
	CacheBytes   int64
	WorkDir      string
	DataDir      string
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
	if len(opts.Clients) > 0 {
		return s.runDistributed(ctx, opts)
	}
	return s.runLocal(ctx, opts)
}

func (s *Service) runLocal(ctx context.Context, opts RunOptions) (*schema.RunResult, error) {
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

	return s.saveRun(opts, runID, started, durationSec, rampSec, engineName, validation, hw,
		pattern, blockSize, datasetSize, qd, threads, rwMix, directIO, *results, raw, nil)
}

func (s *Service) runDistributed(ctx context.Context, opts RunOptions) (*schema.RunResult, error) {
	validation := s.Validate(opts)
	if !opts.SkipValidate && !validation.Passed {
		return nil, fmt.Errorf("validation failed: %v", validation.Errors)
	}

	runID := uuid.New().String()
	started := time.Now().UTC()
	hw := discovery.Snapshot()

	type item struct {
		host string
		run  *schema.RunResult
		err  error
	}
	ch := make(chan item, len(opts.Clients))
	var wg sync.WaitGroup

	for _, host := range opts.Clients {
		wg.Add(1)
		go func(h string) {
			defer wg.Done()
			client := remote.NewClient(h)
			if _, err := client.Health(ctx); err != nil {
				ch <- item{host: h, err: fmt.Errorf("health check %s: %w", h, err)}
				return
			}
			run, err := client.Run(ctx, opts.Profile, opts.Target, opts.Mock, true, opts.CacheBytes)
			ch <- item{host: h, run: run, err: err}
		}(host)
	}
	wg.Wait()
	close(ch)

	var clientRuns []schema.ClientResult
	var resultSet []schema.Results
	var errs []string
	for it := range ch {
		if it.err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", it.host, it.err))
			continue
		}
		clientRuns = append(clientRuns, schema.ClientResult{Host: it.host, Results: it.run.Results})
		resultSet = append(resultSet, it.run.Results)
	}
	if len(resultSet) == 0 {
		return nil, fmt.Errorf("all clients failed: %s", strings.Join(errs, "; "))
	}

	agg := aggregate.Results(resultSet)
	pattern, blockSize, datasetSize, durationSec, rampSec, qd, threads, rwMix, directIO := opts.Profile.ToWorkload()
	engineName := opts.Profile.Engine
	if opts.Mock {
		engineName = "mock"
	}

	run, err := s.saveRun(opts, runID, started, durationSec, rampSec, engineName, validation, hw,
		pattern, blockSize, datasetSize, qd, threads, rwMix, directIO, agg, nil, clientRuns)
	if err != nil {
		return nil, err
	}
	if len(errs) > 0 {
		run.Validation.Warnings = append(run.Validation.Warnings, schema.Warning{
			Rule:     "partial_client_failure",
			Message:  strings.Join(errs, "; "),
			Severity: "warning",
		})
		_ = s.Store.Save(run)
	}
	return run, nil
}

func (s *Service) saveRun(
	opts RunOptions,
	runID string,
	started time.Time,
	durationSec, rampSec int,
	engineName string,
	validation schema.ValidationResult,
	hw schema.HardwareSnapshot,
	pattern, blockSize, datasetSize string,
	qd, threads, rwMix int,
	directIO bool,
	results schema.Results,
	raw *schema.RawEngineOutput,
	clients []schema.ClientResult,
) (*schema.RunResult, error) {
	completed := time.Now().UTC()
	steady := started.Add(time.Duration(rampSec) * time.Second)

	targetType := "block"
	if opts.Profile.Layer == "object" {
		targetType = "object"
	}

	meta := map[string]string{}
	if len(opts.Clients) > 0 {
		meta["clients"] = strings.Join(opts.Clients, ",")
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
			Metadata: meta,
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
		Results:  results,
		Hardware: hw,
		Timestamps: schema.Timestamps{
			StartedAt:            started,
			CompletedAt:          completed,
			SteadyStateReachedAt: &steady,
		},
		RawOutput: raw,
		Clients:   clients,
	}

	if err := s.Store.Save(run); err != nil {
		return nil, err
	}
	return run, nil
}
