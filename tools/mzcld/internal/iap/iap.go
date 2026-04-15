// Package iap provides helpers for authenticating to GCP Identity-Aware Proxy
// protected endpoints and generating IAP tokens via service account
// impersonation.
package iap

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/mozilla/mozcloud/tools/mzcld/internal/gcp"
	"golang.org/x/oauth2"
	"google.golang.org/api/impersonate"
)

// DiscoverOAuthClientID attempts to discover the IAP OAuth Client ID for a
// given hostname by making an unauthenticated request and inspecting the
// redirect to the Google OAuth authorization endpoint.
func DiscoverOAuthClientID(ctx context.Context, host string, client *http.Client) (string, error) {
	raw := host
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}

	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("failed to parse host: %w", err)
	}

	targetURL := fmt.Sprintf("https://%s/", u.Hostname())

	req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("make request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusMovedPermanently {
		location := resp.Header.Get("Location")
		if location == "" {
			return "", fmt.Errorf("redirect response but no Location header")
		}
		if strings.Contains(location, "client_id=") {
			clientID := extractClientID(location)
			if clientID != "" {
				return clientID, nil
			}
		}
		return "", fmt.Errorf("redirect found but no client_id in URL: %s", location)
	}

	return "", fmt.Errorf(
		"could not discover IAP client ID from %s (status: %d)\nMake sure the hostname is correct and protected by IAP",
		u.Hostname(), resp.StatusCode,
	)
}

// NewTokenSource creates an IAP token source by impersonating serviceAccountEmail.
// The returned source handles token caching and refresh automatically.
func NewTokenSource(ctx context.Context, clientID, serviceAccountEmail string) (oauth2.TokenSource, error) {
	ts, err := impersonate.IDTokenSource(ctx, impersonate.IDTokenConfig{
		Audience:        clientID,
		TargetPrincipal: serviceAccountEmail,
		IncludeEmail:    true,
	}, gcp.ClientOption())
	if err != nil {
		return nil, fmt.Errorf(
			"create impersonated token source: %w\n\n"+
				"Make sure:\n"+
				"1. You're authenticated: gcloud auth login or gcloud auth application-default login\n"+
				"2. You have roles/iam.serviceAccountTokenCreator on %s",
			err, serviceAccountEmail,
		)
	}
	return ts, nil
}

// GenerateToken generates an IAP token by impersonating serviceAccountEmail.
// It uses the caller's Application Default Credentials.
func GenerateToken(ctx context.Context, clientID, serviceAccountEmail string) (string, error) {
	ts, err := NewTokenSource(ctx, clientID, serviceAccountEmail)
	if err != nil {
		return "", err
	}

	token, err := ts.Token()
	if err != nil {
		return "", fmt.Errorf("get impersonated token: %w", err)
	}

	if token.AccessToken == "" {
		return "", fmt.Errorf("received empty token")
	}
	return token.AccessToken, nil
}

// GetDefaultServiceAccount returns the default service account for a given
// IAP-protected host. Returns an empty string if no default is known.
func GetDefaultServiceAccount(host string) string {
	hostname := strings.ToLower(host)
	allowedHosts := map[string]string{"argocd": "argocd", "yardstick": "grafana"}
	for host, sa := range allowedHosts {
		if strings.Contains(hostname, host) {
			return fmt.Sprintf("%s-iap-access@moz-fx-platform-mgmt-global.iam.gserviceaccount.com", sa)
		}
	}
	return ""
}

// extractClientID extracts the client_id parameter from a redirect URL.
func extractClientID(redirectURL string) string {
	parts := strings.Split(redirectURL, "client_id=")
	if len(parts) < 2 {
		return ""
	}
	clientIDPart := parts[1]
	if idx := strings.Index(clientIDPart, "&"); idx != -1 {
		clientIDPart = clientIDPart[:idx]
	}
	if strings.HasSuffix(clientIDPart, ".apps.googleusercontent.com") {
		return clientIDPart
	}
	return ""
}
