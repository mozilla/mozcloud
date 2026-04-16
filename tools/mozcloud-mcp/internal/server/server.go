// Package server wires all tool handlers into the MCP server instance.
package server

import (
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/mozilla/mozcloud/tools/mozcloud-mcp/internal/tools/discovery"
	"github.com/mozilla/mozcloud/tools/mozcloud-mcp/internal/tools/helmops"
	"github.com/mozilla/mozcloud/tools/mozcloud-mcp/internal/tools/migration"
	"github.com/mozilla/mozcloud/tools/mozcloud-mcp/internal/tools/renderdiff"
	"github.com/mozilla/mozcloud/tools/mozcloud-mcp/internal/tools/schema"
)

// New creates and returns a fully-configured MCP server with all tools registered.
func New(version string, allowedWriteRoots []string) *mcpserver.MCPServer {
	helmops.AllowedWriteRoots = allowedWriteRoots

	s := mcpserver.NewMCPServer(
		"mozcloud-mcp",
		version,
		mcpserver.WithToolCapabilities(true),
	)

	// --- Group A: OCI / Chart Discovery ---

	s.AddTool(
		mcp.NewTool("helm_chart_latest_version",
			mcp.WithDescription("Fetch the latest version (and available versions) of a Helm chart from an OCI registry"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithString("repository",
				mcp.Required(),
				mcp.Description("OCI registry base URL, e.g. us-west1-docker.pkg.dev/my-project/charts"),
			),
			mcp.WithString("chart_name",
				mcp.Required(),
				mcp.Description("Name of the chart to look up"),
			),
			mcp.WithNumber("limit",
				mcp.Description("Maximum number of versions to return (default 5)"),
			),
		),
		discovery.HelmChartLatestVersion,
	)

	s.AddTool(
		mcp.NewTool("oci_check_auth",
			mcp.WithDescription("Check whether the local Docker credential store has an entry for an OCI registry"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithString("registry",
				mcp.Required(),
				mcp.Description("Registry hostname, e.g. us-west1-docker.pkg.dev"),
			),
		),
		discovery.OciCheckAuth,
	)

	s.AddTool(
		mcp.NewTool("helm_chart_read_file",
			mcp.WithDescription("Read one or more files from a local .tgz Helm chart archive; file_path supports glob patterns"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithString("tgz_path",
				mcp.Required(),
				mcp.Description("Path to the local .tgz chart archive, e.g. charts/mozcloud-0.9.0.tgz"),
			),
			mcp.WithString("file_path",
				mcp.Required(),
				mcp.Description("File name or glob pattern to match inside the archive, e.g. '*/values.yaml' or 'Chart.yaml'"),
			),
		),
		discovery.HelmChartReadFile,
	)

	s.AddTool(
		mcp.NewTool("helm_show_values",
			mcp.WithDescription("Show the default values.yaml for a Helm chart from an OCI registry"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithString("repository",
				mcp.Required(),
				mcp.Description("OCI registry base URL"),
			),
			mcp.WithString("chart_name",
				mcp.Required(),
				mcp.Description("Chart name"),
			),
			mcp.WithString("version",
				mcp.Description("Chart version (omit for latest)"),
			),
		),
		discovery.HelmShowValues,
	)

	s.AddTool(
		mcp.NewTool("helm_show_schema",
			mcp.WithDescription("Retrieve the values.schema.json from a Helm chart in an OCI registry"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithString("repository",
				mcp.Required(),
				mcp.Description("OCI registry base URL"),
			),
			mcp.WithString("chart_name",
				mcp.Required(),
				mcp.Description("Chart name"),
			),
			mcp.WithString("version",
				mcp.Description("Chart version (omit for latest)"),
			),
		),
		discovery.HelmShowSchema,
	)

	// --- Group B: Helm Operations ---

	s.AddTool(
		mcp.NewTool("helm_template",
			mcp.WithDescription("Render Helm chart templates locally and return the resulting Kubernetes manifests"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithString("chart_path",
				mcp.Required(),
				mcp.Description("Path to the Helm chart directory"),
			),
			mcp.WithArray("values_files",
				mcp.Description("Optional list of values file paths"),
				mcp.Items(map[string]any{"type": "string"}),
			),
			mcp.WithString("release_name",
				mcp.Description("Helm release name (defaults to chart name)"),
			),
			mcp.WithString("namespace",
				mcp.Description("Kubernetes namespace (default: default)"),
			),
			mcp.WithBoolean("update_dependencies",
				mcp.Description("Run helm dependency update before templating"),
			),
		),
		helmops.HelmTemplate,
	)

	s.AddTool(
		mcp.NewTool("helm_lint",
			mcp.WithDescription("Run helm lint on a chart and return errors and warnings"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithString("chart_path",
				mcp.Required(),
				mcp.Description("Path to the Helm chart directory"),
			),
			mcp.WithArray("values_files",
				mcp.Description("Optional list of values file paths"),
				mcp.Items(map[string]any{"type": "string"}),
			),
		),
		helmops.HelmLint,
	)

	s.AddTool(
		mcp.NewTool("helm_dependency_build",
			mcp.WithDescription("Build Helm chart dependencies from the lock file (helm dependency build)"),
			mcp.WithString("chart_path",
				mcp.Required(),
				mcp.Description("Path to the Helm chart directory"),
			),
		),
		helmops.HelmDependencyBuild,
	)

	s.AddTool(
		mcp.NewTool("helm_dependency_update",
			mcp.WithDescription("Update Helm chart dependencies and regenerate the lock file (helm dependency update)"),
			mcp.WithString("chart_path",
				mcp.Required(),
				mcp.Description("Path to the Helm chart directory"),
			),
		),
		helmops.HelmDependencyUpdate,
	)

	s.AddTool(
		mcp.NewTool("helm_pull",
			mcp.WithDescription("Pull a Helm chart from an OCI registry to a local directory"),
			mcp.WithString("repository",
				mcp.Required(),
				mcp.Description("OCI registry base URL"),
			),
			mcp.WithString("chart_name",
				mcp.Required(),
				mcp.Description("Chart name"),
			),
			mcp.WithString("version",
				mcp.Description("Chart version (omit for latest)"),
			),
			mcp.WithString("destination",
				mcp.Required(),
				mcp.Description("Local directory to pull the chart into"),
			),
			mcp.WithBoolean("untar",
				mcp.Description("Untar the chart after pulling (default: true)"),
			),
		),
		helmops.HelmPull,
	)

	// --- Group C: render-diff ---

	s.AddTool(
		mcp.NewTool("render_diff",
			mcp.WithDescription("Render a Helm chart and diff it against a git ref using the render-diff binary"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithString("chart_path",
				mcp.Required(),
				mcp.Description("Path to the Helm chart directory"),
			),
			mcp.WithArray("values_files",
				mcp.Description("Optional list of values file paths"),
				mcp.Items(map[string]any{"type": "string"}),
			),
			mcp.WithString("git_ref",
				mcp.Description("Git ref to compare against (default: main)"),
			),
			mcp.WithString("release_name",
				mcp.Description("Helm release name"),
			),
			mcp.WithBoolean("semantic",
				mcp.Description("Use semantic (dyff) diff engine"),
			),
			mcp.WithBoolean("update_dependencies",
				mcp.Description("Run helm dependency update before rendering"),
			),
		),
		renderdiff.RenderDiff,
	)

	s.AddTool(
		mcp.NewTool("render_manifests",
			mcp.WithDescription("Render a Helm chart and return the resulting Kubernetes manifests without diffing"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithString("chart_path",
				mcp.Required(),
				mcp.Description("Path to the Helm chart directory"),
			),
			mcp.WithArray("values_files",
				mcp.Description("Optional list of values file paths"),
				mcp.Items(map[string]any{"type": "string"}),
			),
			mcp.WithString("release_name",
				mcp.Description("Helm release name"),
			),
			mcp.WithBoolean("update_dependencies",
				mcp.Description("Run helm dependency update before rendering"),
			),
		),
		renderdiff.RenderManifests,
	)

	// --- Group D: Schema Validation ---

	s.AddTool(
		mcp.NewTool("schema_validate_values",
			mcp.WithDescription("Validate Helm values against a JSON schema"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithString("values_yaml",
				mcp.Description("Inline YAML values string"),
			),
			mcp.WithString("values_file",
				mcp.Description("Path to a YAML values file"),
			),
			mcp.WithString("schema_json",
				mcp.Description("Inline JSON schema string"),
			),
			mcp.WithString("schema_file",
				mcp.Description("Path to a JSON schema file"),
			),
		),
		schema.SchemaValidateValues,
	)

	// --- Group E: Migration Utilities ---

	s.AddTool(
		mcp.NewTool("migration_preflight_check",
			mcp.WithDescription("Run preflight checks before a mozcloud migration: verifies helm, render-diff, OCI auth, and git cleanliness"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithString("chart_path",
				mcp.Required(),
				mcp.Description("Path to the Helm chart directory"),
			),
			mcp.WithString("registry",
				mcp.Description("OCI registry to check auth for (default: us-west1-docker.pkg.dev)"),
			),
		),
		migration.MigrationPreflightCheck,
	)

	s.AddTool(
		mcp.NewTool("migration_read_status",
			mcp.WithDescription("Read the .migration/STATUS.md and .migration/README.md files from a chart directory"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithString("chart_path",
				mcp.Required(),
				mcp.Description("Path to the Helm chart directory"),
			),
		),
		migration.MigrationReadStatus,
	)

	s.AddTool(
		mcp.NewTool("chart_read_metadata",
			mcp.WithDescription("Read and parse Chart.yaml from a Helm chart directory"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithString("chart_path",
				mcp.Required(),
				mcp.Description("Path to the Helm chart directory"),
			),
		),
		migration.ChartReadMetadata,
	)

	s.AddTool(
		mcp.NewTool("values_list_environments",
			mcp.WithDescription("List values files matching values*.yaml in a chart directory to discover environments"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithString("chart_path",
				mcp.Required(),
				mcp.Description("Path to the Helm chart directory"),
			),
		),
		migration.ValuesListEnvironments,
	)

	return s
}
