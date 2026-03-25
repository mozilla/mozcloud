// Package iap implements the `mzcld iap` command for generating IAP tokens.
package iap

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"time"

	"github.com/mozilla/mozcloud/tools/mzcld/internal/iap"
	"github.com/mozilla/mozcloud/tools/mzcld/internal/ui"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

var (
	hostFlag           string
	serviceAccountFlag string
	proxyFlag          bool
	portFlag           int
)

// Cmd is the root `iap` command.
var Cmd = &cobra.Command{
	Use:   "iap",
	Short: "Generate an IAP token for a protected endpoint",
	Long: `Discover the OAuth Client ID for an IAP-protected hostname and generate
an IAP token via service account impersonation.

The token is printed to stdout, making it suitable for shell capture:

  TOKEN=$(mzcld iap --host argocd.example.mozilla.com)

Use --proxy to start a local HTTP proxy that forwards requests to the
IAP-protected host with the token automatically injected and refreshed:

  mzcld iap --host grafana.example.mozilla.com --proxy`,
	RunE: runIAP,
}

func init() {
	Cmd.Flags().StringVar(&hostFlag, "host", "", "IAP-protected hostname (required)")
	Cmd.Flags().StringVar(&serviceAccountFlag, "service-account", "", "Service account to impersonate (optional)")
	Cmd.Flags().BoolVar(&proxyFlag, "proxy", false, "Start a local proxy that injects the IAP token for each request")
	Cmd.Flags().IntVar(&portFlag, "port", 8080, "Local port for the proxy")
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

	ui.Debug("Using service account: " + serviceAccount)

	if proxyFlag {
		ts, err := iap.NewTokenSource(ctx, clientID, serviceAccount)
		if err != nil {
			return err
		}
		return runProxy(ctx, hostFlag, portFlag, ts)
	}

	token, err := iap.GenerateToken(ctx, clientID, serviceAccount)
	if err != nil {
		return err
	}

	// Print token to stdout — suitable for shell capture
	fmt.Println(token)
	return nil
}

// iapTransport injects the Proxy-Authorization header at the transport layer,
// after httputil.ReverseProxy has already stripped hop-by-hop headers.
type iapTransport struct {
	base http.RoundTripper
	ts   oauth2.TokenSource
}

func (t *iapTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	tok, err := t.ts.Token()
	if err != nil {
		return nil, fmt.Errorf("get IAP token: %w", err)
	}
	req = req.Clone(req.Context())
	req.Header.Set("Proxy-Authorization", "Bearer "+tok.AccessToken)

	if ui.IsDebug() {
		dump, err := httputil.DumpRequestOut(req, true)
		if err != nil {
			ui.Debug("failed to dump request: " + err.Error())
		} else {
			ui.Debug("→ request:\n" + string(dump))
		}
	}

	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	if ui.IsDebug() {
		dump, err := httputil.DumpResponse(resp, true)
		if err != nil {
			ui.Debug("failed to dump response: " + err.Error())
		} else {
			ui.Debug("← response:\n" + string(dump))
		}
	}

	return resp, nil
}

func runProxy(ctx context.Context, host string, port int, ts oauth2.TokenSource) error {
	targetURL := &url.URL{Scheme: "https", Host: host}
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = targetURL.Scheme
		req.URL.Host = targetURL.Host
		req.Host = host
	}

	proxy.Transport = &iapTransport{
		base: http.DefaultTransport,
		ts:   ts,
	}

	addr := "localhost:" + strconv.Itoa(port)
	ui.Success(fmt.Sprintf("IAP proxy listening on http://%s", addr))
	ui.Info(fmt.Sprintf("Forwarding → https://%s", host))
	ui.Dim("Press Ctrl+C to stop")

	srv := &http.Server{
		Addr:    ":" + strconv.Itoa(port),
		Handler: proxy,
	}

	go func() {
		<-ctx.Done()
		srv.Shutdown(context.Background()) //nolint:errcheck
	}()

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
