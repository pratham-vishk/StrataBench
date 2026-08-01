package operator

import (
	"testing"

	"github.com/pratham-vishk/stratabench/internal/manifest"
)

func TestDecideReconcile(t *testing.T) {
	const oldHash = "abc123"
	const newHash = "def456"

	tests := []struct {
		name      string
		phase     string
		stored    string
		current   string
		jobExists bool
		retry     bool
		want      reconcileAction
	}{
		{
			name:      "completed unchanged skips",
			phase:     manifest.PhaseCompleted,
			stored:    oldHash,
			current:   oldHash,
			jobExists: true,
			retry:     false,
			want:      actionSkip,
		},
		{
			name:      "completed retry reruns",
			phase:     manifest.PhaseCompleted,
			stored:    oldHash,
			current:   oldHash,
			jobExists: true,
			retry:     true,
			want:      actionTeardownAndRerun,
		},
		{
			name:      "completed spec change reruns",
			phase:     manifest.PhaseCompleted,
			stored:    oldHash,
			current:   newHash,
			jobExists: true,
			retry:     false,
			want:      actionTeardownAndRerun,
		},
		{
			name:      "failed spec change reruns",
			phase:     manifest.PhaseFailed,
			stored:    oldHash,
			current:   newHash,
			jobExists: true,
			retry:     false,
			want:      actionTeardownAndRerun,
		},
		{
			name:      "pending no job ensures",
			phase:     manifest.PhasePending,
			stored:    "",
			current:   newHash,
			jobExists: false,
			retry:     false,
			want:      actionEnsureJob,
		},
		{
			name:      "running observes",
			phase:     manifest.PhaseRunning,
			stored:    oldHash,
			current:   oldHash,
			jobExists: true,
			retry:     false,
			want:      actionObserveJob,
		},
		{
			name:      "completed missing job after teardown ensures",
			phase:     manifest.PhaseCompleted,
			stored:    oldHash,
			current:   newHash,
			jobExists: false,
			retry:     false,
			want:      actionEnsureJob,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := decideReconcile(tc.phase, tc.stored, tc.current, tc.jobExists, tc.retry)
			if got != tc.want {
				t.Fatalf("got %d want %d", got, tc.want)
			}
		})
	}
}

func TestSpecChanged(t *testing.T) {
	if !specChanged("a", "b") {
		t.Fatal("expected change")
	}
	if specChanged("", "b") {
		t.Fatal("first run is not a change")
	}
	if specChanged("a", "a") {
		t.Fatal("same hash is not a change")
	}
}
