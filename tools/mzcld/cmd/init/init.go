// Package init implements the `mzcld init` command, which checks that the
// local environment has the tools required to work with MozCloud.
package init

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mozilla/mozcloud/tools/mzcld/internal/executil"
	"github.com/mozilla/mozcloud/tools/mzcld/internal/ui"
	"github.com/spf13/cobra"
)

// Cmd is the `mzcld init` cobra command.
var Cmd = &cobra.Command{
	Use:   "init",
	Short: "Check that your local environment is ready for MozCloud",
	Long: `init verifies that required and optional tools are installed and that
OCI registry authentication is configured.

Required tools must be present for mzcld to function. Optional tools
are recommended but their absence only produces a warning.`,
	RunE: run,
}

var registryFlag string

func init() {
	Cmd.Flags().StringVar(&registryFlag, "registry", "us-west1-docker.pkg.dev",
		"OCI registry to check authentication for")
}

func run(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	ui.Header("Checking MozCloud dependencies...")

	checks := []ui.CheckResult{
		checkBinary(ctx, "git", "git", []string{"--version"}, parseFirstWord, true,
			"https://git-scm.com/downloads"),
		checkBinary(ctx, "helm", "helm", []string{"version", "--short"}, trimV, true,
			"brew install helm"),
		checkBinary(ctx, "gcloud", "gcloud", []string{"version"}, parseGcloud, true,
			"https://cloud.google.com/sdk/docs/install"),
		checkBinary(ctx, "kubectl", "kubectl", []string{"version", "--client"}, parseKubectl, true,
			"brew install kubectl"),
		checkBinary(ctx, "kubeconform", "kubeconform", []string{"-v"}, trimV, false,
			"brew install kubeconform"),
		checkBinary(ctx, "render-diff", "render-diff", []string{"--version"}, parseRenderDiff, false,
			"go install github.com/mozilla/mozcloud/tools/render-diff@latest"),
		checkOCIAuth(registryFlag),
	}

	failures := ui.PrintChecks(checks)

	switch {
	case failures == 0:
		ui.Success("Environment is ready.")
	case failures == 1:
		ui.Error("1 required tool is missing.")
		return fmt.Errorf("init failed")
	default:
		ui.Error(fmt.Sprintf("%d required tools are missing.", failures))
		return fmt.Errorf("init failed")
	}

	return nil
}

// checkBinary builds a CheckResult by running cmd with args.
// parse extracts the version string from command output.
// required=false means the check produces a warning, not a failure.
func checkBinary(ctx context.Context, name, bin string, args []string, parse func(string) string, required bool, fix string) ui.CheckResult {
	if !executil.LookPath(bin) {
		return ui.CheckResult{Name: name, OK: false, Warn: !required, Fix: fix}
	}
	out, err := executil.Output(ctx, bin, args...)
	version := ""
	if err == nil {
		version = parse(out)
	}
	return ui.CheckResult{Name: name, Version: version, OK: true}
}

func checkOCIAuth(registry string) ui.CheckResult {
	name := "OCI auth (" + registry + ")"
	home, err := os.UserHomeDir()
	if err != nil {
		return ui.CheckResult{Name: name, OK: false, Fix: "run: gcloud auth configure-docker " + registry}
	}

	data, err := os.ReadFile(filepath.Join(home, ".docker", "config.json"))
	if err != nil {
		return ui.CheckResult{Name: name, OK: false, Fix: "run: gcloud auth configure-docker " + registry}
	}

	var cfg struct {
		Auths       map[string]json.RawMessage `json:"auths"`
		CredHelpers map[string]string          `json:"credHelpers"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return ui.CheckResult{Name: name, OK: false, Fix: "run: gcloud auth configure-docker " + registry}
	}

	_, inAuths := cfg.Auths[registry]
	_, inHelpers := cfg.CredHelpers[registry]
	if inAuths || inHelpers {
		return ui.CheckResult{Name: name, Version: "authenticated", OK: true}
	}
	return ui.CheckResult{Name: name, OK: false, Fix: "run: gcloud auth configure-docker " + registry}
}

// --- version parsers ---

// trimV strips a leading "v" and returns the first whitespace-delimited token.
func trimV(s string) string {
	if f := strings.Fields(s); len(f) > 0 {
		return strings.TrimPrefix(f[0], "v")
	}
	return s
}

// parseFirstWord extracts the version from "git version 2.47.1".
func parseFirstWord(s string) string {
	parts := strings.Fields(s)
	if len(parts) >= 3 {
		return parts[2]
	}
	return s
}

// parseGcloud extracts the version from "Google Cloud SDK 534.0.0".
func parseGcloud(s string) string {
	for _, line := range strings.Split(s, "\n") {
		if strings.HasPrefix(line, "Google Cloud SDK") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				return parts[3]
			}
		}
	}
	return s
}

// parseKubectl extracts the version from "Client Version: v1.32.0".
func parseKubectl(s string) string {
	for _, line := range strings.Split(s, "\n") {
		if strings.HasPrefix(line, "Client Version:") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				return strings.TrimPrefix(parts[2], "v")
			}
		}
	}
	return trimV(s)
}

// parseRenderDiff extracts the version from "render-diff version v0.3.4".
func parseRenderDiff(s string) string {
	parts := strings.Fields(s)
	if len(parts) >= 3 {
		return strings.TrimPrefix(parts[2], "v")
	}
	return trimV(s)
}
