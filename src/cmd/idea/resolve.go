package main

import (
	"errors"

	"github.com/sahil87/idea/internal/idea"
)

// resolveFile wires the persistent flags into the resolution precedence, which
// lives in internal/idea (Constitution IV — cmd/ holds only flag wiring). The
// --system/--main conflict and the out-of-git fallback are decided there.
//
// The --system/--main conflict is a malformed invocation, so it is classified
// as a usage error (exit 2) here in cmd/ — internal/idea only names the
// condition via idea.ErrConflictingTargets; exit-code policy stays in cmd/.
func resolveFile() (string, error) {
	path, err := idea.ResolveBacklogPath(systemFlag, mainFlag, fileFlag)
	if errors.Is(err, idea.ErrConflictingTargets) {
		return "", &usageError{err}
	}
	return path, err
}
