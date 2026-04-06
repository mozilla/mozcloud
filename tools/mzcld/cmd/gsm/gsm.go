// Package gsm implements the `mzcld gsm` command group for managing
// Google Secret Manager secrets.
package gsm

import "github.com/spf13/cobra"

// Cmd is the root command for secret management.
var Cmd = &cobra.Command{
	Use:   "gsm",
	Short: "Manage Google Secret Manager secrets",
	Long:  "View, edit, list, and diff secrets stored in Google Secret Manager.",
}

func init() {
	Cmd.AddCommand(editCmd)
	Cmd.AddCommand(viewCmd)
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(diffCmd)
}
