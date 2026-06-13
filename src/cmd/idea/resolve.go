package main

import "github.com/sahil87/idea/internal/idea"

// resolveFile wires the persistent flags into the resolution precedence, which
// lives in internal/idea (Constitution IV — cmd/ holds only flag wiring). The
// --system/--main conflict and the out-of-git fallback are decided there.
func resolveFile() (string, error) {
	return idea.ResolveBacklogPath(systemFlag, mainFlag, fileFlag)
}
