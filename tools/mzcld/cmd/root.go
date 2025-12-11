package cmd

import (
	"github.com/mozilla/mozcloud/tools/mzcld/cmd/argo"
	"github.com/mozilla/mozcloud/tools/mzcld/cmd/iap"
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:     "mzcld",
		Short:   "MozCloud CLI utilities",
		Long:    "mzcld provides a common set of tools for performing various MozCloud maintenance tasks",
		Version: getVersion(),
	}

	// Include Argo sub-command
	root.AddCommand(argo.NewArgoRootCmd())

	// Include IAP sub-command
	root.AddCommand(iap.NewIAPCmd())

	return root
}
