package operator

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"

	"github.com/pratham-vishk/stratabench/internal/manifest"
)

var benchmarkGVR = schema.GroupVersionResource{
	Group:    "stratabench.io",
	Version:  "v1alpha1",
	Resource: "benchmarks",
}

type Config struct {
	Namespace   string
	ResyncEvery time.Duration
}

type Reconciler struct {
	client dynamic.Interface
	cfg    Config
}

func New(cfg Config) (*Reconciler, error) {
	if cfg.Namespace == "" {
		cfg.Namespace = "stratabench"
	}
	if cfg.ResyncEvery <= 0 {
		cfg.ResyncEvery = 30 * time.Second
	}
	restCfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("in-cluster config: %w (run inside Kubernetes)", err)
	}
	client, err := dynamic.NewForConfig(restCfg)
	if err != nil {
		return nil, err
	}
	return &Reconciler{client: client, cfg: cfg}, nil
}

func (r *Reconciler) Close() error { return nil }

func (r *Reconciler) Run(ctx context.Context) error {
	log.Printf("operator watching benchmarks.%s in namespace %s (Job mode)", benchmarkGVR.Version, r.cfg.Namespace)
	ticker := time.NewTicker(r.cfg.ResyncEvery)
	defer ticker.Stop()

	for {
		if err := r.reconcileAll(ctx); err != nil {
			log.Printf("reconcile error: %v", err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (r *Reconciler) reconcileAll(ctx context.Context) error {
	list, err := r.client.Resource(benchmarkGVR).Namespace(r.cfg.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, item := range list.Items {
		if err := r.reconcileOne(ctx, &item); err != nil {
			log.Printf("benchmark %s: %v", item.GetName(), err)
		}
	}
	return nil
}

func (r *Reconciler) reconcileOne(ctx context.Context, obj *unstructured.Unstructured) error {
	name := obj.GetName()
	b, err := fromUnstructured(obj)
	if err != nil {
		_ = r.patchStatus(ctx, name, "", manifest.BenchmarkStatus{Phase: manifest.PhaseFailed, Message: err.Error()})
		return err
	}
	b.Metadata.Namespace = r.cfg.Namespace
	hash := specHash(b)

	phase, _, _ := unstructured.NestedString(obj.Object, "status", "phase")
	if phase == manifest.PhaseCompleted {
		if obj.GetAnnotations()["stratabench.io/spec-hash"] == hash {
			return nil
		}
	}

	job, err := r.client.Resource(jobGVR).Namespace(r.cfg.Namespace).Get(ctx, jobName(name), metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return r.ensureJob(ctx, b, hash)
	}
	if err != nil {
		return err
	}

	if jobSucceeded(job) {
		result, readErr := manifest.ReadApplyResult(filepath.Join(statusDir(), name+".json"))
		st := manifest.BenchmarkStatus{Phase: manifest.PhaseCompleted, Message: "job succeeded"}
		if readErr == nil && result != nil {
			st.RunID = result.RunID
			st.Message = fmt.Sprintf("profile=%s", result.Profile)
		} else if readErr != nil {
			st.Message = "job succeeded (status file pending)"
		}
		return r.patchStatus(ctx, name, hash, st)
	}
	if jobFailed(job) {
		return r.patchStatus(ctx, name, hash, manifest.BenchmarkStatus{
			Phase:   manifest.PhaseFailed,
			Message: "benchmark job failed",
		})
	}
	if phase != manifest.PhaseRunning {
		return r.patchStatus(ctx, name, hash, manifest.BenchmarkStatus{
			Phase:   manifest.PhaseRunning,
			Message: "job running",
		})
	}
	return nil
}

func (r *Reconciler) ensureJob(ctx context.Context, b *manifest.Benchmark, hash string) error {
	cm, err := buildConfigMap(b)
	if err != nil {
		return err
	}
	cmU, err := toUnstructured(cm)
	if err != nil {
		return err
	}
	_, err = r.client.Resource(cmGVR).Namespace(r.cfg.Namespace).Create(ctx, cmU, metav1.CreateOptions{})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("create configmap: %w", err)
	}

	job, err := buildJob(b)
	if err != nil {
		return err
	}
	jobU, err := toUnstructured(job)
	if err != nil {
		return err
	}
	if _, err := r.client.Resource(jobGVR).Namespace(r.cfg.Namespace).Create(ctx, jobU, metav1.CreateOptions{}); err != nil {
		return fmt.Errorf("create job: %w", err)
	}
	return r.patchStatus(ctx, b.Metadata.Name, hash, manifest.BenchmarkStatus{
		Phase:   manifest.PhaseRunning,
		Message: "job created",
	})
}

func statusDir() string {
	base := os.Getenv("STRATABENCH_DATA_DIR")
	if base == "" {
		base = "/data"
	}
	return filepath.Join(base, "status")
}

func (r *Reconciler) patchStatus(ctx context.Context, name, specHash string, status manifest.BenchmarkStatus) error {
	obj, err := r.client.Resource(benchmarkGVR).Namespace(r.cfg.Namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if specHash != "" {
		ann := obj.GetAnnotations()
		if ann == nil {
			ann = map[string]string{}
		}
		ann["stratabench.io/spec-hash"] = specHash
		obj.SetAnnotations(ann)
		if _, err := r.client.Resource(benchmarkGVR).Namespace(r.cfg.Namespace).Update(ctx, obj, metav1.UpdateOptions{}); err != nil {
			return err
		}
		obj, err = r.client.Resource(benchmarkGVR).Namespace(r.cfg.Namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}
	}
	if err := unstructured.SetNestedField(obj.Object, status.Phase, "status", "phase"); err != nil {
		return err
	}
	if err := unstructured.SetNestedField(obj.Object, status.RunID, "status", "runId"); err != nil {
		return err
	}
	if err := unstructured.SetNestedField(obj.Object, status.Message, "status", "message"); err != nil {
		return err
	}
	_, err = r.client.Resource(benchmarkGVR).Namespace(r.cfg.Namespace).UpdateStatus(ctx, obj, metav1.UpdateOptions{})
	return err
}

func fromUnstructured(obj *unstructured.Unstructured) (*manifest.Benchmark, error) {
	spec, ok, err := unstructured.NestedMap(obj.Object, "spec")
	if err != nil || !ok {
		return nil, fmt.Errorf("missing spec")
	}
	b := &manifest.Benchmark{
		Metadata: manifest.Metadata{Name: obj.GetName(), Namespace: obj.GetNamespace()},
		Spec: manifest.BenchmarkSpec{
			Profile:       nestedString(spec, "profile"),
			Target:        nestedString(spec, "target"),
			Topology:      nestedString(spec, "topology"),
			Mock:          nestedBool(spec, "mock"),
			SkipValidate:  nestedBool(spec, "skipValidate"),
			CheckBaseline: nestedBool(spec, "checkBaseline"),
			CheckHardware: nestedBoolPtr(spec, "checkHardware"),
			Intent:        nestedString(spec, "intent"),
			UseOllama:     nestedBool(spec, "useOllama"),
		},
	}
	if clients, ok := spec["clients"].([]any); ok {
		for _, c := range clients {
			if s, ok := c.(string); ok {
				b.Spec.Clients = append(b.Spec.Clients, s)
			}
		}
	}
	if targets, ok := spec["targets"].([]any); ok {
		for _, t := range targets {
			if s, ok := t.(string); ok {
				b.Spec.Targets = append(b.Spec.Targets, s)
			}
		}
	}
	if b.Spec.Profile == "" && b.Spec.Intent == "" {
		return nil, fmt.Errorf("spec.profile or spec.intent required")
	}
	if b.Spec.Target == "" && len(b.Spec.Targets) == 0 && b.Spec.Intent == "" {
		return nil, fmt.Errorf("spec.target or spec.targets required")
	}
	return b, nil
}

func nestedString(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

func nestedBool(m map[string]any, key string) bool {
	v, ok := m[key]
	if !ok {
		return false
	}
	b, _ := v.(bool)
	return b
}

func nestedBoolPtr(m map[string]any, key string) *bool {
	v, ok := m[key]
	if !ok {
		return nil
	}
	b, ok := v.(bool)
	if !ok {
		return nil
	}
	return &b
}
