package argo

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func NewArgoCLICmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cli [argocd-args...]",
		Short: "Run ArgoCD CLI with IAP authentication",
		Long: `Download and run the ArgoCD CLI for the target instance with automatic IAP authentication.

This command:
  1. Downloads the correct ArgoCD CLI version from the target instance
  2. Caches it in ~/.config/mzcld/bin/ for reuse
  3. Wraps it with IAP authentication headers automatically

The CLI binary is downloaded once per host and reused for subsequent commands.`,
		Example: `  # List applications using ArgoCD CLI
  mzcld argo cli app list

  # Sync an application
  mzcld argo cli app sync my-app

  # Any ArgoCD CLI command works
  mzcld argo cli version
  mzcld argo cli account get-user-info`,
		DisableFlagParsing: true, // Pass all flags to argocd CLI
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check if this is a help request
			for _, arg := range args {
				if arg == "--help" || arg == "-h" || arg == "help" {
					cmd.Help()
					return nil
				}
			}

			ctx := context.Background()

			// Get current context
			currentContext, err := getCurrentContext()
			if err != nil {
				return err
			}

			// Generate IAP token
			iapToken, err := generateIAPToken(ctx, currentContext)
			if err != nil {
				return fmt.Errorf("generate IAP token: %w", err)
			}

			// Ensure ArgoCD CLI is downloaded
			cliPath, err := ensureArgoCLI(ctx, currentContext, iapToken)
			if err != nil {
				return fmt.Errorf("ensure ArgoCD CLI: %w", err)
			}

			// Run ArgoCD CLI with IAP headers
			return runArgoCLI(cliPath, currentContext, iapToken, args)
		},
	}

	return cmd
}

// ensureArgoCLI ensures the ArgoCD CLI binary is downloaded and available
func ensureArgoCLI(ctx context.Context, host, iapToken string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home directory: %w", err)
	}

	// Create bin directory
	binDir := filepath.Join(homeDir, ".config", "mzcld", "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return "", fmt.Errorf("create bin directory: %w", err)
	}

	// Sanitize host for filename (replace dots and slashes)
	sanitizedHost := strings.ReplaceAll(host, ".", "-")
	sanitizedHost = strings.ReplaceAll(sanitizedHost, "/", "-")
	cliPath := filepath.Join(binDir, fmt.Sprintf("%s-argocd", sanitizedHost))

	// Check if already exists
	if _, err := os.Stat(cliPath); err == nil {
		return cliPath, nil
	}

	// Determine platform and architecture
	platform, arch := getPlatformArch()

	// Try common download URL patterns
	downloadURLs := []string{
		fmt.Sprintf("https://%s/download/argocd-%s-%s", host, platform, arch),
		fmt.Sprintf("https://%s/download/argocd-%s%s", host, platform, arch), // without dash
	}

	fmt.Printf("Downloading ArgoCD CLI for %s/%s...\n", platform, arch)

	// Try each download URL until one works
	var lastErr error
	for _, downloadURL := range downloadURLs {
		fmt.Printf("Trying %s...\n", downloadURL)
		if err := downloadFile(ctx, downloadURL, cliPath, iapToken); err != nil {
			lastErr = err
			continue
		}
		// Download successful
		lastErr = nil
		break
	}

	if lastErr != nil {
		return "", fmt.Errorf("download ArgoCD CLI: tried %d URLs, last error: %w", len(downloadURLs), lastErr)
	}

	// Make executable
	if err := os.Chmod(cliPath, 0755); err != nil {
		return "", fmt.Errorf("make executable: %w", err)
	}

	fmt.Printf("ArgoCD CLI downloaded to %s\n", cliPath)

	return cliPath, nil
}

// getPlatformArch returns the platform and architecture for the current system
func getPlatformArch() (string, string) {
	platform := runtime.GOOS
	arch := runtime.GOARCH

	// Map Go arch names to ArgoCD naming
	archMap := map[string]string{
		"amd64": "amd64",
		"arm64": "arm64",
		"386":   "386",
	}

	mappedArch, ok := archMap[arch]
	if !ok {
		mappedArch = arch
	}

	return platform, mappedArch
}

// downloadFile downloads a file with IAP authentication
func downloadFile(ctx context.Context, url, filepath, iapToken string) error {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 5 * time.Minute,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	// Add IAP authentication header
	req.Header.Set("Proxy-Authorization", fmt.Sprintf("Bearer %s", iapToken))

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

// runArgoCLI executes the ArgoCD CLI with IAP authentication
func runArgoCLI(cliPath, host, iapToken string, args []string) error {
	// Prepare ArgoCD CLI command
	cmd := exec.Command(cliPath, args...)

	// Set up environment
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Add ArgoCD server and authentication
	// The ArgoCD CLI expects these environment variables
	cmd.Env = append(cmd.Env, fmt.Sprintf("ARGOCD_SERVER=%s", host))
	cmd.Env = append(cmd.Env, "ARGOCD_GRPC_WEB=true")

	// Add IAP header via ARGOCD_OPTS
	// This passes the header to all ArgoCD CLI requests
	iapHeader := fmt.Sprintf("Proxy-Authorization: Bearer %s", iapToken)
	cmd.Env = append(cmd.Env, fmt.Sprintf("ARGOCD_OPTS=--header \"%s\"", iapHeader))

	// Run the command
	return cmd.Run()
}
