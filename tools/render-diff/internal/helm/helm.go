// Package helm provides functions to render helm charts,
// load multiple values files provided as a cli flag and
// check if the provided path is a helm chart
package helm

import (
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strings"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/engine"
	"helm.sh/helm/v3/pkg/getter"
)

// renderChart loads, merges values, and renders a Helm chart
func RenderChart(chartPath, releaseName string, valuesFiles []string, debug bool) (string, error) {
	chart, err := loadChart(chartPath, debug)
	if err != nil {
		if os.IsNotExist(err) {
			return "", err
		}
		return "", fmt.Errorf("failed to load chart from %s: %w", chartPath, err)
	}

	// Helm Dependency Build
	// Run 'helm dependency build' if dependencies are present
	if chart.Metadata.Dependencies != nil {
		if debug {
			log.Printf("Chart has dependencies, running 'helm dependency build' for: %s", chartPath)
		}

		// We need a basic cli.EnvSettings to init the getter.Providers.
		settings := cli.New()
		settings.Debug = debug // Setting debug to match flag

		getters := getter.All(settings)

		// Create a downloader manager.
		man := downloader.Manager{
			Out:       io.Discard,
			ChartPath: chartPath,
			Getters:   getters,
			Debug:     debug,
		}

		// Run build. This downloads charts into the 'charts/' directory.
		// We are ignoring some log output here, which can be reverted with the --debug flag
		err = silentRun(debug, func() error {
			return man.Build()
		})
		if err != nil {
			return "", fmt.Errorf("failed to run dependency build: %w", err)
		}
		if err != nil {
			return "", fmt.Errorf("failed to run dependency build: %w", err)
		}

		// Reload the chart after building dependencies
		// This ensures the newly downloaded subcharts are included in the render.
		chart, err = loadChart(chartPath, debug)
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

// IsHelmChart will try to load the path as a Helm Chart, if it fails we'll return false
func IsHelmChart(path string) bool {
	_, err := loadChart(path, false)

	return err == nil
}

// loadChart will check the debug bool and either use the
// default loader.Load or our wrappter to run it silently
func loadChart(path string, debug bool) (*chart.Chart, error) {
	var chart *chart.Chart
	err := silentRun(debug, func() error {
		var err error
		chart, err = loader.Load(path)
		return err
	})
	return chart, err
}

// We use symlinks in our charts and end up with a lot of extra output
// Disabling some log output when loading a chart and
// updating dependencies. Use the --debug flag to have full logging.
func silentRun(debug bool, fn func() error) error {
	if debug {
		return fn()
	}

	// If debug is off, silence the global logger
	currentLogger := log.Writer()
	log.SetOutput(io.Discard)

	// Defer the restore
	defer log.SetOutput(currentLogger)

	return fn()
}
