package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hexops/gotextdiff" // This is archived, but I could not find a better alternative at the moment
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/engine"
	"helm.sh/helm/v3/pkg/getter"
)

// ANSI codes for diff colors
const (
	colorRed   = "\033[31m"
	colorGreen = "\033[32m"
	colorCyan  = "\033[36m"
	colorReset = "\033[0m"
)

// valuesArray is a custom type to support multiple --values flags
type valuesArray []string

func (i *valuesArray) String() string {
	return strings.Join(*i, ", ")
}

func (i *valuesArray) Set(value string) error {
	*i = append(*i, value)
	return nil
}

// getRepoRoot finds the top-level directory of the current git repository.
func getRepoRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to find git repo root: %w. Make sure you are running this inside a git repository. Output: %s", err, string(output))
	}
	return strings.TrimSpace(string(output)), nil
}

func main() {
	// Define and Parse Flags
	var valuesFlag valuesArray
	chartPathFlag := flag.String("chart-path", "", "Relative path to the chart (required)")
	gitRefFlag := flag.String("ref", "main", "Target Git ref to compare against (e.g., 'main', 'develop', 'v1.2.0')")

	flag.Var(&valuesFlag, "values", "Path to an additional values file, relative to the chart-path (can be specified multiple times). The chart's 'values.yaml' is always included first.")

	flag.Parse()

	// Chart Path is required
	if *chartPathFlag == "" {
		log.Println("Error: --chart-path flag is required.")
		flag.Usage()
		os.Exit(1)
	}

	log.Printf("Starting Helm chart diff against git ref '%s'", *gitRefFlag)

	// Get Git Root and Define Paths
	repoRoot, err := getRepoRoot()
	if err != nil {
		log.Fatal(err.Error())
	}

	// Get the absolute path from the chart-path flag
	absChartPath, err := filepath.Abs(*chartPathFlag)
	if err != nil {
		log.Fatalf("Failed to resolve absolute chart path for --chart-path %v", err)
	}

	// Get the relative path compared to the repoRoot)
	relativeChartPath, err := filepath.Rel(repoRoot, absChartPath)
	if err != nil {
		log.Fatalf("Failed to resolve relative chart path for --chart-path %v", err)
	}

	if strings.HasPrefix(relativeChartPath, "..") {
		log.Fatalf("Error: The provided path '%s' (resolves to '%s') is outside the git repository root '%s'.", *chartPathFlag, absChartPath, repoRoot)
	}

	localChartPath := filepath.Join(repoRoot, relativeChartPath)

	// Resolve relative values file paths to absolute paths for the local render
	// This means we only support values files located in the chart path provided
	localValuesPaths := make([]string, len(valuesFlag))
	for i, v := range valuesFlag {
		localValuesPaths[i] = filepath.Join(localChartPath, v)
	}

	// Render Local (Feature Branch) Chart
	log.Println("Rendering chart from local branch...")
	localRender, err := renderChart(localChartPath, "release", localValuesPaths)
	if err != nil {
		log.Fatalf("Failed to render local chart: %v", err)
	}

	// Set up Git Worktree for Target Ref
	log.Printf("Creating temporary worktree for '%s' ref...", *gitRefFlag)
	tempDir, err := os.MkdirTemp("", "diff-ref-")
	if err != nil {
		log.Fatalf("Failed to create temp directory: %v", err)
	}

	// Defer LIFO: 1. Remove dir (runs 2nd), 2. Remove worktree (runs 1st)
	// Clean up temp directories before our diff is returned
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			fmt.Printf("Error removing temporary directory %s: %v\n", tempDir, err)
		}
	}()

	defer func() {
		log.Printf("Cleaning up worktree at %s...", tempDir)
		// Using --force to avoid errors if dir is already partially cleaned
		cleanupCmd := exec.Command("git", "worktree", "remove", "--force", tempDir)
		cleanupCmd.Dir = repoRoot // Run from the repo root
		if output, err := cleanupCmd.CombinedOutput(); err != nil {
			// Log as a warning, not fatal, so we don't stop execution
			log.Printf("Warning: failed to run 'git worktree remove'. Manual cleanup may be required. Error: %v, Output: %s", err, string(output))
		}
	}()

	// Create the worktree
	// Using -d to allow checking out a branch that is already checked out (like 'main')
	addCmd := exec.Command("git", "worktree", "add", "-d", tempDir, *gitRefFlag)
	addCmd.Dir = repoRoot // Run from the repo root
	if output, err := addCmd.CombinedOutput(); err != nil {
		log.Fatalf("Failed to create worktree for '%s': %v\nOutput: %s", *gitRefFlag, err, string(output))
	}
	log.Printf("Worktree for '%s' created at: %s", *gitRefFlag, tempDir)

	// Render Target Ref Chart
	targetChartPath := filepath.Join(tempDir, relativeChartPath)

	// Resolve values file paths for the worktree
	targetValuesPaths := make([]string, len(valuesFlag))
	for i, v := range valuesFlag {
		targetValuesPaths[i] = filepath.Join(targetChartPath, v)
	}

	log.Printf("Rendering chart from '%s' ref...", *gitRefFlag)
	targetRender, err := renderChart(targetChartPath, "release", targetValuesPaths)
	if err != nil {
		log.Fatalf("Failed to render '%s' ref chart: %v", *gitRefFlag, err)
	}

	// Generate and Print Diff
	log.Println("Generating diff...")
	diff := createDiff(targetRender, localRender, fmt.Sprintf("%s/%s", *gitRefFlag, relativeChartPath), fmt.Sprintf("local/%s", relativeChartPath))

	if diff == "" {
		fmt.Println("\nNo differences found between rendered charts.")
	} else {
		fmt.Printf("\n--- Chart Differences (%s vs. Local) ---\n", *gitRefFlag)
		fmt.Println(colorizeDiff(diff))
	}
}

// loadValues merges multiple values files in order, mimicking 'helm -f file1 -f file2'
func loadValues(valuesFiles []string) (chartutil.Values, error) {
	mergedValues := chartutil.Values{}

	for _, path := range valuesFiles {
		// Check if file exists. It's not an error if a values file is missing
		// in one branch but not the other; Helm just skips it.
		if _, err := os.Stat(path); os.IsNotExist(err) {
			log.Printf("Warning: values file '%s' not found, skipping.", path)
			continue
		}

		currentValues, err := chartutil.ReadValuesFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read values file %s: %w", path, err)
		}

		// Coalesce merges the two maps, with 'currentValues' overwriting 'mergedValues'
		// This matches Helm, later values files override earlier ones. 'helm -f file1 -f file2'
		mergedValues = chartutil.CoalesceTables(currentValues, mergedValues)
	}
	return mergedValues, nil
}

// renderChart loads, merges values, and renders a Helm chart
func renderChart(chartPath, releaseName string, valuesFiles []string) (string, error) {
	chart, err := loader.Load(chartPath)
	if err != nil {
		return "", fmt.Errorf("failed to load chart from %s: %w", chartPath, err)
	}

	// Helm Dependency Build
	// Run 'helm dependency build' if dependencies are present
	if chart.Metadata.Dependencies != nil {
		log.Printf("Chart has dependencies, running 'helm dependency build' for: %s", chartPath)

		// We need a basic cli.EnvSettings to init the getter.Providers.
		settings := cli.New()
		getters := getter.All(settings)

		// Create a downloader manager.
		man := downloader.Manager{
			Out:       log.Writer(),
			ChartPath: chartPath,
			Getters:   getters,
		}

		// Run build. This downloads charts into the 'charts/' directory.
		if err := man.Build(); err != nil {
			return "", fmt.Errorf("failed to run dependency build: %w", err)
		}

		// Reload the chart after building dependencies
		// This ensures the newly downloaded subcharts are included in the render.
		log.Printf("Reloading chart to include dependencies...")
		chart, err = loader.Load(chartPath)
		if err != nil {
			return "", fmt.Errorf("failed to reload chart after dependency build: %w", err)
		}
	}

	// Load additional values files from the --values flags
	userValues, err := loadValues(valuesFiles)
	if err != nil {
		return "", fmt.Errorf("failed to load/merge values: %w", err)
	}

	// Define release options for the render
	options := chartutil.ReleaseOptions{
		Name:      releaseName, // We don't need a real releaseName or namespace for the diff
		Namespace: "default",
		Revision:  1,
		IsInstall: true,
	}

	// Get render values. This merges the chart's default values (from chart.Values/values.yaml)
	// with the user-supplied values (from userValues).
	renderVals, err := chartutil.ToRenderValues(chart, userValues, options, nil)
	if err != nil {
		return "", fmt.Errorf("failed to prepare render values: %w", err)
	}

	// Render the chart
	renderedTemplates, err := engine.Render(chart, renderVals)
	if err != nil {
		return "", fmt.Errorf("failed to render chart: %w", err)
	}

	// Concatenate all rendered templates into a single string for easier diffing
	var builder strings.Builder
	keys := make([]string, 0, len(renderedTemplates))
	for k := range renderedTemplates {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		content := renderedTemplates[key]
		// Skip empty templates, partials, or NOTES.txt
		if strings.TrimSpace(content) == "" ||
			strings.HasSuffix(key, ".tpl") ||
			strings.HasSuffix(key, "NOTES.txt") {
			continue
		}
		builder.WriteString("---\n")
		builder.WriteString(fmt.Sprintf("# Source: %s\n", key))
		builder.WriteString(content)
		builder.WriteString("\n")
	}

	return builder.String(), nil
}

// createDiff generates a unified diff string between two text inputs.
func createDiff(a, b string, fromName, toName string) string {
	edits := myers.ComputeEdits(span.URI(fromName), a, b)
	diff := gotextdiff.ToUnified(fromName, toName, a, edits)

	return fmt.Sprint(diff)
}

// colorizeDiff adds simple ANSI colors to a diff string.
// We want to see this output in a terminal or as a comment on a PR
// Fast readability is important
func colorizeDiff(diff string) string {
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
