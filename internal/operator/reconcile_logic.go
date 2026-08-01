package operator

import (
	"github.com/pratham-vishk/stratabench/internal/manifest"
)

type reconcileAction int

const (
	actionSkip reconcileAction = iota
	actionEnsureJob
	actionTeardownAndRerun
	actionObserveJob
)

const (
	annotationSpecHash      = "stratabench.io/spec-hash"
	annotationRetry         = "stratabench.io/retry"
	annotationRetryApplied  = "stratabench.io/retry-applied"
)

func decideReconcile(phase, storedHash, currentHash string, jobExists, retryRequested bool) reconcileAction {
	specChanged := storedHash != "" && storedHash != currentHash
	if phase == manifest.PhaseCompleted && !specChanged && !retryRequested {
		return actionSkip
	}
	if (specChanged || retryRequested) && jobExists {
		return actionTeardownAndRerun
	}
	if !jobExists {
		return actionEnsureJob
	}
	return actionObserveJob
}

func specChanged(storedHash, currentHash string) bool {
	return storedHash != "" && storedHash != currentHash
}
