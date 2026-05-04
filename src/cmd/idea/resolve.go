package main

import "github.com/sahil87/idea/internal/idea"

func resolveFile() (string, error) {
	var repoRoot string
	var err error
	if mainFlag {
		repoRoot, err = idea.MainRepoRoot()
	} else {
		repoRoot, err = idea.WorktreeRoot()
	}
	if err != nil {
		return "", err
	}
	return idea.ResolveFilePath(repoRoot, fileFlag), nil
}
