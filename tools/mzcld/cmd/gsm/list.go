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
	Short: "List versions of a secret",
	Long:  "Select a project and secret, then list all versions with state and creation time.",
	RunE:  runList,
}

func init() {
	listCmd.Flags().StringP("project", "p", "", "GCP project ID")
	listCmd.Flags().StringP("secret", "s", "", "Secret name")
}

func runList(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	flagProject, _ := cmd.Flags().GetString("project")
	flagSecret, _ := cmd.Flags().GetString("secret")

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
	if len(versions) == 0 {
		ui.Warn("No versions found for " + secretName)
		return nil
	}

	label := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Width(12)
	state := lipgloss.NewStyle().Width(12)

	ui.Header(fmt.Sprintf("Versions of %s (%d)", secretName, len(versions)))
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

	cacheSelection(projectID, secretName)
	return nil
}
