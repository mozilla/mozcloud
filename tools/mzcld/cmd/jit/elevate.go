package jit

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"charm.land/huh/v2"
	"github.com/charmbracelet/lipgloss"
	"github.com/mozilla/mozcloud/tools/mzcld/internal/gcp"
	"github.com/mozilla/mozcloud/tools/mzcld/internal/pam"
	"github.com/mozilla/mozcloud/tools/mzcld/internal/ui"
	"github.com/spf13/cobra"
)

const bucketName = "moz-fx-jit-integration-global-user-data"

var elevateCmd = &cobra.Command{
	Use:   "elevate [justification]",
	Short: "Request a new JIT access grant",
	Long:  "Request temporary elevated access to a GCP project via Privileged Access Manager.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runElevate,
}

func runElevate(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// --- Auth ----------------------------------------------------------------
	email, err := gcp.EnsureAuthenticated()
	if err != nil {
		return err
	}
	ui.Debug("Authenticated as " + email)

	// --- Justification -------------------------------------------------------
	justification := ""
	if len(args) > 0 {
		justification = args[0]
	}
	if justification == "" {
		if err := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Justification").
					Description("Describe why you need elevated access.").
					Validate(func(s string) error {
						if strings.TrimSpace(s) == "" {
							return fmt.Errorf("justification is required")
						}
						return nil
					}).
					Value(&justification),
			),
		).Run(); err != nil {
			return err
		}
	}

	// --- Load entitlements ---------------------------------------------------
	ui.Info("Loading entitlements...")
	entitlements, err := gcp.LoadEntitlements(ctx, bucketName, email)
	if err != nil {
		return authError(err)
	}
	if len(entitlements) == 0 {
		return fmt.Errorf("no entitlements found for %s", email)
	}

	// Build sorted deduplicated appcode list
	appcodeSet := make(map[string]struct{})
	for _, e := range entitlements {
		appcodeSet[e.Appcode] = struct{}{}
	}
	appcodes := make([]string, 0, len(appcodeSet))
	for ac := range appcodeSet {
		appcodes = append(appcodes, ac)
	}
	sort.Strings(appcodes)

	// --- Load project cache for last-choice shortcut -------------------------
	pc, err := gcp.Load()
	if err != nil {
		ui.Debug("could not load project cache: " + err.Error())
	}

	var chosen gcp.UserEntitlement

	if pc.LastChoice.Appcode != "" {
		// Offer the quick-return option first
		const optLast = "last"
		const optNew = "new"
		var quickChoice string

		lastLabel := fmt.Sprintf("↩ last: %s / %s (%s)",
			pc.LastChoice.Appcode,
			pc.LastChoice.Entitlement,
			pc.LastChoice.Realm,
		)

		if err := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Entitlement").
					Options(
						huh.NewOption(lastLabel, optLast),
						huh.NewOption("Choose different...", optNew),
					).
					Value(&quickChoice),
			),
		).Run(); err != nil {
			return err
		}

		if quickChoice == optLast {
			chosen = pc.LastChoice
		}
	}

	if chosen.Appcode == "" {
		// --- Select appcode --------------------------------------------------
		var selectedAppcode string
		appcodeOpts := make([]huh.Option[string], len(appcodes))
		for i, ac := range appcodes {
			appcodeOpts[i] = huh.NewOption(ac, ac)
		}
		if err := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("App code").
					Options(appcodeOpts...).
					Value(&selectedAppcode),
			),
		).Run(); err != nil {
			return err
		}

		// --- Select entitlement (grouped by realm) ---------------------------
		var filtered []gcp.UserEntitlement
		for _, e := range entitlements {
			if e.Appcode == selectedAppcode {
				filtered = append(filtered, e)
			}
		}

		// Sort: prod before nonprod, then by entitlement name
		sort.Slice(filtered, func(i, j int) bool {
			if filtered[i].Realm != filtered[j].Realm {
				return filtered[i].Realm < filtered[j].Realm
			}
			return filtered[i].Entitlement < filtered[j].Entitlement
		})

		entOpts := make([]huh.Option[string], len(filtered))
		for i, e := range filtered {
			label := fmt.Sprintf("%s  (%s)", e.Entitlement, e.Realm)
			entOpts[i] = huh.NewOption(label, e.Entitlement+"|"+e.Realm+"|"+e.ProjectID)
		}

		var entKey string
		if err := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Entitlement").
					Options(entOpts...).
					Value(&entKey),
			),
		).Run(); err != nil {
			return err
		}

		// Decode back from key
		for _, e := range filtered {
			if e.Entitlement+"|"+e.Realm+"|"+e.ProjectID == entKey {
				chosen = e
				break
			}
		}
	}

	// --- Select duration -----------------------------------------------------
	var duration time.Duration
	if err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[time.Duration]().
				Title("Duration").
				Options(
					huh.NewOption("1 hour", time.Hour),
					huh.NewOption("2 hours", 2*time.Hour),
					huh.NewOption("4 hours", 4*time.Hour),
					huh.NewOption("8 hours", 8*time.Hour),
				).
				Value(&duration),
		),
	).Run(); err != nil {
		return err
	}

	// --- Create grant --------------------------------------------------------
	ui.Info("Requesting grant...")
	grant, err := pam.CreateGrant(ctx, chosen.ProjectID, chosen.Entitlement, justification, duration)
	if err != nil {
		return authError(err)
	}

	// --- Persist grant -------------------------------------------------------
	grants, _ := loadGrants(ctx)
	grants = append(grants, grant)
	if err := saveGrants(grants); err != nil {
		ui.Warn("failed to save grant to cache: " + err.Error())
	}

	// --- Update last choice --------------------------------------------------
	pc.LastChoice = chosen
	if err := pc.Save(); err != nil {
		ui.Warn("failed to save project cache: " + err.Error())
	}

	// --- Display result ------------------------------------------------------
	printGrantCard(grant)
	return nil
}

// printGrantCard renders a styled summary of a newly-created grant.
func printGrantCard(g *pam.Grant) {
	project, entName, err := pam.ExtractParts(g)
	if err != nil {
		ui.Error("could not parse grant: " + err.Error())
		return
	}

	bold := lipgloss.NewStyle().Bold(true)
	label := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Width(22)
	value := lipgloss.NewStyle()

	row := func(k, v string) string {
		return label.Render(k) + value.Render(v)
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("2")).
		Padding(0, 1).
		Render(strings.Join([]string{
			bold.Render("Grant created"),
			"",
			row("Entitlement:", entName),
			row("Project:", project),
			row("State:", pam.StateLabel(g)),
			row("Time remaining:", formatRemaining(pam.TimeRemaining(g))),
		}, "\n"))

	fmt.Println()
	fmt.Println(box)

	if roles := pam.Roles(g); len(roles) > 0 {
		fmt.Println()
		ui.Info("Roles granted:")
		for _, r := range roles {
			ui.Dim("  • " + r)
		}
	}
	fmt.Println()
}

func formatRemaining(d time.Duration) string {
	if d < 0 {
		return "expired"
	}
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d", m, s)
}

func authError(err error) error {
	msg := err.Error()
	if strings.Contains(msg, "Unauthenticated") || strings.Contains(msg, "reauth") {
		return fmt.Errorf("%w\n\nRun: gcloud auth application-default login", err)
	}
	return err
}
