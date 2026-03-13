// Package helmops implements Helm operation tools using the Helm Go SDK:
//   - helm_template
//   - helm_lint
//   - helm_dependency_build
//   - helm_dependency_update
//   - helm_pull
package helmops

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
	"github.com/mozilla/mozcloud/tools/mozcloud-mcp/internal/pathsafe"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/lint/support"
)

// AllowedWriteRoots is the global list of directories that side-effect tools
// may write into. Set by the CLI flag --allowed-write-roots.
var AllowedWriteRoots []string

func helmSettings() *cli.EnvSettings {
	return cli.New()
}

func debugLog(format string, v ...any) {
	fmt.Fprintf(os.Stderr, "[helm] "+format+"\n", v...)
}

// loadValuesFiles deep-merges a list of YAML values files into a single map.
// Later files take precedence over earlier ones, matching Helm CLI behaviour
// when multiple -f flags are passed. A shallow top-level merge is incorrect
// because it would drop nested defaults from earlier files whenever a later
// file contains the same top-level key.
func loadValuesFiles(files []string) (map[string]any, error) {
	merged := map[string]any{}
	for _, f := range files {
		vals, err := chartutil.ReadValuesFile(f)
		if err != nil {
			return nil, fmt.Errorf("reading values file %q: %w", f, err)
		}
		helmutil.DeepMerge(merged, vals)
	}
	return merged, nil
}

// --- helm_template ---

type templateResult struct {
	Manifests       string   `json:"manifests"`
	ResourceCount   int      `json:"resource_count"`
	ResourceSummary []string `json:"resource_summary"`
}

func HelmTemplate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
	namespace := req.GetString("namespace", "default")
	if namespace == "" {
		namespace = "default"
	}
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
		if out, depErr := runDependencyUpdate(absChartPath); depErr != nil {
			return mcp.NewToolResultText(mcperr.New(
				"dep_update_failed",
				"dependency update failed: "+depErr.Error()+"\n"+out,
				"Run `helm dependency update "+absChartPath+"` manually and check Chart.yaml",
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

	vals, err := loadValuesFiles(valuesFiles)
	if err != nil {
		return mcp.NewToolResultText(mcperr.New(
			"values_load_failed",
			"failed to load values files: "+err.Error(),
			"Ensure all values files exist and are valid YAML",
		).JSON()), nil
	}

	settings := helmSettings()
	cfg := new(action.Configuration)
	if err = cfg.Init(settings.RESTClientGetter(), namespace, "memory", debugLog); err != nil {
		return mcp.NewToolResultText(mcperr.New(
			"helm_config_error",
			"failed to init helm action config: "+err.Error(),
			"This is an internal error; ensure KUBECONFIG is set or unset",
		).JSON()), nil
	}

	client := action.NewInstall(cfg)
	client.DryRun = true
	client.ReleaseName = releaseName
	client.Namespace = namespace
	client.Replace = true
	client.ClientOnly = true
	client.IncludeCRDs = true

	rel, err := client.RunWithContext(ctx, ch, vals)
	if err != nil {
		return mcp.NewToolResultText(mcperr.New(
			"helm_template_failed",
			"helm template failed: "+err.Error(),
			"Check chart templates for errors. Run `helm template "+releaseName+" "+absChartPath+"` locally",
		).JSON()), nil
	}

	manifests := rel.Manifest
	summary := helmutil.ParseResources(manifests)
	if summary == nil {
		summary = []string{}
	}

	res := templateResult{
		Manifests:       manifests,
		ResourceCount:   len(summary),
		ResourceSummary: summary,
	}
	b, _ := json.Marshal(res)
	return mcp.NewToolResultText(string(b)), nil
}

// --- helm_lint ---

type lintResult struct {
	Passed   bool     `json:"passed"`
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
}

func HelmLint(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	chartPath := req.GetString("chart_path", "")
	if chartPath == "" {
		return mcp.NewToolResultText(mcperr.New(
			"invalid_input",
			"chart_path is required",
			"Provide the path to the Helm chart directory",
		).JSON()), nil
	}

	valuesFiles := req.GetStringSlice("values_files", nil)

	absChartPath, err := filepath.Abs(chartPath)
	if err != nil {
		return mcp.NewToolResultText(mcperr.New(
			"path_error",
			"cannot resolve chart_path: "+err.Error(),
			"Provide an absolute or valid relative path to the chart directory",
		).JSON()), nil
	}

	vals, err := loadValuesFiles(valuesFiles)
	if err != nil {
		return mcp.NewToolResultText(mcperr.New(
			"values_load_failed",
			"failed to load values files: "+err.Error(),
			"Ensure all values files exist and are valid YAML",
		).JSON()), nil
	}

	linter := action.NewLint()
	result := linter.Run([]string{absChartPath}, vals)

	var errs, warns []string
	for _, msg := range result.Messages {
		switch msg.Severity {
		case support.ErrorSev:
			errs = append(errs, msg.Error())
		case support.WarningSev:
			warns = append(warns, msg.Error())
		}
	}
	if errs == nil {
		errs = []string{}
	}
	if warns == nil {
		warns = []string{}
	}

	res := lintResult{
		Passed:   len(errs) == 0,
		Errors:   errs,
		Warnings: warns,
	}
	b, _ := json.Marshal(res)
	return mcp.NewToolResultText(string(b)), nil
}

// --- helm_dependency_build ---

type depBuildResult struct {
	Success            bool     `json:"success"`
	DependenciesLoaded []string `json:"dependencies_loaded"`
}

func HelmDependencyBuild(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	chartPath := req.GetString("chart_path", "")
	if chartPath == "" {
		return mcp.NewToolResultText(mcperr.New(
			"invalid_input",
			"chart_path is required",
			"Provide the path to the Helm chart directory",
		).JSON()), nil
	}

	absChartPath, err := filepath.Abs(chartPath)
	if err != nil {
		return mcp.NewToolResultText(mcperr.New(
			"path_error",
			"cannot resolve chart_path: "+err.Error(),
			"Provide an absolute or valid relative path to the chart directory",
		).JSON()), nil
	}

	if err = pathsafe.Check(absChartPath, AllowedWriteRoots); err != nil {
		return mcp.NewToolResultText(mcperr.New(
			"path_not_allowed",
			"write path is outside allowed roots: "+err.Error(),
			"Set --allowed-write-roots to include "+absChartPath+" when starting mozcloud-mcp",
		).JSON()), nil
	}

	settings := helmSettings()
	providers := getter.All(settings)

	var buildBuf bytes.Buffer
	man := &downloader.Manager{
		Out:              &buildBuf,
		ChartPath:        absChartPath,
		Getters:          providers,
		RepositoryConfig: settings.RepositoryConfig,
		RepositoryCache:  settings.RepositoryCache,
		Debug:            false,
	}

	if err = man.Build(); err != nil {
		return mcp.NewToolResultText(mcperr.New(
			"dep_build_failed",
			"dependency build failed: "+err.Error()+"\n"+buildBuf.String(),
			"Run `helm dependency build "+absChartPath+"` manually and verify Chart.lock is present",
		).JSON()), nil
	}

	ch, err := loader.Load(absChartPath)
	if err != nil {
		return mcp.NewToolResultText(mcperr.New(
			"chart_load_failed",
			"chart loaded but failed to inspect dependencies: "+err.Error(),
			"Check Chart.yaml for errors",
		).JSON()), nil
	}

	var loaded []string
	for _, dep := range ch.Dependencies() {
		loaded = append(loaded, dep.Metadata.Name+"@"+dep.Metadata.Version)
	}
	if loaded == nil {
		loaded = []string{}
	}

	res := depBuildResult{Success: true, DependenciesLoaded: loaded}
	b, _ := json.Marshal(res)
	return mcp.NewToolResultText(string(b)), nil
}

// --- helm_dependency_update ---

type depUpdateResult struct {
	Success     bool     `json:"success"`
	LockUpdated bool     `json:"lock_updated"`
	Resolved    []string `json:"resolved"`
}

func runDependencyUpdate(absChartPath string) (string, error) {
	var buf bytes.Buffer
	settings := helmSettings()
	providers := getter.All(settings)

	man := &downloader.Manager{
		Out:              &buf,
		ChartPath:        absChartPath,
		Getters:          providers,
		RepositoryConfig: settings.RepositoryConfig,
		RepositoryCache:  settings.RepositoryCache,
		Debug:            false,
	}
	err := man.Update()
	return buf.String(), err
}

func HelmDependencyUpdate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	chartPath := req.GetString("chart_path", "")
	if chartPath == "" {
		return mcp.NewToolResultText(mcperr.New(
			"invalid_input",
			"chart_path is required",
			"Provide the path to the Helm chart directory",
		).JSON()), nil
	}

	absChartPath, err := filepath.Abs(chartPath)
	if err != nil {
		return mcp.NewToolResultText(mcperr.New(
			"path_error",
			"cannot resolve chart_path: "+err.Error(),
			"Provide an absolute or valid relative path to the chart directory",
		).JSON()), nil
	}

	if err = pathsafe.Check(absChartPath, AllowedWriteRoots); err != nil {
		return mcp.NewToolResultText(mcperr.New(
			"path_not_allowed",
			"write path is outside allowed roots: "+err.Error(),
			"Set --allowed-write-roots to include "+absChartPath+" when starting mozcloud-mcp",
		).JSON()), nil
	}

	if out, depErr := runDependencyUpdate(absChartPath); depErr != nil {
		return mcp.NewToolResultText(mcperr.New(
			"dep_update_failed",
			"dependency update failed: "+depErr.Error()+"\n"+out,
			"Run `helm dependency update "+absChartPath+"` manually and check Chart.yaml",
		).JSON()), nil
	}

	ch, err := loader.Load(absChartPath)
	if err != nil {
		return mcp.NewToolResultText(mcperr.New(
			"chart_load_failed",
			"chart loaded but failed to inspect dependencies: "+err.Error(),
			"Check Chart.yaml for errors",
		).JSON()), nil
	}

	var resolved []string
	for _, dep := range ch.Dependencies() {
		resolved = append(resolved, dep.Metadata.Name+"@"+dep.Metadata.Version)
	}
	if resolved == nil {
		resolved = []string{}
	}

	res := depUpdateResult{Success: true, LockUpdated: true, Resolved: resolved}
	b, _ := json.Marshal(res)
	return mcp.NewToolResultText(string(b)), nil
}

// --- helm_pull ---

type pullResult struct {
	Path          string         `json:"path"`
	Version       string         `json:"version"`
	ChartMetadata map[string]any `json:"chart_metadata"`
}

func HelmPull(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	repo := req.GetString("repository", "")
	chartName := req.GetString("chart_name", "")
	version := req.GetString("version", "")
	destination := req.GetString("destination", "")

	if repo == "" || chartName == "" || destination == "" {
		return mcp.NewToolResultText(mcperr.New(
			"invalid_input",
			"repository, chart_name, and destination are required",
			"Provide all three required arguments",
		).JSON()), nil
	}

	absDestination, err := filepath.Abs(destination)
	if err != nil {
		return mcp.NewToolResultText(mcperr.New(
			"path_error",
			"cannot resolve destination: "+err.Error(),
			"Provide a valid path for destination",
		).JSON()), nil
	}

	roots := AllowedWriteRoots
	if len(roots) == 0 {
		roots = []string{absDestination}
	}
	if err := pathsafe.CheckDestination(absDestination, roots); err != nil {
		return mcp.NewToolResultText(mcperr.New(
			"path_not_allowed",
			"destination is outside allowed write roots: "+err.Error(),
			"Set --allowed-write-roots to include "+absDestination+" when starting mozcloud-mcp",
		).JSON()), nil
	}

	untar := req.GetBool("untar", true)

	ociRef := fmt.Sprintf("oci://%s/%s", strings.TrimPrefix(repo, "oci://"), chartName)
	args := []string{"pull", ociRef}
	if version != "" {
		args = append(args, "--version", version)
	}
	if untar {
		args = append(args, "--untar", "--untardir", absDestination)
	} else {
		args = append(args, "--destination", absDestination)
	}

	var buf bytes.Buffer
	cmd := exec.CommandContext(ctx, "helm", args...)
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return mcp.NewToolResultText(mcperr.New(
			"helm_pull_failed",
			fmt.Sprintf("helm pull failed: %s", buf.String()),
			"Ensure you are authenticated: run `gcloud auth configure-docker "+repo+"`",
		).JSON()), nil
	}

	// Try to load chart metadata from the pulled chart
	chartDir := filepath.Join(absDestination, chartName)
	metadata := map[string]any{}
	if ch, err := loader.Load(chartDir); err == nil {
		metadata["name"] = ch.Metadata.Name
		metadata["version"] = ch.Metadata.Version
		metadata["appVersion"] = ch.Metadata.AppVersion
		metadata["description"] = ch.Metadata.Description
		if version == "" {
			version = ch.Metadata.Version
		}
	}

	res := pullResult{
		Path:          chartDir,
		Version:       version,
		ChartMetadata: metadata,
	}
	b, _ := json.Marshal(res)
	return mcp.NewToolResultText(string(b)), nil
}
