// Package diff provides functions for comparing rendered manifests
// It includes our simple CreateDiff functions as well as a semantic diff
// using the homeport/dyff module for more advanced diff features
package diff

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
	"github.com/homeport/dyff/pkg/dyff"
	"github.com/mozilla/mozcloud/tools/render-diff/internal/helm"
	"github.com/mozilla/mozcloud/tools/render-diff/internal/kustomize"

	"github.com/gonvenience/bunt"
	"github.com/gonvenience/ytbx"
	"gopkg.in/yaml.v3"
)

// ANSI codes for simple diff colors
const (
	colorRed   = "\033[31m"
	colorGreen = "\033[32m"
	colorCyan  = "\033[36m"
	colorReset = "\033[0m"
)

// RenderManifests will render a Helm Chart or build a Kustomization
// and return the rendered manifests as a string
func RenderManifests(path string, values []string, debug bool, update bool, release string) (string, error) {
	var renderedManifests string
	var err error
	releaseName := release

	if helm.IsHelmChart(path) {
		// Set releaseName equal to chartName if --release-name is not supplied
		if releaseName == "" {
			chartName, err := helm.GetChartName(path, debug)
			if err != nil {
				releaseName = "release"
			} else {
				releaseName = chartName
			}
		}

		renderedManifests, err = helm.RenderChart(path, releaseName, values, debug, update)
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

// This is the original simple diff configuration
// createDiff generates a unified diff string between two text inputs.
func CreateDiff(a, b string, fromName, toName string) string {
	edits := myers.ComputeEdits(span.URI(fromName), a, b)
	diff := gotextdiff.ToUnified(fromName, toName, a, edits)

	return fmt.Sprint(diff)
}

// colorizeDiff adds simple ANSI colors to a diff string.
// We want to see this output in a terminal or as a comment on a PR
// Fast readability is important
func ColorizeDiff(diff string, plain bool) string {
	if plain {
		return diff
	}
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

// This is more complex but k8s object aware diff engine
// it is better suited for larger scale changes to a k8s resources
func CreateSemanticDiff(targetRender, localRender, fromName, toName string, plain bool) (*dyff.HumanReport, error) {
	// dyff is using bunt for text colouring
	// plain flag & writing to a file turns colours off
	// defaults to ON or AUTO if we get an error
	fi, err := os.Stdout.Stat()
	switch {
	case plain:
		bunt.SetColorSettings(bunt.OFF, bunt.OFF)
	case fi.Mode().IsRegular():
		bunt.SetColorSettings(bunt.OFF, bunt.OFF)
	case err != nil:
		bunt.SetColorSettings(bunt.AUTO, bunt.AUTO)
	default:
		bunt.SetColorSettings(bunt.ON, bunt.ON)
	}

	localRenderFile, err := createInputFileFromString(localRender, toName)
	if err != nil {
		return nil, fmt.Errorf("failed to parse local render for semantic diff: %w", err)
	}

	targetRenderFile, err := createInputFileFromString(targetRender, fromName)
	if err != nil {
		return nil, fmt.Errorf("failed to parse target render for semantic diff: %w", err)
	}

	options := []dyff.CompareOption{
		dyff.IgnoreOrderChanges(true),
		dyff.KubernetesEntityDetection(true),
		dyff.DetectRenames(true),
		dyff.IgnoreWhitespaceChanges(true),
	}

	diff, err := dyff.CompareInputFiles(targetRenderFile, localRenderFile, options...)
	if err != nil {
		return nil, fmt.Errorf("failed to compare manifests: %w", err)
	}

	// Create our human readable report from our diffs
	report := dyff.HumanReport{
		Report:          diff,
		OmitHeader:      true,
		UseGoPatchPaths: true,
	}

	return &report, nil
}

// createInputFileFromString parses a multi-document YAML string into a dyff compatible InputFile format
func createInputFileFromString(content string, location string) (ytbx.InputFile, error) {
	var docs []*yaml.Node
	decoder := yaml.NewDecoder(strings.NewReader(content))

	for {
		var node yaml.Node
		if err := decoder.Decode(&node); err != nil {
			if err == io.EOF {
				break
			}
			return ytbx.InputFile{}, fmt.Errorf("failed to decode YAML from %s: %w", location, err)
		}
		docs = append(docs, &node)
	}

	return ytbx.InputFile{
		Location:  location,
		Documents: docs,
	}, nil
}

// getDocumentName extracts the name from a Diff path
// It uses the RootDescription which contains the K8s resource identifier
func getDocumentNameFromDiff(diff dyff.Diff) string {
	// The Path.RootDescription() contains the K8s resource identifier
	// Example: "apps/v1/Deployment/helloworld"
	desc := diff.Path.RootDescription()

	if desc != "" {
		// Remove parentheses if present: "(apps/v1/Deployment/helloworld)" -> "apps/v1/Deployment/helloworld"
		return strings.Trim(desc, "()")
	}

	return "unknown"
}

// sortedMapValues returns the values from a map[int]string sorted by key
func sortedMapValues(m map[int]string) []string {
	// Get keys and sort them
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	// Simple insertion sort since we expect small numbers of documents
	for i := 1; i < len(keys); i++ {
		key := keys[i]
		j := i - 1
		for j >= 0 && keys[j] > key {
			keys[j+1] = keys[j]
			j--
		}
		keys[j+1] = key
	}

	// Build result array
	result := make([]string, len(keys))
	for i, k := range keys {
		result[i] = m[k]
	}
	return result
}

// PrintChangeSummary prints a concise summary of changes categorized by type
func PrintChangeSummary(report dyff.Report) error {
	// Track document changes by type with their identifiers
	added := make(map[int]string)
	removed := make(map[int]string)
	modified := make(map[int]string)

	// Categorize each document based on the nature of its diffs
	for _, diff := range report.Diffs {
		docIdx := diff.Path.DocumentIdx
		docName := getDocumentNameFromDiff(diff)

		// Check if this is a document-level change (root path with no/few elements)
		isDocumentLevel := len(diff.Path.PathElements) == 0

		// Categorize based on detail kinds
		for _, detail := range diff.Details {
			switch detail.Kind {
			case dyff.ADDITION:
				if isDocumentLevel || detail.From == nil {
					// Document was added
					added[docIdx] = docName
				} else {
					// Field was added to existing document
					modified[docIdx] = docName
				}
			case dyff.REMOVAL:
				if isDocumentLevel || detail.To == nil {
					// Document was removed
					removed[docIdx] = docName
				} else {
					// Field was removed from existing document
					modified[docIdx] = docName
				}
			case dyff.MODIFICATION, dyff.ORDERCHANGE:
				// Document was modified
				modified[docIdx] = docName
			}
		}
	}

	// Remove documents from added/removed if they also appear in modified
	// (they were modified, not wholly added/removed)
	for docIdx := range modified {
		delete(added, docIdx)
		delete(removed, docIdx)
	}

	addedCount := len(added)
	removedCount := len(removed)
	modifiedCount := len(modified)
	totalObjects := addedCount + removedCount + modifiedCount

	// Build summary message
	var parts []string
	if modifiedCount > 0 {
		parts = append(parts, fmt.Sprintf("%d updated", modifiedCount))
	}
	if addedCount > 0 {
		parts = append(parts, fmt.Sprintf("%d added", addedCount))
	}
	if removedCount > 0 {
		parts = append(parts, fmt.Sprintf("%d removed", removedCount))
	}

	if len(parts) == 0 {
		return nil
	}

	changeStr := "change"
	if totalObjects != 1 {
		changeStr = "changes"
	}

	fmt.Printf("\nSummary: %d %s (%s)\n",
		totalObjects, changeStr, strings.Join(parts, ", "))

	// Print detailed lists for each category
	if len(modified) > 0 {
		fmt.Println("\nUpdated:")
		for _, id := range sortedMapValues(modified) {
			fmt.Printf("  - %s\n", id)
		}
	}

	if len(added) > 0 {
		fmt.Println("\nAdded:")
		for _, id := range sortedMapValues(added) {
			fmt.Printf("  - %s\n", id)
		}
	}

	if len(removed) > 0 {
		fmt.Println("\nRemoved:")
		for _, id := range sortedMapValues(removed) {
			fmt.Printf("  - %s\n", id)
		}
	}

	return nil
}
