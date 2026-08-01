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
	"github.com/pratham-vishk/stratabench/internal/baseline"
	"github.com/pratham-vishk/stratabench/internal/discovery"
	"github.com/pratham-vishk/stratabench/internal/engine"
	"github.com/pratham-vishk/stratabench/internal/inventory"
	"github.com/pratham-vishk/stratabench/internal/metrics"
	"github.com/pratham-vishk/stratabench/internal/profile"
	"github.com/pratham-vishk/stratabench/internal/remote"
	"github.com/pratham-vishk/stratabench/internal/schema"
	"github.com/pratham-vishk/stratabench/internal/store"
	"github.com/pratham-vishk/stratabench/internal/topology"
	"github.com/pratham-vishk/stratabench/internal/validator"
)

type RunOptions struct {
	Profile       *profile.Profile
	Target        string
	Targets       []string
	Clients       []string
	Topology      string
	Mock          bool
	SkipValidate  bool
	CheckBaseline bool
	CheckHardware bool
	CacheBytes    int64
	WorkDir       string
	DataDir       string
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
	targets := topology.MergeTargets(opts.Target, opts.Targets)
	primaryTarget := opts.Target
	if primaryTarget == "" && len(targets) > 0 {
		primaryTarget = targets[0]
	}
	return validator.Validate(opts.Profile, validator.Options{
		CacheBytes:    cache,
		CheckHardware: opts.CheckHardware,
		Target:        primaryTarget,
		Mock:          opts.Mock,
		Hardware:      hw,
	})
}

func (s *Service) Run(ctx context.Context, opts RunOptions) (*schema.RunResult, error) {
	targets := topology.MergeTargets(opts.Target, opts.Targets)
	plan, err := topology.Build(opts.Topology, opts.Clients, targets)
	if err != nil {
		return nil, err
	}

	if opts.WorkDir == "" {
		opts.WorkDir = filepath.Join(opts.DataDir, "work")
	}
	if err := os.MkdirAll(opts.WorkDir, 0o755); err != nil {
		return nil, err
	}

	validation := s.Validate(opts)
	if !opts.SkipValidate && !validation.Passed {
		return nil, fmt.Errorf("validation failed: %v", validation.Errors)
	}

	runID := uuid.New().String()
	started := time.Now().UTC()
	hw := discovery.Snapshot()

	type item struct {
		assignment topology.Assignment
		results    schema.Results
		err        error
	}

	ch := make(chan item, len(plan.Assignments))
	var wg sync.WaitGroup

	for _, a := range plan.Assignments {
		wg.Add(1)
		go func(assign topology.Assignment) {
			defer wg.Done()
			res, err := s.runAssignment(ctx, opts, assign)
			ch <- item{assignment: assign, results: res, err: err}
		}(a)
	}
	wg.Wait()
	close(ch)

	var clientRuns []schema.ClientResult
	var resultSet []schema.Results
	var errs []string
	targetBuckets := map[string][]schema.Results{}

	for it := range ch {
		if it.err != nil {
			label := assignmentLabel(it.assignment)
			errs = append(errs, fmt.Sprintf("%s: %v", label, it.err))
			continue
		}
		resultSet = append(resultSet, it.results)
		targetBuckets[it.assignment.Target] = append(targetBuckets[it.assignment.Target], it.results)
		if it.assignment.Client != "" {
			clientRuns = append(clientRuns, schema.ClientResult{
				Host:    it.assignment.Client,
				Target:  it.assignment.Target,
				Results: it.results,
			})
		}
	}
	if len(resultSet) == 0 {
		return nil, fmt.Errorf("all assignments failed: %s", strings.Join(errs, "; "))
	}

	var targetRuns []schema.TargetResult
	for t, rs := range targetBuckets {
		targetRuns = append(targetRuns, schema.TargetResult{Target: t, Results: aggregate.Results(rs)})
	}

	agg := aggregate.Results(resultSet)
	pattern, blockSize, datasetSize, durationSec, rampSec, qd, threads, rwMix, directIO := opts.Profile.ToWorkload()
	engineName := opts.Profile.Engine
	if opts.Mock {
		engineName = "mock"
	}

	primaryTarget := targets[0]
	if len(targets) > 1 {
		primaryTarget = strings.Join(targets, ",")
	}

	run, err := s.saveRun(opts, runID, started, durationSec, rampSec, engineName, validation, hw,
		pattern, blockSize, datasetSize, qd, threads, rwMix, directIO, agg, nil, clientRuns, targetRuns, plan.Mode, primaryTarget)
	if err != nil {
		return nil, err
	}
	if len(errs) > 0 {
		run.Validation.Warnings = append(run.Validation.Warnings, schema.Warning{
			Rule:     "partial_assignment_failure",
			Message:  strings.Join(errs, "; "),
			Severity: "warning",
		})
		_ = s.Store.Save(run)
	}
	return run, nil
}

func (s *Service) runAssignment(ctx context.Context, opts RunOptions, a topology.Assignment) (schema.Results, error) {
	if a.Client == "" {
		runner := engine.ForProfile(opts.Profile, opts.Mock)
		results, _, err := runner.Run(ctx, engine.RunInput{
			Profile: opts.Profile,
			Target:  a.Target,
			Mock:    opts.Mock,
			WorkDir: filepath.Join(opts.WorkDir, sanitizePath(a.Target)),
		})
		if err != nil {
			return schema.Results{}, err
		}
		return *results, nil
	}

	client := remote.NewClient(a.Client)
	if _, err := client.Health(ctx); err != nil {
		return schema.Results{}, fmt.Errorf("health check: %w", err)
	}
	run, err := client.Run(ctx, opts.Profile, a.Target, opts.Mock, true, opts.CheckHardware, opts.CacheBytes)
	if err != nil {
		return schema.Results{}, err
	}
	return run.Results, nil
}

func assignmentLabel(a topology.Assignment) string {
	if a.Client == "" {
		return "local→" + a.Target
	}
	return a.Client + "→" + a.Target
}

func sanitizePath(s string) string {
	s = strings.NewReplacer("/", "_", ":", "_", "@", "_", ",", "_").Replace(s)
	if s == "" {
		return "default"
	}
	return s
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
	targets []schema.TargetResult,
	topologyMode string,
	primaryTarget string,
) (*schema.RunResult, error) {
	completed := time.Now().UTC()
	steady := started.Add(time.Duration(rampSec) * time.Second)

	targetType := "block"
	if opts.Profile.Layer == "object" || strings.HasPrefix(opts.Profile.Layer, "vm-object") {
		targetType = "object"
	}

	meta := map[string]string{}
	if len(opts.Clients) > 0 {
		meta["clients"] = strings.Join(opts.Clients, ",")
	}
	allTargets := topology.MergeTargets(opts.Target, opts.Targets)
	if len(allTargets) > 0 {
		meta["targets"] = strings.Join(allTargets, ",")
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
		Topology:      topologyMode,
		Target: schema.Target{
			Type:     targetType,
			Device:   primaryTarget,
			Endpoint: primaryTarget,
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
		Targets:   targets,
	}

	if err := s.Store.Save(run); err != nil {
		return nil, err
	}
	_ = inventory.Save(s.Store, hw)
	metrics.RecordRun(run)
	return run, nil
}

func (s *Service) CheckRegression(run *schema.RunResult) []baseline.Alert {
	key := baseline.TargetKey(run)
	rec, err := s.Store.GetBaseline(run.Profile, key)
	if err == nil {
		baseRun, err := s.Store.Get(rec.RunID)
		if err == nil {
			return baseline.Check(run, baseRun, baseline.DefaultIOPSDegradePct, baseline.DefaultLatencyDegradePct)
		}
	}

	historyRuns, err := s.Store.ListSince(baseline.RollingSince(baseline.DefaultRollingDays), 500)
	if err != nil || len(historyRuns) == 0 {
		return nil
	}
	history := make([]*schema.RunResult, len(historyRuns))
	for i := range historyRuns {
		history[i] = &historyRuns[i]
	}
	ref := baseline.ReferenceFromHistory(history, run.Profile, key, run.RunID)
	return baseline.CheckAgainstReference(run, ref, baseline.DefaultIOPSDegradePct, baseline.DefaultLatencyDegradePct)
}

func (s *Service) SetBaselineFromRun(runID string) (*store.BaselineRecord, error) {
	run, err := s.Store.Get(runID)
	if err != nil {
		return nil, err
	}
	key := baseline.TargetKey(run)
	if err := s.Store.SetBaseline(run.Profile, key, run.RunID); err != nil {
		return nil, err
	}
	return s.Store.GetBaseline(run.Profile, key)
}
