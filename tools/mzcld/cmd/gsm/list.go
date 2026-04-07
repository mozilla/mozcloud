package gsm

import (
	"fmt"
	"strings"

	"charm.land/huh/v2/spinner"
	"github.com/charmbracelet/lipgloss"
	"github.com/mozilla/mozcloud/tools/mzcld/internal/gsm"
	"github.com/mozilla/mozcloud/tools/mzcld/internal/ui"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List secrets or secret versions",
	Long:  "Without --secret: list all secret names in a project.\nWith --secret: list all versions of that secret.",
	RunE:  runList,
}

func init() {
	listCmd.Flags().StringP("project", "p", "", "GCP project ID")
	listCmd.Flags().StringP("secret", "s", "", "Secret name (list versions of this secret)")
}

func runList(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	flagProject, _ := cmd.Flags().GetString("project")
	flagSecret, _ := cmd.Flags().GetString("secret")

	projectID, err := selectProject(ctx, flagProject)
	if err != nil {
		return err
	}

	// If no secret specified, list all secret names.
	if flagSecret == "" {
		var names []string
		_ = spinner.New().
			Title("Loading secrets...").
			Context(ctx).
			Action(func() {
				names, err = gsm.ListSecrets(ctx, projectID)
			}).
			Run()
		if err != nil {
			return err
		}
		if len(names) == 0 {
			ui.Warn("No secrets found in project " + projectID)
			return nil
		}
		ui.Header(fmt.Sprintf("Secrets in %s (%d)", projectID, len(names)))
		for _, n := range names {
			fmt.Println("  " + n)
		}
		cacheSelection(projectID, "")
		return nil
	}

	// List versions of a specific secret.
	var versions []gsm.VersionInfo
	_ = spinner.New().
		Title("Loading versions...").
		Context(ctx).
		Action(func() {
			versions, err = gsm.ListVersions(ctx, projectID, flagSecret)
		}).
		Run()
	if err != nil {
		return err
	}
	if len(versions) == 0 {
		ui.Warn("No versions found for " + flagSecret)
		return nil
	}

	label := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Width(12)
	state := lipgloss.NewStyle().Width(12)

	ui.Header(fmt.Sprintf("Versions of %s (%d)", flagSecret, len(versions)))
	for _, v := range versions {
		stateStyle := state
		switch {
		case strings.Contains(v.State, "ENABLED"):
			stateStyle = stateStyle.Foreground(lipgloss.Color("2"))
		case strings.Contains(v.State, "DESTROYED"):
			stateStyle = stateStyle.Foreground(lipgloss.Color("1"))
		default:
			stateStyle = stateStyle.Foreground(lipgloss.Color("3"))
		}
		fmt.Println(
			label.Render("  v"+v.Version) +
				stateStyle.Render(v.State) +
				lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render(v.Created.Local().Format("2006-01-02 15:04")),
		)
	}

	cacheSelection(projectID, flagSecret)
	return nil
}
