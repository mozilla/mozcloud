package chart

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	_ "embed"

	"charm.land/huh/v2"
	"github.com/mozilla/mozcloud/tools/mzcld/internal/scaffold"
	"github.com/mozilla/mozcloud/tools/mzcld/internal/ui"
	"github.com/spf13/cobra"
)

//go:embed Chart.yaml.tmpl
var chartYAMLTmpl string

const (
	defaultRepository = "us-west1-docker.pkg.dev/moz-fx-platform-artifacts/mozcloud-charts"
	dependencyChart   = "mozcloud"
)

var newCmd = &cobra.Command{
	Use:   "new",
	Short: "Scaffold a new MozCloud Helm chart",
	Long: `new creates a new Helm chart pre-configured to use the mozcloud dependency chart.

It fetches the latest mozcloud values schema from the OCI registry and generates
an annotated values.yaml so your values stay aligned with the chart's schema.`,
	RunE: runNew,
}

var (
	nameFlag       string
	descFlag       string
	outputFlag     string
	repositoryFlag string
	versionFlag    string
)

func init() {
	newCmd.Flags().StringVar(&nameFlag, "name", "", "Chart name (skips prompt)")
	newCmd.Flags().StringVar(&descFlag, "description", "", "Chart description (skips prompt)")
	newCmd.Flags().StringVar(&outputFlag, "output", "", "Output directory (default: ./<name>)")
	newCmd.Flags().StringVar(&repositoryFlag, "repository", defaultRepository, "OCI repository for the mozcloud chart")
	newCmd.Flags().StringVar(&versionFlag, "version", "", "mozcloud chart version (default: latest)")
	newCmd.Flags().SortFlags = false
}

type chartParams struct {
	Name            string
	Description     string
	MozcloudVersion string
}

func runNew(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	params, err := gatherParams(ctx)
	if err != nil {
		return err
	}

	outDir := outputFlag
	if outDir == "" {
		outDir = params.Name
	}

	if _, err := os.Stat(outDir); err == nil {
		return fmt.Errorf("output directory %q already exists", outDir)
	}

	// Fetch schema from OCI registry
	ui.Info("Fetching mozcloud schema from registry...")
	schemaJSON, err := fetchSchema(ctx, repositoryFlag, versionFlag)
	if err != nil {
		return fmt.Errorf("failed to fetch schema: %w\n\nEnsure you are authenticated: run `gcloud auth configure-docker %s`", err, repositoryFlag)
	}

	var root scaffold.Schema
	if err := json.Unmarshal([]byte(schemaJSON), &root); err != nil {
		return fmt.Errorf("failed to parse schema: %w", err)
	}

	// Generate files
	if err := os.MkdirAll(outDir, 0o750); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	if err := writeChartYAML(outDir, params); err != nil {
		return err
	}
	if err := writeValuesYAML(outDir, &root); err != nil {
		return err
	}

	ui.Header("Created " + outDir + "/")
	ui.Success("Chart.yaml")
	ui.Success("values.yaml  (generated from mozcloud schema " + params.MozcloudVersion + ")")
	fmt.Println()
	ui.Info("Next steps:")
	ui.Dim("  cd " + outDir)
	ui.Dim("  helm dependency update")
	ui.Dim("  # edit values.yaml, then:")
	ui.Dim("  helm template . -f values.yaml")

	return nil
}

func gatherParams(ctx context.Context) (*chartParams, error) {
	params := &chartParams{
		Name:        nameFlag,
		Description: descFlag,
	}

	// Resolve mozcloud version first (needed for display in prompts)
	ui.Info("Resolving latest mozcloud chart version...")
	version, err := resolveVersion(ctx, repositoryFlag, versionFlag)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve chart version: %w", err)
	}
	params.MozcloudVersion = version

	// Prompt for any missing params
	var fields []huh.Field
	if params.Name == "" {
		fields = append(fields,
			huh.NewInput().
				Title("Chart name").
				Description("Lowercase, hyphens allowed (e.g. my-service)").
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("name is required")
					}
					return nil
				}).
				Value(&params.Name),
		)
	}
	if params.Description == "" {
		fields = append(fields,
			huh.NewInput().
				Title("Description").
				Value(&params.Description),
		)
	}

	if len(fields) > 0 {
		form := huh.NewForm(huh.NewGroup(fields...))
		if err := form.Run(); err != nil {
			return nil, err
		}
	}

	return params, nil
}

func resolveVersion(ctx context.Context, repo, version string) (string, error) {
	if version != "" {
		return version, nil
	}
	ociRef := fmt.Sprintf("oci://%s/%s", strings.TrimPrefix(repo, "oci://"), dependencyChart)
	cmd := exec.CommandContext(ctx, "helm", "show", "chart", ociRef)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("helm show chart failed: %w", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		if after, ok := strings.CutPrefix(line, "version:"); ok {
			return strings.TrimSpace(after), nil
		}
	}
	return "latest", nil
}

func fetchSchema(ctx context.Context, repo, version string) (string, error) {
	tmpDir, err := os.MkdirTemp("", "mzcld-schema-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir) //nolint:errcheck

	ociRef := fmt.Sprintf("oci://%s/%s", strings.TrimPrefix(repo, "oci://"), dependencyChart)
	args := []string{"pull", ociRef, "--untar", "--untardir", tmpDir}
	if version != "" {
		args = append(args, "--version", version)
	}

	cmd := exec.CommandContext(ctx, "helm", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("%s", strings.TrimSpace(string(out)))
	}

	schemaPath := filepath.Join(tmpDir, dependencyChart, "values.schema.json")
	data, err := os.ReadFile(schemaPath)
	if err != nil {
		return "", fmt.Errorf("values.schema.json not found in chart: %w", err)
	}
	return string(data), nil
}

func writeChartYAML(outDir string, params *chartParams) error {
	tmpl, err := template.New("Chart.yaml").Parse(chartYAMLTmpl)
	if err != nil {
		return fmt.Errorf("failed to parse Chart.yaml template: %w", err)
	}
	f, err := os.Create(filepath.Join(outDir, "Chart.yaml"))
	if err != nil {
		return err
	}
	defer f.Close() //nolint:errcheck
	return tmpl.Execute(f, params)
}

func writeValuesYAML(outDir string, root *scaffold.Schema) error {
	content := scaffold.GenerateYAML(root, dependencyChart)
	return os.WriteFile(filepath.Join(outDir, "values.yaml"), []byte(content), 0o640)
}
