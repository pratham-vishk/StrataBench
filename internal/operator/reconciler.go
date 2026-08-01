package operator

import (
	"context"
	"fmt"
	"log"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"

	"github.com/pratham-vishk/stratabench/internal/manifest"
	"github.com/pratham-vishk/stratabench/internal/orchestrator"
	"github.com/pratham-vishk/stratabench/internal/paths"
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
	svc    *orchestrator.Service
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
	svc, err := orchestrator.NewService(paths.DataDir())
	if err != nil {
		return nil, err
	}
	return &Reconciler{client: client, svc: svc, cfg: cfg}, nil
}

func (r *Reconciler) Close() error { return r.svc.Close() }

func (r *Reconciler) Run(ctx context.Context) error {
	log.Printf("operator watching benchmarks.%s in namespace %s", benchmarkGVR.Version, r.cfg.Namespace)
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
	phase, _, _ := unstructured.NestedString(obj.Object, "status", "phase")
	if phase == manifest.PhaseCompleted {
		return nil
	}

	name := obj.GetName()
	if err := r.patchStatus(ctx, name, manifest.BenchmarkStatus{Phase: manifest.PhaseRunning, Message: "starting benchmark"}); err != nil {
		return err
	}

	b, err := fromUnstructured(obj)
	if err != nil {
		_ = r.patchStatus(ctx, name, manifest.BenchmarkStatus{Phase: manifest.PhaseFailed, Message: err.Error()})
		return err
	}

	result, err := manifest.Apply(ctx, r.svc, b)
	if err != nil {
		_ = r.patchStatus(ctx, name, manifest.BenchmarkStatus{Phase: manifest.PhaseFailed, Message: err.Error()})
		return err
	}

	return r.patchStatus(ctx, name, manifest.BenchmarkStatus{
		Phase:   manifest.PhaseCompleted,
		RunID:   result.RunID,
		Message: fmt.Sprintf("profile=%s", result.Profile),
	})
}

func (r *Reconciler) patchStatus(ctx context.Context, name string, status manifest.BenchmarkStatus) error {
	obj, err := r.client.Resource(benchmarkGVR).Namespace(r.cfg.Namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
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
			Mock:          nestedBool(spec, "mock"),
			SkipValidate:  nestedBool(spec, "skipValidate"),
			CheckBaseline: nestedBool(spec, "checkBaseline"),
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
	if b.Spec.Profile == "" && b.Spec.Intent == "" {
		return nil, fmt.Errorf("spec.profile or spec.intent required")
	}
	if b.Spec.Target == "" && b.Spec.Intent == "" {
		return nil, fmt.Errorf("spec.target required")
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
