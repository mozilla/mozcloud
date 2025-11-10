// Package diff provides functions for comparing rendered manifests
package diff

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
	"github.com/mozilla/mozcloud/tools/render-diff/internal/helm"
	"github.com/mozilla/mozcloud/tools/render-diff/internal/kustomize"
)

// ANSI codes for diff colors
const (
	colorRed   = "\033[31m"
	colorGreen = "\033[32m"
	colorCyan  = "\033[36m"
	colorReset = "\033[0m"
)

// GetRepoRoot finds the top-level directory of the current git repository.
func GetRepoRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to find git repo root: %w. Make sure you are running this inside a git repository. Output: %s", err, string(output))
	}
	return strings.TrimSpace(string(output)), nil
}

// RenderManifests will render a Helm Chart or build a Kustomization
// and return the rendered manifests as a string
func RenderManifests(path string, values []string, debug bool, update bool) (string, error) {
	var renderedManifests string
	var err error

	if helm.IsHelmChart(path) {
		renderedManifests, err = helm.RenderChart(path, "release", values, debug, update)
		if err != nil {
			return "", fmt.Errorf("failed to render target Chart: '%s'", err)
		}
		return renderedManifests, nil
	} else if kustomize.IsKustomize(path) {
		renderedManifests, err = kustomize.RenderKustomization(path)
		if err != nil {
			return "", fmt.Errorf("failed to build target Kustomization: '%s'", err)
		}
		return renderedManifests, nil
	}

	return "", fmt.Errorf("path: %s is not a valid Helm Chart or Kustomization", path)
}

// createDiff generates a unified diff string between two text inputs.
func CreateDiff(a, b string, fromName, toName string) string {
	edits := myers.ComputeEdits(span.URI(fromName), a, b)
	diff := gotextdiff.ToUnified(fromName, toName, a, edits)

	return fmt.Sprint(diff)
}

// colorizeDiff adds simple ANSI colors to a diff string.
// We want to see this output in a terminal or as a comment on a PR
// Fast readability is important
func ColorizeDiff(diff string) string {
	var coloredDiff strings.Builder
	lines := strings.Split(diff, "\n")

	for _, line := range lines {
		switch {
		// Standard unified diff lines
		case strings.HasPrefix(line, "+"):
			coloredDiff.WriteString(colorGreen + line + colorReset + "\n")
		case strings.HasPrefix(line, "-"):
			coloredDiff.WriteString(colorRed + line + colorReset + "\n")
		case strings.HasPrefix(line, "@@"):
			coloredDiff.WriteString(colorCyan + line + colorReset + "\n")
		// --- and +++ are headers, no special color
		case strings.HasPrefix(line, "---"), strings.HasPrefix(line, "+++"):
			coloredDiff.WriteString(line + "\n")
		// Default (context lines, start with a space)
		default:
			coloredDiff.WriteString(line + "\n")
		}
	}

	return coloredDiff.String()
}
