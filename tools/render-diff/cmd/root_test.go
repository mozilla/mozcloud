package cmd

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"
)

// resetFlags resets all package-level flag variables to their defaults.
// This is for running tests sequentially, as Cobra's flags
// are package-level variables and persist between test runs.
func resetFlags() {
	// Reset to default values from init()
	renderPathFlag = "."
	gitRefFlag = "HEAD"
	valuesFlag = []string{}
	debugFlag = false

	// Reset state variables set by PreRunE
	repoRoot = ""
	fullRef = ""
}

// executeCommand is a helper to run the rootCmd with a given context and args.
// It captures and returns stdout, stderr, and any error from ExecuteContext.
func executeCommand(ctx context.Context, args ...string) (stdout, stderr string, err error) {
	// Reset all global state before this run
	resetFlags()

	// Set the command-line arguments for this test
	rootCmd.SetArgs(args)

	// Capture stdout
	oldOut := os.Stdout
	rOut, wOut, _ := os.Pipe()
	os.Stdout = wOut

	// Capture stderr
	oldErr := os.Stderr
	rErr, wErr, _ := os.Pipe()
	os.Stderr = wErr

	// Defer cleanup to restore stdout/stderr
	defer func() {
		os.Stdout = oldOut
		os.Stderr = oldErr
	}()

	// Run the command
	// We call ExecuteContext directly, which returns an error instead of os.Exit(1)
	err = rootCmd.ExecuteContext(ctx)

	// Close the writers and read the output
	_ = wOut.Close()
	_ = wErr.Close()
	stdoutBytes, _ := io.ReadAll(rOut)
	stderrBytes, _ := io.ReadAll(rErr)

	return string(stdoutBytes), string(stderrBytes), err
}

func TestRootCmd(t *testing.T) {

	t.Run("Success path with Helm chart", func(t *testing.T) {
		path := "./examples/helm/helloworld"
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Skipf("Skipping test, example path not found: %s", path)
		}

		ctx := context.Background()
		stdout, stderr, err := executeCommand(ctx, "--path", path, "--ref", "HEAD")

		if err != nil {
			t.Fatalf("Command failed unexpectedly: %v\nStderr: %s", err, stderr)
		}

		if !strings.Contains(stderr, "Starting diff against git ref 'HEAD'") {
			t.Errorf("Expected log message in stderr, got: %s", stderr)
		}

		if !strings.Contains(stdout, "--- Diff ---") {
			t.Errorf("Expected diff output in stdout, got: %s", stdout)
		}
	})

	t.Run("PersistentPreRunE failure (invalid ref)", func(t *testing.T) {
		ctx := context.Background()
		_, _, err := executeCommand(ctx, "--ref", "this-ref-does-not-exist-12345")

		if err == nil {
			t.Fatal("Command succeeded, but expected an error for invalid ref")
		}

		if !strings.Contains(err.Error(), "invalid or non-existent ref") {
			t.Errorf("Expected error message about 'invalid ref', got: %v", err)
		}
	})

	t.Run("RunE failure (path outside repo)", func(t *testing.T) {
		// We use a path that is guaranteed to be outside the repo
		path := os.TempDir()

		ctx := context.Background()
		_, _, err := executeCommand(ctx, "--path", path)

		if err == nil {
			t.Fatalf("Command succeeded, but expected an error for path outside repo. Path: %s", path)
		}

		if !strings.Contains(err.Error(), "outside the git repository root") {
			t.Errorf("Expected error message about 'outside...root', got: %v", err)
		}
	})
}
