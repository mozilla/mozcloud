// Package jit implements the `mzcld jit` command group for managing Just-In-Time
// access entitlements via GCP Privileged Access Manager.
package jit

import (
	"context"
	"fmt"

	"github.com/mozilla/mozcloud/tools/mzcld/internal/cache"
	"github.com/mozilla/mozcloud/tools/mzcld/internal/pam"
	"github.com/spf13/cobra"
)

const grantsFile = "grants.json"

// Cmd is the root `jit` command.
var Cmd = &cobra.Command{
	Use:   "jit",
	Short: "Manage JIT (Just-In-Time) access entitlements",
	Long:  "Request, view, and revoke temporary elevated access to GCP projects via Privileged Access Manager.",
}

func init() {
	Cmd.AddCommand(elevateCmd, stateCmd, revokeCmd)
}

// loadGrants reads the grants list from cache, prunes expired grants, and
// refreshes any APPROVAL_AWAITED grants from the PAM API. The list is saved
// back to disk if any changes were made.
func loadGrants(ctx context.Context) ([]*pam.Grant, error) {
	if !cache.Exists(grantsFile) {
		return []*pam.Grant{}, nil
	}

	data, err := cache.Load(grantsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load grants: %w", err)
	}

	grants, err := pam.UnmarshalGrants(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse grants: %w", err)
	}

	dirty := false

	var kept []*pam.Grant
	for i, g := range grants {
		if pam.IsExpired(g) {
			dirty = true
			continue
		}

		if pam.IsApprovalAwaited(g) {
			fresh, err := pam.GetGrant(ctx, g.GetName())
			if err == nil && !pam.IsApprovalAwaited(fresh) {
				grants[i] = fresh
				dirty = true
			}
		}

		kept = append(kept, grants[i])
	}

	if dirty {
		if err := saveGrants(kept); err != nil {
			return nil, fmt.Errorf("failed to save updated grants: %w", err)
		}
	}

	return kept, nil
}

// saveGrants persists grants to ~/.mzcld/grants.json.
func saveGrants(grants []*pam.Grant) error {
	data, err := pam.MarshalGrants(grants)
	if err != nil {
		return fmt.Errorf("failed to marshal grants: %w", err)
	}
	return cache.Save(grantsFile, data)
}
