// Package cmd implements the command-line interface for render-diff
// using the Cobra library.
package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/mozilla/mozcloud/tools/render-diff/internal/diff"
	"github.com/mozilla/mozcloud/tools/render-diff/internal/git"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

// Package vars
// Includes flag vars and some set during PreRun
var (
	valuesFlag     []string
	renderPathFlag string
	gitRefFlag     string
	debugFlag      bool

	repoRoot string
	fullRef  string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "render-diff",
	Short: "A CLI tool to render Helm/Kustomize and diff manifests between a local revision and target ref.",
	Long: `render-diff provides a fast and local preview of your Kubernetes manifest changes.

It renders your local Helm chart or Kustomize overlay to compare the resulting manifests against the version in a target git ref (like 'main' or 'develop'). It prints a colored diff of the final rendered YAML.`,
	Version: getVersion(),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		log.SetFlags(0) // Disabling timestamps for log output

		// A local git installation is required
		_, err := exec.LookPath("git")
		if err != nil {
			return fmt.Errorf("git not found in PATH: %w", err)
		}

		// Get Git repository root
		repoRoot, err = diff.GetRepoRoot()
		if err != nil {
			return err
		}

		// Try to find the upstream for our target ref
		upstreamRef := exec.Command("git", "rev-parse", "--abbrev-ref", gitRefFlag+"@{u}")
		upstreamRef.Dir = repoRoot

		output, err := upstreamRef.CombinedOutput()
		if err == nil {
			fullRef = strings.TrimSpace(string(output))
			if debugFlag {
				log.Printf("Found upstream for '%s', using '%s'", gitRefFlag, fullRef)
			}
		} else {
			fullRef = gitRefFlag
			log.Printf("No upstream found for '%s', using local ref", fullRef)
		}

		// Validate
		validateRef := exec.Command("git", "rev-parse", "--verify", "--quiet", fullRef)
		validateRef.Dir = repoRoot

		if out, err := validateRef.CombinedOutput(); err != nil {
			return fmt.Errorf("invalid or non-existent ref %q: %s", fullRef, strings.TrimSpace(string(out)))
		}

		return nil
	},

	RunE: func(cmd *cobra.Command, args []string) error {
		log.Printf("Starting diff against git ref '%s':", fullRef)

		// Get the absolute path from the path flag
		absPath, err := filepath.Abs(renderPathFlag)
		if err != nil {
			return fmt.Errorf("failed to resolve absolute path for -path %w", err)
		}

		// Get the relative path compared to the repoRoot)
		relativePath, err := filepath.Rel(repoRoot, absPath)
		if err != nil {
			return fmt.Errorf("failed to resolve relative path for -path %w", err)
		}

		if strings.HasPrefix(relativePath, "..") {
			return fmt.Errorf("the provided path '%s' (resolves to '%s') is outside the git repository root '%s'", renderPathFlag, absPath, repoRoot)
		}

		localPath := filepath.Join(repoRoot, relativePath)

		// Resolve relative values file paths to absolute paths for the local render
		// This means we only support values files located in the path provided
		localValuesPaths := make([]string, len(valuesFlag))
		for i, v := range valuesFlag {
			localValuesPaths[i] = filepath.Join(localPath, v)
		}

		// Setup temporary work tree for diffs
		tempDir, cleanup, err := git.SetupWorkTree(repoRoot, fullRef)
		if err != nil {
			return err
		}
		// We want this to run after we have generated our diffs
		defer cleanup()

		targetPath := filepath.Join(tempDir, relativePath)

		// Resolve values file paths for the worktree
		targetValuesPaths := make([]string, len(valuesFlag))
		for i, v := range valuesFlag {
			targetValuesPaths[i] = filepath.Join(targetPath, v)
		}

		// Create localRender and targetRender outside of goroutines
		// Create errgroup for chart/kustomization rendering
		var localRender, targetRender string
		g := new(errgroup.Group)

		// Render local Chart or Kustomization
		g.Go(func() error {
			localRender, err = diff.RenderManifests(localPath, localValuesPaths, debugFlag)
			if err != nil {
				return fmt.Errorf("failed to render path in local ref: %w", err)
			}
			return nil
		})

		// Render target Ref Chart or Kustomization
		g.Go(func() error {
			targetRender, err = diff.RenderManifests(targetPath, targetValuesPaths, debugFlag)
			if err != nil {
				// If the path does not exist in the target ref
				// We can assume it's a new addition and diff against
				// an empty string instead.
				if os.IsNotExist(err) {
					targetRender = ""
				} else {
					return fmt.Errorf("failed to render target ref manifests: %w", err)
				}
			}
			return nil
		})

		// Ensure both rendering goroutines have finished before creating our diff
		err = g.Wait()
		if err != nil {
			return err
		}

		// Generate and Print Diff
		renderedDiff := diff.CreateDiff(targetRender, localRender, fmt.Sprintf("%s/%s", fullRef, relativePath), fmt.Sprintf("local/%s", relativePath))

		if renderedDiff == "" {
			fmt.Println("\nNo differences found between rendered manifests.")
		} else {
			fmt.Printf("\n--- Diff (%s vs. local) ---\n", fullRef)
			fmt.Println(diff.ColorizeDiff(renderedDiff))
		}

		// We should not have any errors to return at this point
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	// Create a context that is cancelled on an interrupt signal
	// We want to ensure the work tree cleanup is run even if interrupted.
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	err := rootCmd.ExecuteContext(ctx)
	if err != nil {
		os.Exit(1)
	}
}

// Initializes our RootCmd with the flags below.
// Defaults to current working directory if path is not set
func init() {
	rootCmd.PersistentFlags().StringVarP(&renderPathFlag, "path", "p", ".", "Relative path to the chart or kustomization directory")
	rootCmd.PersistentFlags().StringVarP(&gitRefFlag, "ref", "r", "main", "Target Git ref to compare against with optional remote. Remote will default to 'origin' if not specified (origin/main)")
	rootCmd.PersistentFlags().StringSliceVarP(&valuesFlag, "values", "f", []string{}, "Path to an additional values file (can be specified multiple times). The chart's default values.yaml is always loaded first")
	rootCmd.PersistentFlags().BoolVarP(&debugFlag, "debug", "d", false, "Enable verbose logging for debugging")

	rootCmd.Flags().SortFlags = false
	rootCmd.PersistentFlags().SortFlags = false
}
