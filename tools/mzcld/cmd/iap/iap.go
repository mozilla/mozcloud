package iap

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"google.golang.org/api/impersonate"
)

func NewIAPCmd() *cobra.Command {
	var (
		host           string
		serviceAccount string
		format         string
		verbose        bool
		debug          bool
	)

	cmd := &cobra.Command{
		Use:   "iap",
		Short: "Generate IAP token for ArgoCD",
		Long: `Generate an Identity-Aware Proxy (IAP) token for accessing ArgoCD.

This command automatically:
  - Discovers the OAuth Client ID from the IAP-protected host
  - Impersonates the appropriate service account
  - Generates an ID token for IAP authentication

Prerequisites:
  - You must be authenticated with gcloud: gcloud auth login
  - You must have roles/iam.serviceAccountTokenCreator on the service account

Examples:
  # Generate token for sandbox ArgoCD
  export IAP_TOKEN=$(mzcld iap --host sandbox.argocd.global.mozgcp.net)

  # Use with ArgoCD login
  argocd login sandbox.argocd.global.mozgcp.net \
    --header "Proxy-Authorization: Bearer $IAP_TOKEN" \
    --grpc-web --sso

  # Use with ArgoCD commands
  argocd app list --header "Proxy-Authorization: Bearer $IAP_TOKEN" --grpc-web

  # Override the default service account
  export IAP_TOKEN=$(mzcld iap \
    --host sandbox.argocd.global.mozgcp.net \
    --service-account my-custom-sa@project.iam.gserviceaccount.com)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if host == "" {
				return fmt.Errorf("--host is required")
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// Auto-discover client ID from host
			if verbose {
				if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "Discovering IAP client ID from %s...\n", host); err != nil {
					return fmt.Errorf("failed writing to stderr: %w", err)
				}
			}

			clientID, err := DiscoverClientID(ctx, host)
			if err != nil {
				return fmt.Errorf("discover client ID from %s: %w", host, err)
			}

			if verbose {
				if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "Discovered client ID: %s\n", clientID); err != nil {
					return fmt.Errorf("failed writing to stderr: %w", err)
				}
			}

			// Determine service account to impersonate
			if serviceAccount == "" {
				serviceAccount = GetDefaultServiceAccount(host)
			}

			if verbose {
				if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "Impersonating service account: %s\n", serviceAccount); err != nil {
					return fmt.Errorf("failed writing to stderr: %w", err)
				}
			}

			// Generate token via service account impersonation
			token, err := GenerateToken(ctx, clientID, serviceAccount)
			if err != nil {
				return fmt.Errorf("generate IAP token: %w", err)
			}

			if debug {
				if err := debugToken(cmd, token, clientID); err != nil {
					if _, errw := fmt.Fprintf(cmd.ErrOrStderr(), "Warning: failed to decode token for debugging: %v\n", err); errw != nil {
						return fmt.Errorf("failed writing to stderr: %w", errw)
					}
				}
			}

			switch format {
			case "export":
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "export IAP_TOKEN=%s\n", token); err != nil {
					return fmt.Errorf("failed writing to stdout: %w", err)
				}
			case "json":
				output := struct {
					Token    string `json:"token"`
					Host     string `json:"host"`
					ClientID string `json:"client_id"`
				}{
					Token:    token,
					Host:     host,
					ClientID: clientID,
				}
				b, err := json.MarshalIndent(output, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal json: %w", err)
				}
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\n", b); err != nil {
					return fmt.Errorf("failed writing to stdout: %w", err)
				}
			default:
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), token); err != nil {
					return fmt.Errorf("failed writing to stdout: %w", err)
				}
			}

			if verbose {
				if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "IAP token generated successfully\n"); err != nil {
					return fmt.Errorf("failed writing to stderr: %w", err)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&host, "host", "", "ArgoCD hostname (e.g., sandbox.argocd.global.mozgcp.net)")
	cmd.Flags().StringVar(&serviceAccount, "service-account", "", "Service account to impersonate (defaults based on hostname)")
	cmd.Flags().StringVarP(&format, "format", "f", "raw", "Output format: raw, export, or json")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	cmd.Flags().BoolVarP(&debug, "debug", "d", false, "Enable debug output (shows token claims)")

	if err := cmd.MarkFlagRequired("host"); err != nil {
		panic(fmt.Sprintf("failed to mark host flag as required: %v", err))
	}

	return cmd
}

// GetDefaultServiceAccount returns the default service account for a given host
// Should we create a mzcld-cli service account for platform IAP access
func GetDefaultServiceAccount(host string) string {
	// Map known hostnames to their service accounts
	switch {
	case strings.Contains(host, "sandbox"):
		return "argocd-sandbox@moz-fx-platform-mgmt-global.iam.gserviceaccount.com"
	case strings.Contains(host, "webservices") || strings.Contains(host, "web"):
		return "argocd-webservices@moz-fx-platform-mgmt-global.iam.gserviceaccount.com"
	case strings.Contains(host, "dataservices") || strings.Contains(host, "data"):
		return "argocd-dataservices@moz-fx-platform-mgmt-global.iam.gserviceaccount.com"
	default:
		// Default to sandbox for unknown environments
		return "argocd-sandbox@moz-fx-platform-mgmt-global.iam.gserviceaccount.com"
	}
}

// GenerateToken generates an IAP token by impersonating a service account
func GenerateToken(ctx context.Context, clientID, serviceAccountEmail string) (string, error) {
	// Create an impersonated ID token source
	// This uses the user's Application Default Credentials to impersonate the service account
	ts, err := impersonate.IDTokenSource(ctx, impersonate.IDTokenConfig{
		Audience:        clientID,
		TargetPrincipal: serviceAccountEmail,
		IncludeEmail:    true,
	})
	if err != nil {
		return "", fmt.Errorf("create impersonated token source: %w\n\n"+
			"Make sure:\n"+
			"1. You're authenticated: gcloud auth login or gcloud auth application-default login\n"+
			"2. You have roles/iam.serviceAccountTokenCreator on %s", err, serviceAccountEmail)
	}

	// Get the ID token
	token, err := ts.Token()
	if err != nil {
		return "", fmt.Errorf("get impersonated token: %w", err)
	}

	// The AccessToken field actually contains the ID token
	if token.AccessToken == "" {
		return "", fmt.Errorf("received empty token")
	}

	return token.AccessToken, nil
}

// debugToken decodes and displays the JWT token claims for debugging
func debugToken(cmd *cobra.Command, token string, expectedAudience string) error {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return fmt.Errorf("invalid JWT format")
	}

	// Decode the payload (second part)
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return fmt.Errorf("unmarshal claims: %w", err)
	}

	if _, err := fmt.Fprintln(cmd.ErrOrStderr(), "\n=== Token Debug Info ==="); err != nil {
		return fmt.Errorf("failed writing to stderr: %w", err)
	}

	// Check key claims
	if aud, ok := claims["aud"].(string); ok {
		if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "Audience (aud): %s\n", aud); err != nil {
			return fmt.Errorf("failed writing to stderr: %w", err)
		}
		if aud != expectedAudience {
			if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "WARNING: Audience does not match expected client ID!\n"); err != nil {
				return fmt.Errorf("failed writing to stderr: %w", err)
			}
			if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "Expected: %s\n", expectedAudience); err != nil {
				return fmt.Errorf("failed writing to stderr: %w", err)
			}
		}
	}

	if iss, ok := claims["iss"].(string); ok {
		if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "Issuer (iss): %s\n", iss); err != nil {
			return fmt.Errorf("failed writing to stderr: %w", err)
		}
	}

	if email, ok := claims["email"].(string); ok {
		if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "Email: %s\n", email); err != nil {
			return fmt.Errorf("failed writing to stderr: %w", err)
		}
	}

	if exp, ok := claims["exp"].(float64); ok {
		expTime := time.Unix(int64(exp), 0)
		if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "Expires: %s\n", expTime.Format(time.RFC3339)); err != nil {
			return fmt.Errorf("failed writing to stderr: %w", err)
		}
		if time.Now().After(expTime) {
			if _, err := fmt.Fprintln(cmd.ErrOrStderr(), "WARNING: Token is expired!"); err != nil {
				return fmt.Errorf("failed writing to stderr: %w", err)
			}
		}
	}

	if _, err := fmt.Fprintln(cmd.ErrOrStderr(), "========================\n"); err != nil {
		return fmt.Errorf("failed writing to stderr: %w", err)
	}

	return nil
}
