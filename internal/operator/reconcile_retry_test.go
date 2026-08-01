package operator

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestRetryState(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"metadata": map[string]any{
			"annotations": map[string]any{
				annotationSpecHash:     "abc",
				annotationRetry:        "2",
				annotationRetryApplied: "1",
			},
		},
	}}
	hash, token, requested := retryState(obj)
	if hash != "abc" || token != "2" || !requested {
		t.Fatalf("hash=%q token=%q requested=%v", hash, token, requested)
	}
	obj.Object["metadata"].(map[string]any)["annotations"].(map[string]any)[annotationRetryApplied] = "2"
	_, _, requested = retryState(obj)
	if requested {
		t.Fatal("expected retry not requested when already applied")
	}
}
