package argo

import (
	"context"
	"fmt"
	"time"

	appsv1alpha1 "github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
	"github.com/mozilla/mozcloud/tools/mzcld/internal/kube"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewArgoRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "argo",
		Short: "ArgoCD utilities",
		Long:  "Utilities for managing and accessing ArgoCD",
	}

	// Add subcommands
	cmd.AddCommand(NewArgoMaintenanceCmd())
	cmd.AddCommand(NewArgoAppCmd())
	cmd.AddCommand(NewArgoLoginCmd())
	cmd.AddCommand(NewArgoCLICmd())

	return cmd
}

func NewArgoMaintenanceCmd() *cobra.Command {
	var (
		namespace string
		stateFile string
		selector  string
		force     bool
		disable   bool
		restore   bool
		timeout   time.Duration

		hasApps bool // set in PreRunE
	)

	cmd := &cobra.Command{
		Use:   "maintenance",
		Short: "Inspect and manage Argo CD Applications sync state",
		Long: `Inspect and manage Argo CD Applications.

Default (no flags):
  - Create a snapshot of Application AutoSync state
  - Print a status table (AutoSync + active sync)
  - No changes are made to the cluster

Actions:
  --disable-sync   Disable AutoSync for matching Applications
  --restore-sync   Restore AutoSync state from snapshot

If active sync operations are detected, the command waits up to --timeout
(default: 2m) for them to complete before continuing.`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Check once whether any Applications exist; if not, print message and skip RunE work.
			ctx := context.Background()

			if namespace == "" {
				namespace = "argocd"
			}

			k8sClient, _, err := kube.NewClient()
			if err != nil {
				return err
			}

			var appList appsv1alpha1.ApplicationList
			opts := []client.ListOption{client.InNamespace(namespace)}

			if selector != "" {
				sel, err := labels.Parse(selector)
				if err != nil {
					return fmt.Errorf("invalid label selector: %w", err)
				}
				opts = append(opts, client.MatchingLabelsSelector{Selector: sel})
			}

			if err := k8sClient.List(ctx, &appList, opts...); err != nil {
				return fmt.Errorf("list applications: %w", err)
			}

			if len(appList.Items) == 0 {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(),
					"No Argo CD Applications found in namespace %q\n", namespace); err != nil {
					return fmt.Errorf("failed writing to stdout: %w", err)
				}
				hasApps = false
				// Return nil so Cobra exits cleanly, but RunE will check hasApps and no-op.
				return nil
			}

			hasApps = true
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// If PreRunE determined there are no apps, do nothing further.
			if !hasApps {
				return nil
			}

			if disable && restore {
				return fmt.Errorf("only one of --disable-sync or --restore-sync may be provided")
			}

			if disable {
				return runMaintenanceEnable(cmd, namespace, stateFile, selector, force, timeout)
			}
			if restore {
				return runMaintenanceRestore(cmd, namespace, stateFile)
			}

			// Default: snapshot + read-only table
			if err := runMaintenanceSnapshotOnly(cmd, namespace, stateFile, selector, force, timeout); err != nil {
				return err
			}
			return runMaintenanceDryRun(cmd, namespace, selector)
		},
	}

	cmd.Flags().StringVarP(&namespace, "namespace", "n", "argocd", "Namespace containing Argo CD Applications")
	cmd.Flags().StringVar(&stateFile, "state-file", "", "Path to snapshot file (default: ~/.local/state/mzcld/argo-maintenance/<namespace>-snapshot.json)")
	cmd.Flags().StringVar(&selector, "selector", "", "Label selector to filter Applications")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing snapshot file")
	cmd.Flags().BoolVar(&disable, "disable-sync", false, "Disable AutoSync for matching Applications (save snapshot first)")
	cmd.Flags().BoolVar(&restore, "restore-sync", false, "Restore AutoSync state from snapshot")
	cmd.Flags().DurationVar(&timeout, "timeout", 2*time.Minute, "Maximum time to wait for active syncs to complete before giving up")

	return cmd
}
