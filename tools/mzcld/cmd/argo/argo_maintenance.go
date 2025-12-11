package argo

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	appsv1alpha1 "github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
	"github.com/mozilla/mozcloud/tools/mzcld/internal/kube"
	"github.com/mozilla/mozcloud/tools/mzcld/internal/state"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func runMaintenanceEnable(cmd *cobra.Command, namespace, stateFile, selector string, force bool, timeout time.Duration) error {
	ctx := context.Background()

	if namespace == "" {
		namespace = "argocd"
	}

	if stateFile == "" {
		home, _ := os.UserHomeDir()
		stateFile = filepath.Join(home, ".local", "state", "mzcld", "argo-maintenance", namespace+"-snapshot.json")
	}

	// Avoid accidentally overwriting an existing snapshot unless forced
	if !force {
		if _, err := os.Stat(stateFile); err == nil {
			return fmt.Errorf("state file %s already exists (use --force to overwrite)", stateFile)
		}
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

	// Check for active syncs and wait for them to clear before proceeding.
	active := collectActiveSyncs(&appList)
	if len(active) > 0 {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(),
			"Found %d Applications with active sync operations; waiting for them to complete (timeout: %s):\n", len(active), timeout); err != nil {
			return fmt.Errorf("failed writing to stdout: %w", err)
		}
		for _, name := range active {
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "- %s\n", name); err != nil {
				return fmt.Errorf("failed writing to stdout: %w", err)
			}
		}

		if err := waitForActiveSyncsClear(ctx, k8sClient, cmd, timeout, opts...); err != nil {
			return err
		}
	}

	snap := &state.Snapshot{
		Cluster:   "",
		Namespace: namespace,
		SavedAt:   time.Now().UTC(),
	}

	// Capture current syncPolicy
	for i := range appList.Items {
		app := &appList.Items[i]

		raw := json.RawMessage("null")
		if app.Spec.SyncPolicy != nil {
			b, err := json.Marshal(app.Spec.SyncPolicy)
			if err != nil {
				return fmt.Errorf("marshal syncPolicy %s: %w", app.Name, err)
			}
			raw = b
		}

		snap.Applications = append(snap.Applications, state.ApplicationSyncState{
			Name:       app.Name,
			Namespace:  app.Namespace,
			SyncPolicy: raw,
		})
	}

	// Write snapshot to disk
	if err := state.Save(stateFile, snap); err != nil {
		return fmt.Errorf("save snapshot: %w", err)
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(),
		"Saved state for %d applications to %s\n", len(snap.Applications), stateFile); err != nil {
		return fmt.Errorf("failed writing to stdout: %w", err)
	}

	// Disable autosync
	changed := 0
	for i := range appList.Items {
		app := &appList.Items[i]

		if app.Spec.SyncPolicy == nil || app.Spec.SyncPolicy.Automated == nil {
			if _, err := fmt.Fprintf(cmd.OutOrStdout(),
				"- %s: AutoSync already disabled\n", app.Name); err != nil {
				return fmt.Errorf("failed writing to stdout: %w", err)
			}
			continue
		}

		// Remove Automated block
		app.Spec.SyncPolicy.Automated = nil

		// Optionally clean up empty syncPolicy
		if app.Spec.SyncPolicy.SyncOptions == nil && app.Spec.SyncPolicy.Retry == nil {
			app.Spec.SyncPolicy = nil
		}

		if err := k8sClient.Update(ctx, app); err != nil {
			return fmt.Errorf("update application %s: %w", app.Name, err)
		}

		if _, err := fmt.Fprintf(cmd.OutOrStdout(),
			"- %s: AutoSync disabled\n", app.Name); err != nil {
			return fmt.Errorf("failed writing to stdout: %w", err)
		}
		changed++
	}

	if _, err := fmt.Fprintf(cmd.OutOrStdout(),
		"Maintenance mode enabled. Applications changed: %d\n", changed); err != nil {
		return fmt.Errorf("failed writing to stdout: %w", err)
	}
	return nil
}

func runMaintenanceRestore(cmd *cobra.Command, namespace, stateFile string) error {
	if namespace == "" {
		namespace = "argocd"
	}

	if stateFile == "" {
		home, _ := os.UserHomeDir()
		stateFile = filepath.Join(home, ".local", "state", "mzcld", "argo-maintenance", namespace+"-snapshot.json")
	}

	snap, err := state.Load(stateFile)
	if err != nil {
		return fmt.Errorf("load snapshot: %w", err)
	}

	k8sClient, _, err := kube.NewClient()
	if err != nil {
		return err
	}

	ctx := context.Background()
	restored := 0
	skipped := 0

	for _, appState := range snap.Applications {
		var app appsv1alpha1.Application
		key := types.NamespacedName{Namespace: appState.Namespace, Name: appState.Name}

		if err := k8sClient.Get(ctx, key, &app); err != nil {
			if _, errw := fmt.Fprintf(cmd.OutOrStdout(),
				"- %s/%s: SKIP (not found: %v)\n", appState.Namespace, appState.Name, err); errw != nil {
				return fmt.Errorf("failed writing to stdout: %w", errw)
			}
			skipped++
			continue
		}

		if string(appState.SyncPolicy) == "null" {
			app.Spec.SyncPolicy = nil
		} else {
			var sp appsv1alpha1.SyncPolicy
			if err := json.Unmarshal(appState.SyncPolicy, &sp); err != nil {
				return fmt.Errorf("unmarshal syncPolicy for %s: %w", appState.Name, err)
			}
			app.Spec.SyncPolicy = &sp
		}

		if err := k8sClient.Update(ctx, &app); err != nil {
			return fmt.Errorf("update application %s: %w", app.Name, err)
		}

		if _, err := fmt.Fprintf(cmd.OutOrStdout(),
			"- %s/%s restored\n", appState.Namespace, appState.Name); err != nil {
			return fmt.Errorf("failed writing to stdout: %w", err)
		}
		restored++
	}

	if _, err := fmt.Fprintf(cmd.OutOrStdout(),
		"Restore complete. Restored: %d, skipped: %d\n", restored, skipped); err != nil {
		return fmt.Errorf("failed writing to stdout: %w", err)
	}
	return nil
}

// runMaintenanceSnapshotOnly writes the snapshot file but does not change any Applications.
// This is used by the default (non-mutating) mode; it is not exposed as a separate flag.
func runMaintenanceSnapshotOnly(cmd *cobra.Command, namespace, stateFile, selector string, force bool, timeout time.Duration) error {
	ctx := context.Background()

	if namespace == "" {
		namespace = "argocd"
	}

	if stateFile == "" {
		home, _ := os.UserHomeDir()
		stateFile = filepath.Join(home, ".local", "state", "mzcld", "argo-maintenance", namespace+"-snapshot.json")
	}

	if !force {
		if _, err := os.Stat(stateFile); err == nil {
			return fmt.Errorf("state file %s already exists (use --force to overwrite)", stateFile)
		}
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

	// Check for active syncs and wait for them to clear before snapshotting.
	active := collectActiveSyncs(&appList)
	if len(active) > 0 {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(),
			"Found %d Applications with active sync operations; waiting for them to complete (timeout: %s):\n", len(active), timeout); err != nil {
			return fmt.Errorf("failed writing to stdout: %w", err)
		}
		for _, name := range active {
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "- %s\n", name); err != nil {
				return fmt.Errorf("failed writing to stdout: %w", err)
			}
		}

		if err := waitForActiveSyncsClear(ctx, k8sClient, cmd, timeout, opts...); err != nil {
			return err
		}
	}

	snap := &state.Snapshot{
		Cluster:   "",
		Namespace: namespace,
		SavedAt:   time.Now().UTC(),
	}

	for i := range appList.Items {
		app := &appList.Items[i]

		raw := json.RawMessage("null")
		if app.Spec.SyncPolicy != nil {
			b, err := json.Marshal(app.Spec.SyncPolicy)
			if err != nil {
				return fmt.Errorf("marshal syncPolicy %s: %w", app.Name, err)
			}
			raw = b
		}

		snap.Applications = append(snap.Applications, state.ApplicationSyncState{
			Name:       app.Name,
			Namespace:  app.Namespace,
			SyncPolicy: raw,
		})
	}

	if err := state.Save(stateFile, snap); err != nil {
		return fmt.Errorf("save snapshot: %w", err)
	}

	if _, err := fmt.Fprintf(cmd.OutOrStdout(),
		"Saved snapshot for %d applications to %s\n(No changes made to cluster)\n",
		len(snap.Applications), stateFile); err != nil {
		return fmt.Errorf("failed writing to stdout: %w", err)
	}

	return nil
}

// runMaintenanceDryRun lists each Application and whether AutoSync is enabled, and whether an active sync is in progress.
func runMaintenanceDryRun(cmd *cobra.Command, namespace, selector string) error {
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

	if _, err := fmt.Fprintf(cmd.OutOrStdout(),
		"Dry run: listing Argo CD Applications in namespace %q\n\n", namespace); err != nil {
		return fmt.Errorf("failed writing to stdout: %w", err)
	}

	// Using tabwriter for output to make this table readable
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)

	// Print Table Header
	if _, err := fmt.Fprintln(w, "NAME\tAUTOSYNC\tACTIVE_SYNC\t"); err != nil {
		return fmt.Errorf("failed writing to stdout: %w", err)
	}

	// Print each row
	for i := range appList.Items {
		app := &appList.Items[i]
		autoSync := app.Spec.SyncPolicy != nil && app.Spec.SyncPolicy.Automated != nil
		activeSync := isActiveSync(app)
		if _, err := fmt.Fprintf(w, "%s\t%t\t%t\t\n", app.Name, autoSync, activeSync); err != nil {
			return fmt.Errorf("failed writing to stdout: %w", err)
		}
	}

	if err := w.Flush(); err != nil {
		return fmt.Errorf("flush tabwriter: %w", err)
	}

	return nil
}

// waitForActiveSyncsClear polls until no Applications have active sync operations, or a timeout is reached.
func waitForActiveSyncsClear(parentCtx context.Context, k8sClient client.Client, cmd *cobra.Command, timeout time.Duration, opts ...client.ListOption) error {
	// Use the provided timeout so callers can tune behavior.
	ctx, cancel := context.WithTimeout(parentCtx, timeout)
	defer cancel()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for active sync operations to complete (timeout: %s)", timeout)
		case <-ticker.C:
			var appList appsv1alpha1.ApplicationList
			if err := k8sClient.List(ctx, &appList, opts...); err != nil {
				return fmt.Errorf("list applications while waiting for active syncs: %w", err)
			}
			active := collectActiveSyncs(&appList)
			if len(active) == 0 {
				if _, err := fmt.Fprintln(cmd.OutOrStdout(),
					"No active sync operations remain; proceeding."); err != nil {
					return fmt.Errorf("failed writing to stdout: %w", err)
				}
				return nil
			}
		}
	}
}

// isActiveSync returns true if the Application has an active (running) operation.
func isActiveSync(app *appsv1alpha1.Application) bool {
	if app.Status.OperationState == nil {
		return false
	}
	return app.Status.OperationState.Phase == "Running"
}

// collectActiveSyncs returns the names of Applications that currently have an active sync operation.
func collectActiveSyncs(list *appsv1alpha1.ApplicationList) []string {
	var names []string
	for i := range list.Items {
		app := &list.Items[i]
		if isActiveSync(app) {
			names = append(names, app.Name)
		}
	}
	return names
}
