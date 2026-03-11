// Package claude implements the `mzcld claude` subcommand group.
package claude

import "github.com/spf13/cobra"

// Cmd is the `mzcld claude` parent command.
var Cmd = &cobra.Command{
	Use:   "claude",
	Short: "Manage Claude Code skills, agents, and MCP servers for MozCloud",
}

func init() {
	Cmd.AddCommand(installCmd)
}
