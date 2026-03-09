// Package migration implements migration utility tools:
//   - migration_preflight_check
//   - migration_read_status
//   - chart_read_metadata
//   - values_list_environments
package migration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mozilla/mozcloud/tools/mozcloud-mcp/internal/mcperr"
	"gopkg.in/yaml.v3"
)

// --- migration_preflight_check ---

type preflightCheck struct {
	Name    string `json:"name"`
	Passed  bool   `json:"passed"`
	Message string `json:"message"`
}

type preflightResult struct {
	AllPassed bool             `json:"all_passed"`
	Blockers  []string         `json:"blockers"`
	Checks    []preflightCheck `json:"checks"`
}

func MigrationPreflightCheck(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	chartPath := req.GetString("chart_path", "")
	if chartPath == "" {
		return mcp.NewToolResultText(mcperr.New(
			"invalid_input",
			"chart_path is required",
			"Provide the path to the Helm chart directory",
		).JSON()), nil
	}

	registry := req.GetString("registry", "us-west1-docker.pkg.dev")
	if registry == "" {
		registry = "us-west1-docker.pkg.dev"
	}

	var checks []preflightCheck
	var blockers []string

	// Check: helm available
	helmCheck := preflightCheck{Name: "helm_available"}
	helmPath, err := exec.LookPath("helm")
	if err != nil {
		helmCheck.Passed = false
		helmCheck.Message = "helm not found in PATH"
		blockers = append(blockers, "helm not found in PATH: install helm from https://helm.sh/docs/intro/install/")
	} else {
		helmCheck.Passed = true
		helmCheck.Message = "helm found at " + helmPath
	}
	checks = append(checks, helmCheck)

	// Check: helm not v4
	if helmCheck.Passed {
		versionCheck := preflightCheck{Name: "helm_version_v3"}
		out, err := exec.CommandContext(ctx, "helm", "version", "--short").Output()
		if err == nil {
			vStr := strings.TrimSpace(string(out))
			if strings.HasPrefix(vStr, "v4") {
				versionCheck.Passed = false
				versionCheck.Message = "helm v4 detected (" + vStr + "); mozcloud requires helm v3"
				blockers = append(blockers, "helm v4 is not supported; install helm v3 from https://helm.sh/docs/intro/install/")
			} else {
				versionCheck.Passed = true
				versionCheck.Message = "helm version: " + vStr
			}
		} else {
			versionCheck.Passed = false
			versionCheck.Message = "could not determine helm version: " + err.Error()
		}
		checks = append(checks, versionCheck)
	}

	// Check: render-diff available
	rdCheck := preflightCheck{Name: "render_diff_available"}
	rdPath, err := exec.LookPath("render-diff")
	if err != nil {
		rdCheck.Passed = false
		rdCheck.Message = "render-diff not found in PATH"
		blockers = append(blockers, "Install render-diff: go install github.com/mozilla/mozcloud/tools/render-diff@latest")
	} else {
		rdCheck.Passed = true
		rdCheck.Message = "render-diff found at " + rdPath
	}
	checks = append(checks, rdCheck)

	// Check: OCI auth
	authCheck := preflightCheck{Name: "oci_auth"}
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".docker", "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		authCheck.Passed = false
		authCheck.Message = "docker config not found at " + configPath
		blockers = append(blockers, "Run: gcloud auth configure-docker "+registry)
	} else {
		var cfg struct {
			Auths       map[string]json.RawMessage `json:"auths"`
			CredHelpers map[string]string          `json:"credHelpers"`
		}
		if err := json.Unmarshal(data, &cfg); err == nil {
			_, inAuths := cfg.Auths[registry]
			_, inCredHelpers := cfg.CredHelpers[registry]
			if inAuths || inCredHelpers {
				authCheck.Passed = true
				authCheck.Message = "authenticated to " + registry
			} else {
				authCheck.Passed = false
				authCheck.Message = "no credential entry found for " + registry
				blockers = append(blockers, "Run: gcloud auth configure-docker "+registry)
			}
		} else {
			authCheck.Passed = false
			authCheck.Message = "failed to parse docker config: " + err.Error()
		}
	}
	checks = append(checks, authCheck)

	// Check: git clean
	gitCheck := preflightCheck{Name: "git_clean"}
	out, err := exec.CommandContext(ctx, "git", "-C", chartPath, "status", "--porcelain").Output()
	if err != nil {
		gitCheck.Passed = false
		gitCheck.Message = "could not run git status: " + err.Error()
	} else if strings.TrimSpace(string(out)) != "" {
		gitCheck.Passed = false
		gitCheck.Message = "git working tree is not clean:\n" + string(out)
		blockers = append(blockers, "Commit or stash your changes before migrating: run `git status` to see dirty files")
	} else {
		gitCheck.Passed = true
		gitCheck.Message = "git working tree is clean"
	}
	checks = append(checks, gitCheck)

	res := preflightResult{
		AllPassed: len(blockers) == 0,
		Blockers:  blockers,
		Checks:    checks,
	}
	if res.Blockers == nil {
		res.Blockers = []string{}
	}
	b, _ := json.Marshal(res)
	return mcp.NewToolResultText(string(b)), nil
}

// --- migration_read_status ---

type migrationStatusResult struct {
	StatusContent string `json:"status_content"`
	ReadmeContent string `json:"readme_content"`
	StatusExists  bool   `json:"status_exists"`
	ReadmeExists  bool   `json:"readme_exists"`
}

func MigrationReadStatus(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	chartPath := req.GetString("chart_path", "")
	if chartPath == "" {
		return mcp.NewToolResultText(mcperr.New(
			"invalid_input",
			"chart_path is required",
			"Provide the path to the Helm chart directory",
		).JSON()), nil
	}

	migDir := filepath.Join(chartPath, ".migration")

	statusPath := filepath.Join(migDir, "STATUS.md")
	readmePath := filepath.Join(migDir, "README.md")

	statusContent, statusErr := os.ReadFile(statusPath)
	readmeContent, readmeErr := os.ReadFile(readmePath)

	res := migrationStatusResult{
		StatusContent: string(statusContent),
		ReadmeContent: string(readmeContent),
		StatusExists:  statusErr == nil,
		ReadmeExists:  readmeErr == nil,
	}
	b, _ := json.Marshal(res)
	return mcp.NewToolResultText(string(b)), nil
}

// --- chart_read_metadata ---

type chartMetadataResult struct {
	Name                  string                 `json:"name"`
	Version               string                 `json:"version"`
	AppVersion            string                 `json:"app_version"`
	Description           string                 `json:"description"`
	APIVersion            string                 `json:"api_version"`
	Type                  string                 `json:"type"`
	Dependencies          []map[string]string    `json:"dependencies"`
	HasMozcloudDependency bool                   `json:"has_mozcloud_dependency"`
	Raw                   map[string]interface{} `json:"raw"`
}

func ChartReadMetadata(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	chartPath := req.GetString("chart_path", "")
	if chartPath == "" {
		return mcp.NewToolResultText(mcperr.New(
			"invalid_input",
			"chart_path is required",
			"Provide the path to the Helm chart directory",
		).JSON()), nil
	}

	chartYAMLPath := filepath.Join(chartPath, "Chart.yaml")
	data, err := os.ReadFile(chartYAMLPath)
	if err != nil {
		return mcp.NewToolResultText(mcperr.New(
			"file_read_error",
			"failed to read Chart.yaml: "+err.Error(),
			"Ensure chart_path contains a Chart.yaml file",
		).JSON()), nil
	}

	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return mcp.NewToolResultText(mcperr.New(
			"yaml_parse_error",
			"failed to parse Chart.yaml: "+err.Error(),
			"Ensure Chart.yaml is valid YAML. Validate with: helm lint "+chartPath,
		).JSON()), nil
	}

	res := chartMetadataResult{Raw: raw}
	if v, ok := raw["name"].(string); ok {
		res.Name = v
	}
	if v, ok := raw["version"].(string); ok {
		res.Version = v
	}
	if v, ok := raw["appVersion"].(string); ok {
		res.AppVersion = v
	}
	if v, ok := raw["description"].(string); ok {
		res.Description = v
	}
	if v, ok := raw["apiVersion"].(string); ok {
		res.APIVersion = v
	}
	if v, ok := raw["type"].(string); ok {
		res.Type = v
	}

	if deps, ok := raw["dependencies"].([]interface{}); ok {
		for _, dep := range deps {
			if d, ok := dep.(map[string]interface{}); ok {
				entry := map[string]string{}
				for k, v := range d {
					if s, ok := v.(string); ok {
						entry[k] = s
					}
				}
				res.Dependencies = append(res.Dependencies, entry)
				if name, ok := entry["name"]; ok {
					if strings.Contains(strings.ToLower(name), "mozcloud") {
						res.HasMozcloudDependency = true
					}
				}
				if repo, ok := entry["repository"]; ok {
					if strings.Contains(strings.ToLower(repo), "mozcloud") {
						res.HasMozcloudDependency = true
					}
				}
			}
		}
	}
	if res.Dependencies == nil {
		res.Dependencies = []map[string]string{}
	}

	b, _ := json.Marshal(res)
	return mcp.NewToolResultText(string(b)), nil
}

// --- values_list_environments ---

type environmentEntry struct {
	File        string `json:"file"`
	Environment string `json:"environment"`
}

type valuesListEnvironmentsResult struct {
	Environments []environmentEntry `json:"environments"`
	Total        int                `json:"total"`
}

func ValuesListEnvironments(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	chartPath := req.GetString("chart_path", "")
	if chartPath == "" {
		return mcp.NewToolResultText(mcperr.New(
			"invalid_input",
			"chart_path is required",
			"Provide the path to the Helm chart directory",
		).JSON()), nil
	}

	pattern := filepath.Join(chartPath, "values*.yaml")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return mcp.NewToolResultText(mcperr.New(
			"glob_error",
			"failed to glob values files: "+err.Error(),
			"Ensure chart_path is a valid directory",
		).JSON()), nil
	}

	var envs []environmentEntry
	for _, match := range matches {
		base := filepath.Base(match)
		// Derive environment name from filename:
		// values.yaml -> "default"
		// values-prod.yaml -> "prod"
		// values-staging.yaml -> "staging"
		env := "default"
		trimmed := strings.TrimSuffix(base, ".yaml")
		trimmed = strings.TrimSuffix(trimmed, ".yml")
		if trimmed != "values" {
			// Remove leading "values-" or "values."
			env = strings.TrimPrefix(trimmed, "values-")
			env = strings.TrimPrefix(env, "values.")
			if env == trimmed {
				env = trimmed
			}
		}
		envs = append(envs, environmentEntry{
			File:        match,
			Environment: env,
		})
	}

	// Also scan subdirectories one level deep for env-specific values
	entries, err := os.ReadDir(chartPath)
	if err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			subPattern := filepath.Join(chartPath, e.Name(), "values*.yaml")
			subMatches, _ := filepath.Glob(subPattern)
			for _, match := range subMatches {
				base := filepath.Base(match)
				env := fmt.Sprintf("%s/%s", e.Name(), strings.TrimSuffix(base, ".yaml"))
				envs = append(envs, environmentEntry{
					File:        match,
					Environment: env,
				})
			}
		}
	}

	if envs == nil {
		envs = []environmentEntry{}
	}

	res := valuesListEnvironmentsResult{
		Environments: envs,
		Total:        len(envs),
	}
	b, _ := json.Marshal(res)
	return mcp.NewToolResultText(string(b)), nil
}
