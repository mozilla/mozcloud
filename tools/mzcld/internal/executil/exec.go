// Package executil provides lightweight helpers for running external commands.
package executil

import (
	"bytes"
	"context"
	"os/exec"
	"strings"

	"github.com/mozilla/mozcloud/tools/mzcld/internal/ui"
)

// LookPath reports whether name is available in PATH.
func LookPath(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// Output runs name with args and returns trimmed stdout.
// Stderr is discarded. Returns an error if the command fails.
func Output(ctx context.Context, name string, args ...string) (string, error) {
	ui.Debug("exec: " + name + " " + strings.Join(args, " "))
	cmd := exec.CommandContext(ctx, name, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	return strings.TrimSpace(out.String()), err
}

// Combined runs name with args and returns trimmed combined stdout+stderr.
func Combined(ctx context.Context, name string, args ...string) (string, error) {
	ui.Debug("exec: " + name + " " + strings.Join(args, " "))
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}
