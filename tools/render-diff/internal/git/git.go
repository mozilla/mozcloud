// Package git provides functions for setting up a temporary git work tree
package git

import (
	"fmt"
	"log"
	"os"
	"os/exec"
)

func SetupWorkTree(repoRoot, gitRef string) (string, func(), error) {

	// Fetch from all remotes
	fetchCmd := exec.Command("git", "fetch", "--all")
	fetchCmd.Dir = repoRoot
	if output, err := fetchCmd.CombinedOutput(); err != nil {
		return "", nil, fmt.Errorf("failed to run 'git fetch --all': %w\nOutput: %s", err, string(output))
	}

	// Set up a Git Worktree for gitref
	tempDir, err := os.MkdirTemp("", "diff-ref-")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp directory: %v", err)
	}

	// Combined worktree and tempdir cleanup
	// Returning this function to defer in rootCmd
	cleanup := func() {
		// Using --force to avoid errors if dir is already partially cleaned
		cleanupCmd := exec.Command("git", "worktree", "remove", "--force", tempDir)
		cleanupCmd.Dir = repoRoot
		if output, err := cleanupCmd.CombinedOutput(); err != nil {
			log.Printf("Warning: failed to run 'git worktree remove'. Manual cleanup may be required. Error: %v, Output: %s", err, string(output))
		}
		if err := os.RemoveAll(tempDir); err != nil {
			fmt.Printf("error removing temporary directory %s: %v\n", tempDir, err)
		}
	}

	// Create the worktree
	// Using -d to allow checking out a branch that is already checked out (like 'main')
	addCmd := exec.Command("git", "worktree", "add", "-d", tempDir, gitRef)
	addCmd.Dir = repoRoot
	if output, err := addCmd.CombinedOutput(); err != nil {
		return "", nil, fmt.Errorf("failed to create worktree for '%s': %v\nOutput: %s", gitRef, err, string(output))
	}

	return tempDir, cleanup, nil
}
