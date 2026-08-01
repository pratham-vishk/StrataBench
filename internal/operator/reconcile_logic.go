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

func decideReconcile(phase, storedHash, currentHash string, jobExists bool) reconcileAction {
	specChanged := storedHash != "" && storedHash != currentHash
	if phase == manifest.PhaseCompleted && !specChanged {
		return actionSkip
	}
	if specChanged && jobExists {
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
