package argo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	argocdclient "github.com/argoproj/argo-cd/v3/pkg/apiclient"
	"github.com/argoproj/argo-cd/v3/pkg/apiclient/application"
	"github.com/mozilla/mozcloud/tools/mzcld/cmd/iap"
	"github.com/spf13/cobra"
)

func NewArgoAppCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "app",
		Short: "Manage ArgoCD applications",
		Long:  "Manage ArgoCD applications with automatic IAP authentication",
	}

	cmd.AddCommand(NewArgoAppListCmd())
	cmd.AddCommand(NewArgoAppSyncCmd())
	cmd.AddCommand(NewArgoAppRollbackCmd())

	return cmd
}

func NewArgoAppListCmd() *cobra.Command {
	var (
		host      string
		namespace string
		selector  string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List applications",
		Long:  "List ArgoCD applications with automatic IAP authentication",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Use current context if host not specified
			if host == "" {
				currentContext, err := getCurrentContext()
				if err != nil {
					return err
				}
				host = currentContext
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// Generate IAP token
			iapToken, err := generateIAPToken(ctx, host)
			if err != nil {
				return fmt.Errorf("generate IAP token: %w", err)
			}

			// Create ArgoCD client
			conn, appClient := mustNewApplicationClient(ctx, host, iapToken)
			defer conn.Close()

			// List applications
			query := &application.ApplicationQuery{}

			// Only filter by namespace if explicitly provided
			if cmd.Flags().Changed("namespace") {
				query.AppNamespace = &namespace
			}

			if selector != "" {
				query.Selector = &selector
			}

			appList, err := appClient.List(ctx, query)
			if err != nil {
				return fmt.Errorf("list applications: %w", err)
			}

			// Display results
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tNAMESPACE\tCLUSTER\tSYNC\tHEALTH\tREVISION")

			for _, app := range appList.Items {
				syncStatus := string(app.Status.Sync.Status)
				healthStatus := string(app.Status.Health.Status)
				revision := app.Status.Sync.Revision
				if len(revision) > 8 {
					revision = revision[:8]
				}

				cluster := app.Spec.Destination.Name
				if cluster == "" {
					cluster = app.Spec.Destination.Server
				}

				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
					app.Name,
					app.Namespace,
					cluster,
					syncStatus,
					healthStatus,
					revision,
				)
			}
			w.Flush()

			return nil
		},
	}

	cmd.Flags().StringVar(&host, "host", "", "ArgoCD hostname (uses current context if not specified)")
	cmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Filter by namespace (queries all namespaces if not specified)")
	cmd.Flags().StringVar(&selector, "selector", "", "Label selector to filter applications")

	return cmd
}

func NewArgoAppSyncCmd() *cobra.Command {
	var (
		host     string
		appName  string
		prune    bool
		dryRun   bool
		revision string
	)

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync an application",
		Long:  "Sync an ArgoCD application to its target state",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Use current context if host not specified
			if host == "" {
				currentContext, err := getCurrentContext()
				if err != nil {
					return err
				}
				host = currentContext
			}

			if appName == "" {
				return fmt.Errorf("--name is required")
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			// Generate IAP token
			iapToken, err := generateIAPToken(ctx, host)
			if err != nil {
				return fmt.Errorf("generate IAP token: %w", err)
			}

			// Create ArgoCD client
			conn, appClient := mustNewApplicationClient(ctx, host, iapToken)
			defer conn.Close()

			// Build sync request
			syncReq := &application.ApplicationSyncRequest{
				Name:   &appName,
				Prune:  &prune,
				DryRun: &dryRun,
			}

			if revision != "" {
				syncReq.Revision = &revision
			}

			// Trigger sync
			fmt.Fprintf(cmd.OutOrStdout(), "Syncing application %s...\n", appName)
			_, err = appClient.Sync(ctx, syncReq)
			if err != nil {
				return fmt.Errorf("sync application: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Sync initiated successfully\n")

			return nil
		},
	}

	cmd.Flags().StringVar(&host, "host", "", "ArgoCD hostname (uses current context if not specified)")
	cmd.Flags().StringVar(&appName, "name", "", "Application name")
	cmd.Flags().BoolVar(&prune, "prune", false, "Allow deleting unexpected resources")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview sync without executing")
	cmd.Flags().StringVar(&revision, "revision", "", "Sync to a specific revision")

	cmd.MarkFlagRequired("name")

	return cmd
}

func NewArgoAppRollbackCmd() *cobra.Command {
	var (
		host    string
		appName string
		id      int64
	)

	cmd := &cobra.Command{
		Use:   "rollback",
		Short: "Rollback an application",
		Long:  "Rollback an ArgoCD application to a previous deployment",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Use current context if host not specified
			if host == "" {
				currentContext, err := getCurrentContext()
				if err != nil {
					return err
				}
				host = currentContext
			}

			if appName == "" {
				return fmt.Errorf("--name is required")
			}
			if id == 0 {
				return fmt.Errorf("--id is required")
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// Generate IAP token
			iapToken, err := generateIAPToken(ctx, host)
			if err != nil {
				return fmt.Errorf("generate IAP token: %w", err)
			}

			// Create ArgoCD client
			conn, appClient := mustNewApplicationClient(ctx, host, iapToken)
			defer conn.Close()

			// Rollback
			fmt.Fprintf(cmd.OutOrStdout(), "Rolling back application %s to deployment %d...\n", appName, id)
			_, err = appClient.Rollback(ctx, &application.ApplicationRollbackRequest{
				Name: &appName,
				Id:   &id,
			})
			if err != nil {
				return fmt.Errorf("rollback application: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Rollback initiated successfully\n")

			return nil
		},
	}

	cmd.Flags().StringVar(&host, "host", "", "ArgoCD hostname (uses current context if not specified)")
	cmd.Flags().StringVar(&appName, "name", "", "Application name")
	cmd.Flags().Int64Var(&id, "id", 0, "Deployment ID to rollback to")

	cmd.MarkFlagRequired("name")
	cmd.MarkFlagRequired("id")

	return cmd
}

// getCurrentContext loads the current context from ArgoCD config
func getCurrentContext() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home directory: %w", err)
	}
	configPath := filepath.Join(homeDir, ".config", "argocd", "config")

	configData, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("read config file: %w\nHint: Run 'mzcld argo login --host <hostname>' first", err)
	}

	var config struct {
		CurrentContext string `json:"current-context"`
	}
	if err := json.Unmarshal(configData, &config); err != nil {
		return "", fmt.Errorf("parse config: %w", err)
	}

	if config.CurrentContext == "" {
		return "", fmt.Errorf("no current context set\nHint: Run 'mzcld argo login --host <hostname>' first")
	}

	return config.CurrentContext, nil
}

// generateIAPToken generates an IAP token for the given host
func generateIAPToken(ctx context.Context, host string) (string, error) {
	// Discover client ID
	clientID, err := iap.DiscoverClientID(ctx, host)
	if err != nil {
		return "", fmt.Errorf("discover client ID: %w", err)
	}

	// Get service account
	serviceAccount := iap.GetDefaultServiceAccount(host)

	// Generate token
	token, err := iap.GenerateToken(ctx, clientID, serviceAccount)
	if err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}

	return token, nil
}

// mustNewApplicationClient creates an application client or exits
func mustNewApplicationClient(ctx context.Context, host, iapToken string) (io.Closer, application.ApplicationServiceClient) {
	// Get home directory for config path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
		os.Exit(1)
	}
	configPath := filepath.Join(homeDir, ".config", "argocd", "config")

	// Normalize host - remove any protocol prefix if present
	normalizedHost := host
	normalizedHost = strings.TrimPrefix(normalizedHost, "https://")
	normalizedHost = strings.TrimPrefix(normalizedHost, "http://")

	clientOpts := argocdclient.ClientOptions{
		ServerAddr:      normalizedHost,
		ConfigPath:      configPath,
		Context:         normalizedHost, // Context name matches normalized hostname
		GRPCWeb:         true,
		GRPCWebRootPath: "",
		Insecure:        false,
		Headers: []string{
			fmt.Sprintf("Proxy-Authorization: Bearer %s", iapToken),
		},
	}

	client, err := argocdclient.NewClient(&clientOpts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating ArgoCD client: %v\n", err)
		os.Exit(1)
	}

	conn, appClient, err := client.NewApplicationClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating application client: %v\n", err)
		os.Exit(1)
	}

	return conn, appClient
}
