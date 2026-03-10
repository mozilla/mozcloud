package migration_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mozilla/mozcloud/tools/mozcloud-mcp/internal/tools/migration"
)

// callTool is a helper that sets a chart_path argument and calls the handler.
func callTool(t *testing.T, fn func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error), chartPath string) string {
	t.Helper()
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"chart_path": chartPath}
	result, err := fn(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Content) == 0 {
		t.Fatal("result has no content")
	}
	text, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}
	return text.Text
}

// --- ChartReadMetadata ---

func TestChartReadMetadata_Valid(t *testing.T) {
	dir := t.TempDir()
	chartYAML := `apiVersion: v2
name: my-app
version: 1.2.3
appVersion: "4.5.6"
description: A test chart
dependencies:
  - name: redis
    version: ">=1.0.0"
    repository: https://charts.example.com
`
	if err := os.WriteFile(filepath.Join(dir, "Chart.yaml"), []byte(chartYAML), 0644); err != nil {
		t.Fatal(err)
	}

	raw := callTool(t, migration.ChartReadMetadata, dir)

	var result struct {
		Name                  string `json:"name"`
		Version               string `json:"version"`
		AppVersion            string `json:"app_version"`
		HasMozcloudDependency bool   `json:"has_mozcloud_dependency"`
	}
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result.Name != "my-app" {
		t.Errorf("name: got %q, want %q", result.Name, "my-app")
	}
	if result.Version != "1.2.3" {
		t.Errorf("version: got %q, want %q", result.Version, "1.2.3")
	}
	if result.AppVersion != "4.5.6" {
		t.Errorf("app_version: got %q, want %q", result.AppVersion, "4.5.6")
	}
	if result.HasMozcloudDependency {
		t.Error("expected has_mozcloud_dependency=false")
	}
}

func TestChartReadMetadata_DetectsMozcloudDependency(t *testing.T) {
	dir := t.TempDir()
	chartYAML := `apiVersion: v2
name: tenant-app
version: 0.1.0
dependencies:
  - name: mozcloud
    version: "1.0.0"
    repository: oci://us-west1-docker.pkg.dev/moz-fx-platform-artifacts/mozcloud-charts
    condition: mozcloud.enabled
`
	if err := os.WriteFile(filepath.Join(dir, "Chart.yaml"), []byte(chartYAML), 0644); err != nil {
		t.Fatal(err)
	}

	raw := callTool(t, migration.ChartReadMetadata, dir)

	var result struct {
		HasMozcloudDependency bool `json:"has_mozcloud_dependency"`
	}
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if !result.HasMozcloudDependency {
		t.Error("expected has_mozcloud_dependency=true")
	}
}

func TestChartReadMetadata_Missing(t *testing.T) {
	dir := t.TempDir()
	raw := callTool(t, migration.ChartReadMetadata, dir)

	var errResp struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(raw), &errResp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if errResp.Error.Code != "file_read_error" {
		t.Errorf("code: got %q, want %q", errResp.Error.Code, "file_read_error")
	}
}

func TestChartReadMetadata_MissingChartPath(t *testing.T) {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{}
	result, err := migration.ChartReadMetadata(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := result.Content[0].(mcp.TextContent).Text
	var errResp struct {
		Error struct{ Code string `json:"code"` } `json:"error"`
	}
	if err := json.Unmarshal([]byte(text), &errResp); err != nil {
		t.Fatal(err)
	}
	if errResp.Error.Code != "invalid_input" {
		t.Errorf("code: got %q, want %q", errResp.Error.Code, "invalid_input")
	}
}

// --- ValuesListEnvironments ---

func TestValuesListEnvironments(t *testing.T) {
	dir := t.TempDir()
	files := []string{
		"values.yaml",
		"values-dev.yaml",
		"values-stage.yaml",
		"values-stage-europe-west1.yaml",
		"values-prod.yaml",
	}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(dir, f), []byte(""), 0644); err != nil {
			t.Fatal(err)
		}
	}

	raw := callTool(t, migration.ValuesListEnvironments, dir)

	var result struct {
		Environments []struct {
			File        string `json:"file"`
			Environment string `json:"environment"`
		} `json:"environments"`
		Total int `json:"total"`
	}
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result.Total != len(files) {
		t.Errorf("total: got %d, want %d", result.Total, len(files))
	}

	envNames := map[string]bool{}
	for _, e := range result.Environments {
		envNames[e.Environment] = true
	}
	expected := map[string]string{
		"values.yaml":                    "default",
		"values-dev.yaml":                "dev",
		"values-stage.yaml":              "stage",
		"values-stage-europe-west1.yaml": "stage-europe-west1",
		"values-prod.yaml":               "prod",
	}
	for _, e := range result.Environments {
		base := filepath.Base(e.File)
		want, ok := expected[base]
		if !ok {
			t.Errorf("unexpected file in result: %s", base)
			continue
		}
		if e.Environment != want {
			t.Errorf("environment for %s: got %q, want %q", base, e.Environment, want)
		}
	}
}

func TestValuesListEnvironments_Empty(t *testing.T) {
	dir := t.TempDir()
	raw := callTool(t, migration.ValuesListEnvironments, dir)

	var result struct {
		Total int `json:"total"`
	}
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result.Total != 0 {
		t.Errorf("total: got %d, want 0", result.Total)
	}
}

// --- MigrationReadStatus ---

func TestMigrationReadStatus_BothExist(t *testing.T) {
	dir := t.TempDir()
	migDir := filepath.Join(dir, ".migration")
	if err := os.MkdirAll(migDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(migDir, "STATUS.md"), []byte("# Status"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(migDir, "README.md"), []byte("# README"), 0644); err != nil {
		t.Fatal(err)
	}

	raw := callTool(t, migration.MigrationReadStatus, dir)

	var result struct {
		StatusContent string `json:"status_content"`
		ReadmeContent string `json:"readme_content"`
		StatusExists  bool   `json:"status_exists"`
		ReadmeExists  bool   `json:"readme_exists"`
	}
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if !result.StatusExists {
		t.Error("expected status_exists=true")
	}
	if !result.ReadmeExists {
		t.Error("expected readme_exists=true")
	}
	if result.StatusContent != "# Status" {
		t.Errorf("status_content: got %q, want %q", result.StatusContent, "# Status")
	}
	if result.ReadmeContent != "# README" {
		t.Errorf("readme_content: got %q, want %q", result.ReadmeContent, "# README")
	}
}

func TestMigrationReadStatus_NeitherExist(t *testing.T) {
	dir := t.TempDir()
	raw := callTool(t, migration.MigrationReadStatus, dir)

	var result struct {
		StatusExists bool `json:"status_exists"`
		ReadmeExists bool `json:"readme_exists"`
	}
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result.StatusExists {
		t.Error("expected status_exists=false")
	}
	if result.ReadmeExists {
		t.Error("expected readme_exists=false")
	}
}
