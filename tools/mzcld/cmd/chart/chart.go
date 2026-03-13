// Package chart implements the `mzcld chart` subcommand group.
package chart

import "github.com/spf13/cobra"

// Cmd is the `mzcld chart` parent command.
var Cmd = &cobra.Command{
	Use:   "chart",
	Short: "Work with MozCloud Helm charts",
}

func init() {
	Cmd.AddCommand(newCmd)
}
