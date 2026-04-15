package jit

import (
	"fmt"

	"charm.land/huh/v2"
	"github.com/mozilla/mozcloud/tools/mzcld/internal/pam"
	"github.com/mozilla/mozcloud/tools/mzcld/internal/ui"
	"github.com/spf13/cobra"
)

var revokeCmd = &cobra.Command{
	Use:   "revoke",
	Short: "Revoke an active JIT grant",
	Long:  "Select an active JIT grant and revoke it via Privileged Access Manager.",
	RunE:  runRevoke,
}

func runRevoke(cmd *cobra.Command, _ []string) error {
	grants, err := loadGrants(cmd.Context())
	if err != nil {
		return err
	}

	if len(grants) == 0 {
		ui.Info("No active grants to revoke.")
		return nil
	}

	// Build select options
	opts := make([]huh.Option[int], len(grants))
	for i, g := range grants {
		project, entName, err := pam.ExtractParts(g)
		if err != nil {
			opts[i] = huh.NewOption(g.GetName(), i)
			continue
		}
		label := fmt.Sprintf("%s  (%s)  %s", entName, project, formatRemaining(pam.TimeRemaining(g)))
		opts[i] = huh.NewOption(label, i)
	}

	var idx int
	if err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[int]().
				Title("Select grant to revoke").
				Options(opts...).
				Value(&idx),
		),
	).Run(); err != nil {
		return err
	}

	if idx < 0 || idx >= len(grants) {
		return fmt.Errorf("invalid selection")
	}

	target := grants[idx]

	if err := pam.RevokeGrant(cmd.Context(), target.GetName()); err != nil {
		return authError(err)
	}

	// Remove from local list
	updated := make([]*pam.Grant, 0, len(grants)-1)
	for i, g := range grants {
		if i != idx {
			updated = append(updated, g)
		}
	}
	if err := saveGrants(updated); err != nil {
		ui.Warn("grant revoked, but failed to update local cache: " + err.Error())
	}

	ui.Success("Grant revoked.")
	return nil
}
