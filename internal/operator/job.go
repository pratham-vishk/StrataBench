package operator

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/pratham-vishk/stratabench/internal/manifest"
)

var (
	jobGVR = schema.GroupVersionResource{Group: "batch", Version: "v1", Resource: "jobs"}
	cmGVR  = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}
)

func jobName(benchmark string) string {
	name := "bench-" + benchmark
	if len(name) > 63 {
		name = name[:63]
	}
	return strings.TrimRight(name, "-")
}

func configMapName(benchmark string) string {
	name := "bench-cm-" + benchmark
	if len(name) > 63 {
		name = name[:63]
	}
	return strings.TrimRight(name, "-")
}

func statusResultPath(benchmark string) string {
	return fmt.Sprintf("/data/status/%s.json", benchmark)
}

func specHash(b *manifest.Benchmark) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%s|%v|%v|%t|%t",
		b.Spec.Profile, b.Spec.Target, b.Spec.Intent,
		b.Spec.Clients, b.Spec.Targets, b.Spec.Mock, b.Spec.SkipValidate)))
	return hex.EncodeToString(h[:8])
}

func jobImage() string {
	if v := os.Getenv("STRATABENCH_JOB_IMAGE"); v != "" {
		return v
	}
	return "ghcr.io/pratham-vishk/stratabench:latest"
}

func dataPVC() string {
	if v := os.Getenv("STRATABENCH_DATA_PVC"); v != "" {
		return v
	}
	return "stratabench-data"
}

func buildConfigMap(b *manifest.Benchmark) (*corev1.ConfigMap, error) {
	yaml, err := b.ToYAML()
	if err != nil {
		return nil, err
	}
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName(b.Metadata.Name),
			Namespace: b.Metadata.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "stratabench-operator",
				"stratabench.io/benchmark":     b.Metadata.Name,
			},
		},
		Data: map[string]string{
			"benchmark.yaml": string(yaml),
		},
	}, nil
}

func buildJob(b *manifest.Benchmark) (*batchv1.Job, error) {
	cmName := configMapName(b.Metadata.Name)
	statusOut := statusResultPath(b.Metadata.Name)
	backoff := int32(0)
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName(b.Metadata.Name),
			Namespace: b.Metadata.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "stratabench-operator",
				"stratabench.io/benchmark":     b.Metadata.Name,
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: &backoff,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"stratabench.io/benchmark": b.Metadata.Name,
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:  "stratabench",
							Image: jobImage(),
							Command: []string{"/usr/local/bin/stratabench"},
							Args: []string{
								"apply", "/manifests/benchmark.yaml",
								"--status-out", statusOut,
							},
							Env: []corev1.EnvVar{
								{Name: "STRATABENCH_ROOT", Value: "/etc/stratabench"},
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "manifest", MountPath: "/manifests"},
								{Name: "data", MountPath: "/data"},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "manifest",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{Name: cmName},
								},
							},
						},
						{
							Name: "data",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: dataPVC(),
								},
							},
						},
					},
				},
			},
		},
	}, nil
}

func toUnstructured(obj runtime.Object) (*unstructured.Unstructured, error) {
	u := &unstructured.Unstructured{}
	u.Object, _ = runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	return u, nil
}

func jobFailed(job *unstructured.Unstructured) bool {
	conditions, found, _ := unstructured.NestedSlice(job.Object, "status", "conditions")
	if !found {
		return false
	}
	for _, c := range conditions {
		m, _ := c.(map[string]any)
		if m["type"] == "Failed" && m["status"] == "True" {
			return true
		}
	}
	return false
}

func jobSucceeded(job *unstructured.Unstructured) bool {
	succeeded, found, _ := unstructured.NestedInt64(job.Object, "status", "succeeded")
	return found && succeeded > 0
}
