// Package renderdiff implements tools that shell out to the render-diff binary:
//   - render_diff
//   - render_manifests (uses Helm SDK directly for rendering)
package renderdiff

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mozilla/mozcloud/tools/mozcloud-mcp/internal/helmutil"
	"github.com/mozilla/mozcloud/tools/mozcloud-mcp/internal/mcperr"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
)

const renderDiffBinary = "render-diff"

func debugLog(format string, v ...any) {
	fmt.Fprintf(os.Stderr, "[helm] "+format+"\n", v...)
}

// --- render_diff ---

type renderDiffResult struct {
	HasDiff  bool   `json:"has_diff"`
	DiffText string `json:"diff_text"`
	Summary  string `json:"summary"`
}

func RenderDiff(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	chartPath := req.GetString("chart_path", "")
	if chartPath == "" {
		return mcp.NewToolResultText(mcperr.New(
			"invalid_input",
			"chart_path is required",
			"Provide the path to the Helm chart directory",
		).JSON()), nil
	}

	// Resolve chart path to absolute so we can use it as the working directory
	absChartPath, err := filepath.Abs(chartPath)
	if err != nil {
		return mcp.NewToolResultText(mcperr.New(
			"path_error",
			"cannot resolve chart_path: "+err.Error(),
			"Provide an absolute or valid relative path to the chart directory",
		).JSON()), nil
	}

	// Build render-diff args.
	// We run render-diff with cmd.Dir=absChartPath and pass --path . so that
	// render-diff treats values files as relative to the chart directory rather
	// than joining them with an absolute --path (which doubles the path in Go's
	// filepath.Join when values are already absolute).
	args := []string{"--path", ".", "--no-color"}

	if gitRef := req.GetString("git_ref", ""); gitRef != "" {
		args = append(args, "--ref", gitRef)
	}
	if releaseName := req.GetString("release_name", ""); releaseName != "" {
		args = append(args, "--release-name", releaseName)
	}
	if req.GetBool("semantic", false) {
		args = append(args, "--semantic")
	}
	if req.GetBool("update_dependencies", false) {
		args = append(args, "--update")
	}
	for _, f := range req.GetStringSlice("values_files", nil) {
		// Convert absolute paths to relative (from absChartPath) so render-diff
		// can resolve them correctly relative to --path .
		if filepath.IsAbs(f) {
			if rel, relErr := filepath.Rel(absChartPath, f); relErr == nil {
				f = rel
			}
		}
		args = append(args, "--values", f)
	}

	path, err := exec.LookPath(renderDiffBinary)
	if err != nil {
		return mcp.NewToolResultText(mcperr.New(
			"binary_not_found",
			"render-diff binary not found in PATH",
			"Install render-diff: go install github.com/mozilla/mozcloud/tools/render-diff@latest",
		).JSON()), nil
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Dir = absChartPath
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()

	output := stdout.String()
	hasDiff := false
	summary := "No differences found"

	if runErr != nil {
		// render-diff exits non-zero when there are differences
		if strings.Contains(output, "---") || strings.Contains(output, "+++") || strings.Contains(output, "Diff") {
			hasDiff = true
			summary = helmutil.SummarizeDiff(output)
		} else {
			// Actual error
			return mcp.NewToolResultText(mcperr.New(
				"render_diff_failed",
				fmt.Sprintf("render-diff failed: %s %s", output, stderr.String()),
				"Check that the chart_path is valid and git ref exists. Run `render-diff --path "+chartPath+"` manually",
			).JSON()), nil
		}
	} else {
		if strings.Contains(output, "---") || strings.Contains(output, "+++") {
			hasDiff = true
			summary = helmutil.SummarizeDiff(output)
		}
	}

	res := renderDiffResult{
		HasDiff:  hasDiff,
		DiffText: output,
		Summary:  summary,
	}
	b, _ := json.Marshal(res)
	return mcp.NewToolResultText(string(b)), nil
}

// --- render_manifests ---

type renderManifestsResult struct {
	Manifests     string   `json:"manifests"`
	Resources     []string `json:"resources"`
	ResourceCount int      `json:"resource_count"`
}

func RenderManifests(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	chartPath := req.GetString("chart_path", "")
	if chartPath == "" {
		return mcp.NewToolResultText(mcperr.New(
			"invalid_input",
			"chart_path is required",
			"Provide the path to the Helm chart directory",
		).JSON()), nil
	}

	valuesFiles := req.GetStringSlice("values_files", nil)
	releaseName := req.GetString("release_name", "")
	updateDeps := req.GetBool("update_dependencies", false)

	absChartPath, err := filepath.Abs(chartPath)
	if err != nil {
		return mcp.NewToolResultText(mcperr.New(
			"path_error",
			"cannot resolve chart_path: "+err.Error(),
			"Provide an absolute or valid relative path to the chart directory",
		).JSON()), nil
	}

	if updateDeps {
		settings := cli.New()
		providers := getter.All(settings)
		man := &downloader.Manager{
			Out:              os.Stderr,
			ChartPath:        absChartPath,
			Getters:          providers,
			RepositoryConfig: settings.RepositoryConfig,
			RepositoryCache:  settings.RepositoryCache,
		}
		if err := man.Update(); err != nil {
			return mcp.NewToolResultText(mcperr.New(
				"dep_update_failed",
				"dependency update failed: "+err.Error(),
				"Run `helm dependency update "+absChartPath+"` manually",
			).JSON()), nil
		}
	}

	ch, err := loader.Load(absChartPath)
	if err != nil {
		return mcp.NewToolResultText(mcperr.New(
			"chart_load_failed",
			"failed to load chart: "+err.Error(),
			"Ensure chart_path points to a valid Helm chart directory with Chart.yaml",
		).JSON()), nil
	}

	if releaseName == "" {
		releaseName = ch.Metadata.Name
	}

	vals := map[string]any{}
	for _, f := range valuesFiles {
		v, err := chartutil.ReadValuesFile(f)
		if err != nil {
			return mcp.NewToolResultText(mcperr.New(
				"values_load_failed",
				"failed to read values file "+f+": "+err.Error(),
				"Ensure the values file exists and is valid YAML",
			).JSON()), nil
		}
		helmutil.DeepMerge(vals, v)
	}

	settings := cli.New()
	cfg := new(action.Configuration)
	if err := cfg.Init(settings.RESTClientGetter(), "default", "memory", debugLog); err != nil {
		return mcp.NewToolResultText(mcperr.New(
			"helm_config_error",
			"failed to init helm action config: "+err.Error(),
			"This is an internal error; ensure KUBECONFIG is set or unset",
		).JSON()), nil
	}

	client := action.NewInstall(cfg)
	client.DryRun = true
	client.ReleaseName = releaseName
	client.Namespace = "default"
	client.Replace = true
	client.ClientOnly = true
	client.IncludeCRDs = true

	rel, err := client.RunWithContext(ctx, ch, vals)
	if err != nil {
		return mcp.NewToolResultText(mcperr.New(
			"helm_template_failed",
			"rendering failed: "+err.Error(),
			"Check chart templates for errors. Run `helm template "+releaseName+" "+absChartPath+"` locally",
		).JSON()), nil
	}

	manifests := rel.Manifest
	resources := helmutil.ParseResources(manifests)
	if resources == nil {
		resources = []string{}
	}

	res := renderManifestsResult{
		Manifests:     manifests,
		Resources:     resources,
		ResourceCount: len(resources),
	}
	b, _ := json.Marshal(res)
	return mcp.NewToolResultText(string(b)), nil
}
