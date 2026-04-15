// Package cmd implements the mzcld command-line interface.
package cmd

import (
	"fmt"
	"os"

	"github.com/mozilla/mozcloud/tools/mzcld/cmd/bastion"
	"github.com/mozilla/mozcloud/tools/mzcld/cmd/chart"
	"github.com/mozilla/mozcloud/tools/mzcld/cmd/claude"
	"github.com/mozilla/mozcloud/tools/mzcld/cmd/gsm"
	mzinit "github.com/mozilla/mozcloud/tools/mzcld/cmd/init"
	"github.com/mozilla/mozcloud/tools/mzcld/cmd/iap"
	"github.com/mozilla/mozcloud/tools/mzcld/cmd/jit"
	"github.com/mozilla/mozcloud/tools/mzcld/internal/gcp"
	"github.com/mozilla/mozcloud/tools/mzcld/internal/ui"
	"github.com/spf13/cobra"
)

var (
	debugFlag    bool
	noColorFlag  bool
	authModeFlag string

	// Version is set at build time via -ldflags.
	Version = "dev"
)

var rootCmd = &cobra.Command{
	Use:     "mzcld",
	Short:   "MozCloud CLI",
	Long:    "mzcld is the unified CLI for interacting with the MozCloud platform.",
	Version: Version,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if noColorFlag {
			os.Setenv("NO_COLOR", "1")
		}
		ui.SetDebug(debugFlag)
		switch gcp.AuthMode(authModeFlag) {
		case gcp.AuthModeGcloud, gcp.AuthModeADC:
			gcp.SetAuthMode(gcp.AuthMode(authModeFlag))
		default:
			return fmt.Errorf("invalid --auth-mode %q: must be \"gcloud\" or \"adc\"", authModeFlag)
		}
		return nil
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
	rootCmd.PersistentFlags().StringVar(&authModeFlag, "auth-mode", "gcloud", "Auth mode: gcloud (default, RAPT-safe) or adc (for CI/service accounts)")
	rootCmd.PersistentFlags().SortFlags = false

	rootCmd.AddCommand(mzinit.Cmd)
	rootCmd.AddCommand(claude.Cmd)
	rootCmd.AddCommand(chart.Cmd)
	rootCmd.AddCommand(jit.Cmd)
	rootCmd.AddCommand(iap.Cmd)
	rootCmd.AddCommand(bastion.Cmd)
	rootCmd.AddCommand(gsm.Cmd)
}
