// Package discovery implements OCI/chart discovery tools:
//   - helm_chart_latest_version
//   - oci_check_auth
//   - helm_show_values
//   - helm_show_schema
package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mozilla/mozcloud/tools/mozcloud-mcp/internal/mcperr"
)

// helmExec runs a helm subcommand and returns combined stdout+stderr.
func helmExec(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "helm", args...)
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// --- helm_chart_latest_version ---

type chartLatestVersionResult struct {
	Latest   string   `json:"latest"`
	Versions []string `json:"versions"`
}

func HelmChartLatestVersion(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	repo := req.GetString("repository", "")
	if repo == "" {
		return mcp.NewToolResultText(mcperr.New(
			"invalid_input",
			"repository is required",
			"Provide the OCI repository base URL, e.g. us-west1-docker.pkg.dev/my-project/charts",
		).JSON()), nil
	}
	chartName := req.GetString("chart_name", "")
	if chartName == "" {
		return mcp.NewToolResultText(mcperr.New(
			"invalid_input",
			"chart_name is required",
			"Provide the chart name to look up",
		).JSON()), nil
	}

	ociRef := fmt.Sprintf("oci://%s/%s", strings.TrimPrefix(repo, "oci://"), chartName)

	// Fetch latest by running helm show chart without --version
	out, err := helmExec(ctx, "show", "chart", ociRef)
	if err != nil {
		return mcp.NewToolResultText(mcperr.New(
			"helm_show_chart_failed",
			fmt.Sprintf("helm show chart failed: %s", out),
			"Ensure you are authenticated to the OCI registry: run `gcloud auth configure-docker "+repo+"`",
		).JSON()), nil
	}

	// Parse version from the chart metadata YAML output
	latest := ""
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "version:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				latest = strings.TrimSpace(parts[1])
			}
		}
	}

	res := chartLatestVersionResult{
		Latest:   latest,
		Versions: []string{},
	}
	if latest != "" {
		res.Versions = []string{latest}
	}

	b, _ := json.Marshal(res)
	return mcp.NewToolResultText(string(b)), nil
}

// --- oci_check_auth ---

type ociCheckAuthResult struct {
	Authenticated bool        `json:"authenticated"`
	Error         interface{} `json:"error"`
}

func OciCheckAuth(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	registry := req.GetString("registry", "")
	if registry == "" {
		return mcp.NewToolResultText(mcperr.New(
			"invalid_input",
			"registry is required",
			"Provide the registry hostname, e.g. us-west1-docker.pkg.dev",
		).JSON()), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return mcp.NewToolResultText(mcperr.New(
			"homedir_error",
			"cannot determine home directory: "+err.Error(),
			"Ensure HOME is set in your environment",
		).JSON()), nil
	}

	configPath := filepath.Join(home, ".docker", "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		res := ociCheckAuthResult{
			Authenticated: false,
			Error:         fmt.Sprintf("docker config not found at %s", configPath),
		}
		b, _ := json.Marshal(res)
		return mcp.NewToolResultText(string(b)), nil
	}

	var cfg struct {
		Auths       map[string]json.RawMessage `json:"auths"`
		CredHelpers map[string]string          `json:"credHelpers"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return mcp.NewToolResultText(mcperr.New(
			"docker_config_parse_error",
			"failed to parse ~/.docker/config.json: "+err.Error(),
			"Regenerate docker config by running: gcloud auth configure-docker "+registry,
		).JSON()), nil
	}

	_, inAuths := cfg.Auths[registry]
	_, inCredHelpers := cfg.CredHelpers[registry]
	authenticated := inAuths || inCredHelpers

	res := ociCheckAuthResult{Authenticated: authenticated, Error: nil}
	if !authenticated {
		res.Error = fmt.Sprintf("no credential entry found for %s in ~/.docker/config.json", registry)
	}
	b, _ := json.Marshal(res)
	return mcp.NewToolResultText(string(b)), nil
}

// --- helm_show_values ---

type showValuesResult struct {
	ValuesYAML string `json:"values_yaml"`
}

func HelmShowValues(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	repo := req.GetString("repository", "")
	chartName := req.GetString("chart_name", "")
	version := req.GetString("version", "")

	if repo == "" || chartName == "" {
		return mcp.NewToolResultText(mcperr.New(
			"invalid_input",
			"repository and chart_name are required",
			"Provide both repository and chart_name arguments",
		).JSON()), nil
	}

	ociRef := fmt.Sprintf("oci://%s/%s", strings.TrimPrefix(repo, "oci://"), chartName)
	args := []string{"show", "values", ociRef}
	if version != "" {
		args = append(args, "--version", version)
	}

	out, err := helmExec(ctx, args...)
	if err != nil {
		return mcp.NewToolResultText(mcperr.New(
			"helm_show_values_failed",
			fmt.Sprintf("helm show values failed: %s", out),
			"Ensure you are authenticated: run `gcloud auth configure-docker "+repo+"`",
		).JSON()), nil
	}

	res := showValuesResult{ValuesYAML: out}
	b, _ := json.Marshal(res)
	return mcp.NewToolResultText(string(b)), nil
}

// --- helm_show_schema ---

type showSchemaResult struct {
	SchemaJSON string `json:"schema_json"`
}

func HelmShowSchema(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	repo := req.GetString("repository", "")
	chartName := req.GetString("chart_name", "")
	version := req.GetString("version", "")

	if repo == "" || chartName == "" {
		return mcp.NewToolResultText(mcperr.New(
			"invalid_input",
			"repository and chart_name are required",
			"Provide both repository and chart_name arguments",
		).JSON()), nil
	}

	// Pull the chart to a temp dir, read values.schema.json, then clean up.
	tmpDir := filepath.Join(os.TempDir(), fmt.Sprintf("mozcloud-schema-%d", rand.Int63()))
	if err := os.MkdirAll(tmpDir, 0o750); err != nil {
		return mcp.NewToolResultText(mcperr.New(
			"tmpdir_error",
			"failed to create temp directory: "+err.Error(),
			"Check disk space and permissions on "+os.TempDir(),
		).JSON()), nil
	}
	defer os.RemoveAll(tmpDir)

	ociRef := fmt.Sprintf("oci://%s/%s", strings.TrimPrefix(repo, "oci://"), chartName)
	args := []string{"pull", ociRef, "--untar", "--untardir", tmpDir}
	if version != "" {
		args = append(args, "--version", version)
	}

	if out, err := helmExec(ctx, args...); err != nil {
		return mcp.NewToolResultText(mcperr.New(
			"helm_pull_failed",
			fmt.Sprintf("helm pull failed: %s", out),
			"Ensure you are authenticated: run `gcloud auth configure-docker "+repo+"`",
		).JSON()), nil
	}

	// The chart is unpacked as tmpDir/<chart_name>/values.schema.json
	schemaPath := filepath.Join(tmpDir, chartName, "values.schema.json")
	data, err := os.ReadFile(schemaPath)
	if err != nil {
		// Schema file may not exist — return empty
		res := showSchemaResult{SchemaJSON: ""}
		b, _ := json.Marshal(res)
		return mcp.NewToolResultText(string(b)), nil
	}

	res := showSchemaResult{SchemaJSON: string(data)}
	b, _ := json.Marshal(res)
	return mcp.NewToolResultText(string(b)), nil
}
