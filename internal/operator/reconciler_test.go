package operator

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestFromUnstructured(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"metadata": map[string]any{"name": "test-bench", "namespace": "stratabench"},
		"spec": map[string]any{
			"profile":       "ssd-random-4k",
			"target":        "/dev/nvme0n1",
			"mock":          true,
			"skipValidate":  true,
			"checkBaseline": false,
			"clients":       []any{"10.0.0.1:7777", "10.0.0.2:7777"},
		},
	}}
	b, err := fromUnstructured(obj)
	if err != nil {
		t.Fatal(err)
	}
	if b.Spec.Profile != "ssd-random-4k" {
		t.Fatalf("profile: got %q", b.Spec.Profile)
	}
	if len(b.Spec.Clients) != 2 {
		t.Fatalf("clients: got %d", len(b.Spec.Clients))
	}
	if !b.Spec.Mock {
		t.Fatal("expected mock=true")
	}
}

func TestFromUnstructuredIntent(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"metadata": map[string]any{"name": "agent-bench"},
		"spec": map[string]any{
			"intent":    "nvme oltp",
			"target":    "/dev/nvme0n1",
			"useOllama": true,
		},
	}}
	b, err := fromUnstructured(obj)
	if err != nil {
		t.Fatal(err)
	}
	if b.Spec.Intent != "nvme oltp" || !b.Spec.UseOllama {
		t.Fatalf("unexpected spec: %+v", b.Spec)
	}
}
