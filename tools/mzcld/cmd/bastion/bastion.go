// Package bastion implements the `mzcld bastion` command for connecting to
// Mozilla GCP bastion hosts via SSH tunnel.
package bastion

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"charm.land/huh/v2"
	"github.com/mozilla/mozcloud/tools/mzcld/internal/bastion"
	"github.com/mozilla/mozcloud/tools/mzcld/internal/ui"
	"github.com/spf13/cobra"
)

// Cmd is the `bastion` command.
var Cmd = &cobra.Command{
	Use:   "bastion",
	Short: "Connect to a Mozilla bastion host via SSH SOCKS proxy",
	Long: `Opens an SSH tunnel through a GCP IAP bastion host, creating a SOCKS proxy
on a well-known local port that maps to the selected realm and region.`,
	RunE: runBastion,
}

func runBastion(cmd *cobra.Command, _ []string) error {
	// --- Load cached selection -----------------------------------------------
	cached, err := bastion.Load()
	if err != nil {
		ui.Debug("failed to load bastion cache: " + err.Error())
	}

	var realm, region string

	if cached != nil && cached.Realm != "" && cached.Region != "" {
		// Offer the quick-return option
		const optLast = "last"
		const optNew = "new"
		var choice string

		lastLabel := fmt.Sprintf("↩ last: %s/%s", cached.Realm, cached.Region)

		if err := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Bastion connection").
					Options(
						huh.NewOption(lastLabel, optLast),
						huh.NewOption("Choose different...", optNew),
					).
					Value(&choice),
			),
		).Run(); err != nil {
			return err
		}

		if choice == optLast {
			realm = cached.Realm
			region = cached.Region
		}
	}

	if realm == "" || region == "" {
		// --- Select realm ----------------------------------------------------
		realmOpts := make([]huh.Option[string], len(bastion.BastionRealms))
		for i, r := range bastion.BastionRealms {
			realmOpts[i] = huh.NewOption(r, r)
		}
		if err := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Realm").
					Options(realmOpts...).
					Value(&realm),
			),
		).Run(); err != nil {
			return err
		}

		// --- Select region ---------------------------------------------------
		regionOpts := make([]huh.Option[string], len(bastion.BastionRegions))
		for i, r := range bastion.BastionRegions {
			regionOpts[i] = huh.NewOption(r, r)
		}
		if err := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Region").
					Options(regionOpts...).
					Value(&region),
			),
		).Run(); err != nil {
			return err
		}
	}

	// --- Save selection ------------------------------------------------------
	if err := bastion.Save(&bastion.BastionCache{
		Realm:      realm,
		Region:     region,
		LastAccess: time.Now().Format(time.RFC3339),
	}); err != nil {
		ui.Warn("failed to save bastion cache: " + err.Error())
	}

	// --- Resolve connection parameters ---------------------------------------
	key := realm + ":" + region
	port, ok := bastion.BastionPorts[key]
	if !ok {
		return fmt.Errorf("unknown realm/region combination: %s", key)
	}

	zone, ok := bastion.BastionZone[region]
	if !ok {
		return fmt.Errorf("unknown region: %s", region)
	}

	project := fmt.Sprintf("moz-fx-bastion-%s-global", realm)
	bastionHost := fmt.Sprintf("bastion-%s", region)

	ui.Info(fmt.Sprintf("Connecting to %s in %s/%s on SOCKS port %s", bastionHost, realm, region, port))

	// --- Exec gcloud ssh -----------------------------------------------------
	//nolint:gosec
	gcloudCmd := exec.Command("gcloud", "compute", "ssh",
		"--project", project,
		"--tunnel-through-iap",
		"--zone", zone,
		bastionHost,
		"--",
		"-D", port,
		"-N",
	)
	gcloudCmd.Stdin = os.Stdin
	gcloudCmd.Stdout = os.Stdout
	gcloudCmd.Stderr = os.Stderr

	if err := gcloudCmd.Run(); err != nil {
		return fmt.Errorf("gcloud compute ssh exited with error: %w", err)
	}
	return nil
}
