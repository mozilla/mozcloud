package iap

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DiscoverClientID attempts to discover the IAP OAuth Client ID for a given hostname
// by making an unauthenticated request and inspecting the IAP redirect
func DiscoverClientID(ctx context.Context, hostname string) (string, error) {
	// Ensure hostname doesn't have protocol
	hostname = strings.TrimPrefix(hostname, "https://")
	hostname = strings.TrimPrefix(hostname, "http://")

	// Try HTTPS first (IAP always uses HTTPS)
	url := fmt.Sprintf("https://%s/", hostname)

	// Create a client that doesn't follow redirects
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("make request: %w", err)
	}
	defer resp.Body.Close()

	// IAP returns a 302 redirect to the OAuth authorization endpoint
	if resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusMovedPermanently {
		location := resp.Header.Get("Location")
		if location == "" {
			return "", fmt.Errorf("redirect response but no Location header")
		}

		// Parse the redirect URL to extract client_id parameter
		// Example: https://accounts.google.com/o/oauth2/v2/auth?client_id=XXX.apps.googleusercontent.com&...
		if strings.Contains(location, "client_id=") {
			clientID := extractClientID(location)
			if clientID != "" {
				return clientID, nil
			}
		}

		return "", fmt.Errorf("redirect found but no client_id in URL: %s", location)
	}

	// Try reading response body for IAP error page which might contain client ID
	body, err := io.ReadAll(resp.Body)
	if err == nil {
		if clientID := extractClientIDFromHTML(string(body)); clientID != "" {
			return clientID, nil
		}
	}

	return "", fmt.Errorf("could not discover IAP client ID from %s (status: %d)\nMake sure the hostname is correct and protected by IAP", hostname, resp.StatusCode)
}

// extractClientID extracts the client_id parameter from a URL query string
func extractClientID(redirectURL string) string {
	// Look for client_id= in the URL
	parts := strings.Split(redirectURL, "client_id=")
	if len(parts) < 2 {
		return ""
	}

	// Get everything after client_id= until the next & or end of string
	clientIDPart := parts[1]
	if idx := strings.Index(clientIDPart, "&"); idx != -1 {
		clientIDPart = clientIDPart[:idx]
	}

	// Validate it looks like a client ID
	if strings.HasSuffix(clientIDPart, ".apps.googleusercontent.com") {
		return clientIDPart
	}

	return ""
}

// extractClientIDFromHTML tries to find client ID in HTML response
// IAP error pages sometimes embed the client ID in data attributes or JavaScript
func extractClientIDFromHTML(html string) string {
	// Simple string search (not using regexp to keep it lightweight)
	if strings.Contains(html, ".apps.googleusercontent.com") {
		// Extract the full client ID
		start := strings.Index(html, ".apps.googleusercontent.com")
		if start == -1 {
			return ""
		}

		// Look backwards to find the start of the client ID
		for i := start - 1; i >= 0; i-- {
			if html[i] == '"' || html[i] == '\'' || html[i] == '=' || html[i] == ' ' {
				clientID := html[i+1 : start+len(".apps.googleusercontent.com")]
				if strings.Contains(clientID, "-") && len(clientID) > 20 {
					return clientID
				}
				break
			}
		}
	}

	return ""
}

// IAPConfig stores configuration for an IAP-protected service
type IAPConfig struct {
	Hostname       string `json:"hostname"`
	ClientID       string `json:"client_id"`
	ServiceAccount string `json:"service_account,omitempty"`
	DiscoveredAt   string `json:"discovered_at,omitempty"`
}

// saveIAPConfig saves the discovered IAP configuration to cache
func saveIAPConfig(hostname, clientID, serviceAccount string) error {
	// This could be extended to save to a config file for reuse
	// For now, we'll just return nil
	return nil
}

// loadIAPConfig loads cached IAP configuration
func loadIAPConfig(hostname string) (*IAPConfig, error) {
	// This could be extended to load from a config file
	// For now, we'll return not found
	return nil, fmt.Errorf("no cached config for %s", hostname)
}
