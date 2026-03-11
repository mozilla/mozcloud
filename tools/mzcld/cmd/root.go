// Package cmd implements the mzcld command-line interface.
package cmd

import (
	"os"

	"github.com/mozilla/mozcloud/tools/mzcld/cmd/claude"
	mzinit "github.com/mozilla/mozcloud/tools/mzcld/cmd/init"
	"github.com/mozilla/mozcloud/tools/mzcld/internal/ui"
	"github.com/spf13/cobra"
)

var (
	debugFlag   bool
	noColorFlag bool

	// Version is set at build time via -ldflags.
	Version = "dev"
)

var rootCmd = &cobra.Command{
	Use:     "mzcld",
	Short:   "MozCloud CLI",
	Long:    "mzcld is the unified CLI for interacting with the MozCloud platform.",
	Version: Version,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if noColorFlag {
			os.Setenv("NO_COLOR", "1")
		}
		ui.SetDebug(debugFlag)
	},
	SilenceUsage: true,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&debugFlag, "debug", "d", false, "Enable verbose debug output")
	rootCmd.PersistentFlags().BoolVar(&noColorFlag, "no-color", false, "Disable colored output")
	rootCmd.PersistentFlags().SortFlags = false

	rootCmd.AddCommand(mzinit.Cmd)
	rootCmd.AddCommand(claude.Cmd)
}
