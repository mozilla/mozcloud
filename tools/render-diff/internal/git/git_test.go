package git

import (
	"os"
	"testing"
)

func TestSetupWorkTree(t *testing.T) {
	repoRoot, _ := GetRepoRoot()

	t.Run("Success with valid ref", func(t *testing.T) {
		gitRef := "HEAD"

		tempDir, cleanup, err := SetupWorkTree(repoRoot, gitRef)
		if err != nil {
			t.Fatalf("SetupWorkTree() with valid ref failed: %v", err)
		}

		if cleanup == nil {
			t.Fatal("SetupWorkTree() returned a nil cleanup function")
		}

		if tempDir == "" {
			t.Fatal("SetupWorkTree() returned an empty tempDir path")
		}

		if _, err := os.Stat(tempDir); os.IsNotExist(err) {
			t.Fatalf("SetupWorkTree() reported success, but tempDir does not exist: %s", tempDir)
		}

		cleanup()
		if _, err := os.Stat(tempDir); !os.IsNotExist(err) {
			t.Errorf("Cleanup function failed: tempDir still exists: %s", tempDir)
		}
	})

	t.Run("Failure with invalid ref", func(t *testing.T) {
		gitRef := "this-ref-does-not-exist-12345"

		tempDir, cleanup, err := SetupWorkTree(repoRoot, gitRef)

		if err == nil {
			t.Fatal("SetupWorkTree() with invalid ref succeeded, but expected an error")
		}

		if cleanup != nil {
			t.Error("SetupWorkTree() returned a non-nil cleanup function on failure")
			cleanup()
		}

		if tempDir != "" {
			t.Errorf("SetupWorkTree() returned a non-empty tempDir on failure: %s", tempDir)
		}
	})
}
