package gsm

import (
	"fmt"
	"strings"

	"charm.land/huh/v2"
	"charm.land/huh/v2/spinner"
	"github.com/charmbracelet/lipgloss"
	"github.com/mozilla/mozcloud/tools/mzcld/internal/gsm"
	"github.com/mozilla/mozcloud/tools/mzcld/internal/ui"
	"github.com/pmezard/go-difflib/difflib"
	"github.com/spf13/cobra"
)

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Diff two versions of a secret",
	Long:  "Show a unified diff between two versions of a secret.\nVersions can be selected interactively or via --a and --b flags.",
	RunE:  runDiff,
}

func init() {
	diffCmd.Flags().StringP("project", "p", "", "GCP project ID")
	diffCmd.Flags().StringP("secret", "s", "", "Secret name")
	diffCmd.Flags().StringP("a", "a", "", "First version to compare")
	diffCmd.Flags().StringP("b", "b", "", "Second version to compare")
}

func runDiff(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	flagProject, _ := cmd.Flags().GetString("project")
	flagSecret, _ := cmd.Flags().GetString("secret")
	verA, _ := cmd.Flags().GetString("a")
	verB, _ := cmd.Flags().GetString("b")

	projectID, err := selectProject(ctx, flagProject)
	if err != nil {
		return err
	}

	client, err := gsm.NewClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close() //nolint:errcheck

	secretName, _, err := selectSecret(ctx, client, projectID, flagSecret, false)
	if err != nil {
		return err
	}

	// If versions not provided, let the user pick interactively.
	if verA == "" || verB == "" {
		var versions []gsm.VersionInfo
		_ = spinner.New().
			Title("Loading versions...").
			Context(ctx).
			Action(func() {
				versions, err = client.ListVersions(ctx, projectID, secretName)
			}).
			Run()
		if err != nil {
			return err
		}
		if len(versions) < 2 {
			return fmt.Errorf("need at least 2 versions to diff, found %d", len(versions))
		}

		opts := make([]huh.Option[string], len(versions))
		for i, v := range versions {
			label := fmt.Sprintf("v%s  %s  %s", v.Version, v.State, v.Created.Local().Format("2006-01-02 15:04"))
			opts[i] = huh.NewOption(label, v.Version)
		}

		if verA == "" {
			if err := huh.NewForm(
				huh.NewGroup(
					huh.NewSelect[string]().
						Title("First version (older)").
						Options(opts...).
						Value(&verA),
				),
			).Run(); err != nil {
				return err
			}
		}

		if verB == "" {
			if err := huh.NewForm(
				huh.NewGroup(
					huh.NewSelect[string]().
						Title("Second version (newer)").
						Options(opts...).
						Value(&verB),
				),
			).Run(); err != nil {
				return err
			}
		}
	}

	var dataA, dataB []byte
	var fetchErr error
	_ = spinner.New().
		Title(fmt.Sprintf("Fetching versions %s and %s...", verA, verB)).
		Context(ctx).
		Action(func() {
			dataA, fetchErr = client.GetSecretVersion(ctx, projectID, secretName, verA)
			if fetchErr != nil {
				return
			}
			dataB, fetchErr = client.GetSecretVersion(ctx, projectID, secretName, verB)
		}).
		Run()
	if fetchErr != nil {
		return fetchErr
	}

	diff, err := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        difflib.SplitLines(string(dataA)),
		B:        difflib.SplitLines(string(dataB)),
		FromFile: fmt.Sprintf("v%s", verA),
		ToFile:   fmt.Sprintf("v%s", verB),
		Context:  3,
	})
	if err != nil {
		return fmt.Errorf("failed to generate diff: %w", err)
	}

	if diff == "" {
		ui.Info("No differences between versions.")
		return nil
	}

	add := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	del := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	hdr := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))

	for _, line := range strings.Split(strings.TrimSuffix(diff, "\n"), "\n") {
		switch {
		case strings.HasPrefix(line, "+++"), strings.HasPrefix(line, "---"), strings.HasPrefix(line, "@@"):
			fmt.Println(hdr.Render(line))
		case strings.HasPrefix(line, "+"):
			fmt.Println(add.Render(line))
		case strings.HasPrefix(line, "-"):
			fmt.Println(del.Render(line))
		default:
			fmt.Println(line)
		}
	}

	cacheSelection(projectID, secretName)
	return nil
}
