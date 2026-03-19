package jit

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/mozilla/mozcloud/tools/mzcld/internal/pam"
	"github.com/mozilla/mozcloud/tools/mzcld/internal/ui"
	"github.com/spf13/cobra"
)

var stateCmd = &cobra.Command{
	Use:   "state",
	Short: "View active JIT grants",
	Long:  "Display current JIT grants and optionally view detailed information about one.",
	RunE:  runState,
}

func runState(cmd *cobra.Command, _ []string) error {
	grants, err := loadGrants(cmd.Context())
	if err != nil {
		return err
	}

	if len(grants) == 0 {
		ui.Info("No active grants.")
		return nil
	}

	// --- Render summary blocks -----------------------------------------------
	ui.Header("Active JIT Grants")
	for _, g := range grants {
		printGrantSummary(g)
	}

	// --- Offer detail view ---------------------------------------------------
	const exitKey = "__exit__"
	opts := make([]huh.Option[string], 0, len(grants)+1)
	for i, g := range grants {
		project, entName, err := pam.ExtractParts(g)
		if err != nil {
			continue
		}
		label := fmt.Sprintf("%s  (%s)  %s", entName, project, formatRemaining(pam.TimeRemaining(g)))
		opts = append(opts, huh.NewOption(label, fmt.Sprintf("%d", i)))
	}
	opts = append(opts, huh.NewOption("Exit", exitKey))

	var selected string
	if err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("View grant details").
				Options(opts...).
				Value(&selected),
		),
	).Run(); err != nil {
		return err
	}

	if selected == exitKey {
		return nil
	}

	// Parse selected index and show detailed view
	var idx int
	if _, err := fmt.Sscanf(selected, "%d", &idx); err != nil || idx < 0 || idx >= len(grants) {
		return nil
	}
	printGrantDetail(grants[idx])
	return nil
}

// printGrantSummary renders a short lipgloss block for a single grant.
func printGrantSummary(g *pam.Grant) {
	project, entName, err := pam.ExtractParts(g)
	if err != nil {
		return
	}

	state := pam.StateLabel(g)

	stateColor := lipgloss.Color("2") // green
	if pam.IsApprovalAwaited(g) {
		stateColor = lipgloss.Color("3") // yellow
	}

	label := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Width(18)
	stateStyle := lipgloss.NewStyle().Foreground(stateColor).Bold(true)

	block := strings.Join([]string{
		label.Render("Entitlement:") + entName,
		label.Render("Project:") + project,
		label.Render("State:") + stateStyle.Render(state),
		label.Render("Time remaining:") + formatRemaining(pam.TimeRemaining(g)),
	}, "\n")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("4")).
		Padding(0, 1).
		Render(block)

	fmt.Println(box)
}

// printGrantDetail renders a full detail view for a single grant.
func printGrantDetail(g *pam.Grant) {
	project, entName, err := pam.ExtractParts(g)
	if err != nil {
		ui.Error("could not parse grant: " + err.Error())
		return
	}

	remaining := pam.TimeRemaining(g)
	endTime := time.Now().Add(remaining)
	startTime := endTime.Add(-g.GetRequestedDuration().AsDuration())

	bold := lipgloss.NewStyle().Bold(true)
	label := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Width(24)

	rows := []string{
		bold.Render("Grant Details"),
		"",
		label.Render("Entitlement:") + entName,
		label.Render("Project:") + project,
		label.Render("State:") + pam.StateLabel(g),
		label.Render("Requester:") + g.GetRequester(),
		label.Render("Justification:") + pam.GetJustification(g),
		label.Render("Start time:") + startTime.Format(time.RFC822),
		label.Render("End time:") + endTime.Format(time.RFC822),
		label.Render("Time remaining:") + formatRemaining(remaining),
	}

	if roles := pam.Roles(g); len(roles) > 0 {
		rows = append(rows, "")
		rows = append(rows, label.Render("Roles:"))
		for _, r := range roles {
			rows = append(rows, "  • "+r)
		}
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("5")).
		Padding(0, 1).
		Render(strings.Join(rows, "\n"))

	fmt.Println()
	fmt.Println(box)
	fmt.Println()
}
