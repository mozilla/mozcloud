package argo

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	argocdclient "github.com/argoproj/argo-cd/v3/pkg/apiclient"
	settingspkg "github.com/argoproj/argo-cd/v3/pkg/apiclient/settings"
	"github.com/argoproj/argo-cd/v3/util/rand"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

func NewArgoLoginCmd() *cobra.Command {
	var (
		host    string
		sso     bool
		verbose bool
	)

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Login to ArgoCD",
		Long: `Login to ArgoCD with automatic IAP and SSO authentication.

This command performs a two-step authentication process:
  1. Generates an IAP token using service account impersonation to pass through Google's IAP
  2. Performs SSO/OAuth2 login with ArgoCD (opens browser for authentication)
  3. Saves the ArgoCD auth token to ~/.config/argocd/config

The IAP token allows requests to pass through Google's Identity-Aware Proxy, while
the ArgoCD auth token authenticates you with the ArgoCD server itself.

Note: ArgoCD auth tokens typically expire after 24 hours. Re-run this command when needed.

Examples:
  # Login to sandbox ArgoCD
  mzcld argo login --host sandbox.argocd.global.mozgcp.net

  # Login with verbose output
  mzcld argo login --host sandbox.argocd.global.mozgcp.net --verbose`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if host == "" {
				return fmt.Errorf("--host is required")
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// Generate IAP token
			if verbose {
				if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "Generating IAP token...\n"); err != nil {
					return fmt.Errorf("failed writing to stderr: %w", err)
				}
			}

			iapToken, err := generateIAPToken(ctx, host)
			if err != nil {
				return fmt.Errorf("generate IAP token: %w", err)
			}

			if verbose {
				if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "Authenticating with ArgoCD via SSO...\n"); err != nil {
					return fmt.Errorf("failed writing to stderr: %w", err)
				}
			}

			// Get ArgoCD auth token using SSO
			argoToken, err := performSSOLogin(ctx, host, iapToken, verbose, cmd)
			if err != nil {
				return fmt.Errorf("SSO login: %w", err)
			}

			if verbose {
				if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "Saving ArgoCD auth token to config...\n"); err != nil {
					return fmt.Errorf("failed writing to stderr: %w", err)
				}
			}

			// Save ArgoCD auth token to config file
			if err := saveArgoConfig(host, argoToken); err != nil {
				return fmt.Errorf("save config: %w", err)
			}

			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Successfully logged in to %s\n", host); err != nil {
				return fmt.Errorf("failed writing to stdout: %w", err)
			}

			if verbose {
				if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "Note: Auth token will expire after 24 hours, re-run login when needed\n"); err != nil {
					return fmt.Errorf("failed writing to stderr: %w", err)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&host, "host", "", "ArgoCD hostname (e.g., sandbox.argocd.global.mozgcp.net)")
	cmd.Flags().BoolVar(&sso, "sso", true, "Use SSO authentication (default true)")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")

	if err := cmd.MarkFlagRequired("host"); err != nil {
		panic(fmt.Sprintf("failed to mark host flag as required: %v", err))
	}

	return cmd
}

// ArgoConfig represents the ArgoCD configuration file structure
type ArgoConfig struct {
	CurrentContext string                 `json:"current-context"`
	Contexts       []ArgoConfigContext    `json:"contexts"`
	Servers        []ArgoConfigServer     `json:"servers"`
	Users          []ArgoConfigUser       `json:"users,omitempty"`
}

type ArgoConfigContext struct {
	Name   string `json:"name"`
	User   string `json:"user"`
	Server string `json:"server"`
}

type ArgoConfigServer struct {
	Server                       string `json:"server"`
	CACertificateAuthorityData   string `json:"certificate-authority-data,omitempty"`
	ClientCertificateData        string `json:"client-certificate-data,omitempty"`
	ClientCertificateKeyData     string `json:"client-key-data,omitempty"`
	Insecure                     bool   `json:"insecure,omitempty"`
	GRPCWeb                      bool   `json:"grpc-web,omitempty"`
	GRPCWebRootPath              string `json:"grpc-web-root-path,omitempty"`
}

type ArgoConfigUser struct {
	Name         string `json:"name"`
	AuthToken    string `json:"auth-token,omitempty"`
	RefreshToken string `json:"refresh-token,omitempty"`
}

// saveArgoConfig saves the session token to the ArgoCD config file
func saveArgoConfig(host, token string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config", "argocd")
	configPath := filepath.Join(configDir, "config")

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	// Read existing config or create new one
	var config ArgoConfig
	configData, err := os.ReadFile(configPath)
	if err == nil {
		// Parse existing config
		if err := json.Unmarshal(configData, &config); err != nil {
			// If parsing fails, treat as new config
			config = ArgoConfig{}
		}
	}

	// Normalize host - remove any protocol prefix
	normalizedHost := host
	normalizedHost = strings.TrimPrefix(normalizedHost, "https://")
	normalizedHost = strings.TrimPrefix(normalizedHost, "http://")

	contextName := normalizedHost

	// Update or add server (store without https:// prefix)
	serverFound := false
	for i, server := range config.Servers {
		if server.Server == normalizedHost {
			config.Servers[i].GRPCWeb = true
			serverFound = true
			break
		}
	}
	if !serverFound {
		config.Servers = append(config.Servers, ArgoConfigServer{
			Server:  normalizedHost,
			GRPCWeb: true,
		})
	}

	// Update or add user
	userFound := false
	for i, user := range config.Users {
		if user.Name == contextName {
			config.Users[i].AuthToken = token
			userFound = true
			break
		}
	}
	if !userFound {
		config.Users = append(config.Users, ArgoConfigUser{
			Name:      contextName,
			AuthToken: token,
		})
	}

	// Update or add context
	contextFound := false
	for i, ctx := range config.Contexts {
		if ctx.Name == contextName {
			config.Contexts[i].Server = normalizedHost
			config.Contexts[i].User = contextName
			contextFound = true
			break
		}
	}
	if !contextFound {
		config.Contexts = append(config.Contexts, ArgoConfigContext{
			Name:   contextName,
			User:   contextName,
			Server: normalizedHost,
		})
	}

	// Set as current context
	config.CurrentContext = contextName

	// Write config file
	configJSON, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, configJSON, 0600); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	return nil
}

// performSSOLogin performs the SSO login flow with ArgoCD using the IAP token for proxy authentication
func performSSOLogin(ctx context.Context, host, iapToken string, verbose bool, cmd *cobra.Command) (string, error) {
	// Create ArgoCD client with IAP authentication
	clientOpts := &argocdclient.ClientOptions{
		ServerAddr:      host,
		GRPCWeb:         true,
		GRPCWebRootPath: "",
		Insecure:        false,
		Headers: []string{
			fmt.Sprintf("Proxy-Authorization: Bearer %s", iapToken),
		},
	}

	client, err := argocdclient.NewClient(clientOpts)
	if err != nil {
		return "", fmt.Errorf("create ArgoCD client: %w", err)
	}

	// Get ArgoCD settings to retrieve OIDC configuration
	if verbose {
		if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "Fetching ArgoCD settings...\n"); err != nil {
			return "", fmt.Errorf("failed writing to stderr: %w", err)
		}
	}

	conn, settingsClient, err := client.NewSettingsClient()
	if err != nil {
		return "", fmt.Errorf("create settings client: %w", err)
	}
	defer conn.Close()

	settings, err := settingsClient.Get(ctx, &settingspkg.SettingsQuery{})
	if err != nil {
		return "", fmt.Errorf("get settings: %w", err)
	}

	if settings.OIDCConfig == nil && settings.DexConfig == nil {
		return "", fmt.Errorf("ArgoCD server is not configured with SSO/OIDC")
	}

	if verbose {
		if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "Starting OAuth2 SSO flow...\n"); err != nil {
			return "", fmt.Errorf("failed writing to stderr: %w", err)
		}
	}

	// Get HTTP client with IAP authentication
	httpClient, err := client.HTTPClient()
	if err != nil {
		return "", fmt.Errorf("create HTTP client: %w", err)
	}

	// Create OIDC context with our HTTP client that includes IAP headers
	ctx = oidc.ClientContext(ctx, httpClient)

	// Perform OAuth2 login flow using the ArgoCD client
	oauth2conf, provider, err := client.OIDCConfig(ctx, settings)
	if err != nil {
		return "", fmt.Errorf("get OIDC config: %w", err)
	}

	// Use local server for OAuth2 callback
	port := 8085
	tokenString, _, err := performOAuth2Flow(ctx, port, settings.OIDCConfig, oauth2conf, provider, httpClient, verbose, cmd)
	if err != nil {
		return "", fmt.Errorf("OAuth2 flow: %w", err)
	}

	return tokenString, nil
}

// performOAuth2Flow performs the OAuth2 authorization code flow
func performOAuth2Flow(
	ctx context.Context,
	port int,
	oidcSettings *settingspkg.OIDCConfig,
	oauth2conf *oauth2.Config,
	provider *oidc.Provider,
	httpClient *http.Client,
	verbose bool,
	cmd *cobra.Command,
) (string, string, error) {
	oauth2conf.RedirectURL = fmt.Sprintf("http://localhost:%d/auth/callback", port)

	// State nonce for OAuth2 security
	stateNonce, err := rand.String(24)
	if err != nil {
		return "", "", fmt.Errorf("generate state nonce: %w", err)
	}

	var tokenString string
	var refreshToken string
	completionChan := make(chan error)

	// PKCE implementation
	codeVerifier, err := rand.StringFromCharset(43, "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~")
	if err != nil {
		return "", "", fmt.Errorf("generate code verifier: %w", err)
	}
	codeChallengeHash := sha256.Sum256([]byte(codeVerifier))
	codeChallenge := base64.RawURLEncoding.EncodeToString(codeChallengeHash[:])

	// Callback handler
	http.HandleFunc("/auth/callback", func(w http.ResponseWriter, r *http.Request) {
		if formErr := r.FormValue("error"); formErr != "" {
			errMsg := fmt.Sprintf("%s: %s", formErr, r.FormValue("error_description"))
			http.Error(w, html.EscapeString(errMsg), http.StatusBadRequest)
			completionChan <- fmt.Errorf("%s", errMsg)
			return
		}

		if state := r.FormValue("state"); state != stateNonce {
			http.Error(w, "Invalid state nonce", http.StatusBadRequest)
			completionChan <- fmt.Errorf("invalid state nonce")
			return
		}

		code := r.FormValue("code")
		if code == "" {
			http.Error(w, "No code in request", http.StatusBadRequest)
			completionChan <- fmt.Errorf("no code in request")
			return
		}

		// Exchange code for token using our HTTP client with IAP headers
		exchangeCtx := context.WithValue(ctx, oauth2.HTTPClient, httpClient)
		opts := []oauth2.AuthCodeOption{oauth2.SetAuthURLParam("code_verifier", codeVerifier)}
		token, err := oauth2conf.Exchange(exchangeCtx, code, opts...)
		if err != nil {
			http.Error(w, "Failed to exchange code for token", http.StatusInternalServerError)
			completionChan <- fmt.Errorf("exchange code: %w", err)
			return
		}

		idToken, ok := token.Extra("id_token").(string)
		if !ok {
			http.Error(w, "No id_token in response", http.StatusInternalServerError)
			completionChan <- fmt.Errorf("no id_token in response")
			return
		}

		tokenString = idToken
		refreshToken = token.RefreshToken

		fmt.Fprintf(w, "<h1>Authentication successful!</h1><p>You can close this window and return to the CLI.</p>")
		completionChan <- nil
	})

	// Start local server
	server := &http.Server{Addr: fmt.Sprintf(":%d", port)}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			completionChan <- fmt.Errorf("HTTP server error: %w", err)
		}
	}()
	defer server.Shutdown(ctx)

	// Build authorization URL
	authCodeOpts := []oauth2.AuthCodeOption{
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	}
	authCodeURL := oauth2conf.AuthCodeURL(stateNonce, authCodeOpts...)

	if verbose {
		if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "Opening browser for SSO login...\n"); err != nil {
			return "", "", fmt.Errorf("failed writing to stderr: %w", err)
		}
		if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "If browser doesn't open, visit: %s\n", authCodeURL); err != nil {
			return "", "", fmt.Errorf("failed writing to stderr: %w", err)
		}
	} else {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Opening browser for SSO login...\n"); err != nil {
			return "", "", fmt.Errorf("failed writing to stdout: %w", err)
		}
	}

	// Open browser
	if err := open.Run(authCodeURL); err != nil {
		if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "Failed to open browser automatically. Please visit:\n%s\n", authCodeURL); err != nil {
			return "", "", fmt.Errorf("failed writing to stderr: %w", err)
		}
	}

	// Wait for callback
	if err := <-completionChan; err != nil {
		return "", "", err
	}

	return tokenString, refreshToken, nil
}
