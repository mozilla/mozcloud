// Package iap implements the `mzcld iap` command for generating IAP tokens.
package iap

import (
	"fmt"
	"net/http"
	"time"

	"github.com/mozilla/mozcloud/tools/mzcld/internal/iap"
	"github.com/mozilla/mozcloud/tools/mzcld/internal/ui"
	"github.com/spf13/cobra"
)

var (
	hostFlag           string
	serviceAccountFlag string
)

// Cmd is the root `iap` command.
var Cmd = &cobra.Command{
	Use:   "iap",
	Short: "Generate an IAP token for a protected endpoint",
	Long: `Discover the OAuth Client ID for an IAP-protected hostname and generate
an IAP token via service account impersonation.

The token is printed to stdout, making it suitable for shell capture:

  TOKEN=$(mzcld iap --host argocd.example.mozilla.com)`,
	RunE: runIAP,
}

func init() {
	Cmd.Flags().StringVar(&hostFlag, "host", "", "IAP-protected hostname (required)")
	Cmd.Flags().StringVar(&serviceAccountFlag, "service-account", "", "Service account to impersonate (optional)")
	_ = Cmd.MarkFlagRequired("host")
	Cmd.Flags().SortFlags = false
}

func runIAP(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	// Build an HTTP client that does not follow redirects and has a short timeout
	httpClient := &http.Client{
		Timeout: 2 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	ui.Debug(fmt.Sprintf("Discovering OAuth Client ID for %s...", hostFlag))

	clientID, err := iap.DiscoverOAuthClientID(ctx, hostFlag, httpClient)
	if err != nil {
		return fmt.Errorf("failed to discover IAP client ID: %w", err)
	}

	ui.Debug("Discovered client ID: " + clientID)

	// Resolve service account
	serviceAccount := serviceAccountFlag
	if serviceAccount == "" {
		serviceAccount = iap.GetDefaultServiceAccount(hostFlag)
	}
	if serviceAccount == "" {
		return fmt.Errorf("--service-account is required for host %s (no default known)", hostFlag)
	}

	ui.Debug("Generating token for service account: " + serviceAccount)

	token, err := iap.GenerateToken(ctx, clientID, serviceAccount)
	if err != nil {
		return err
	}

	// Print token to stdout — suitable for shell capture
	fmt.Println(token)
	return nil
}
