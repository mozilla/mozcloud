// Package gcp provides helpers for authenticating with GCP and loading
// entitlement data for the current user.
package gcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"cloud.google.com/go/storage"
	"charm.land/huh/v2"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"

	"github.com/mozilla/mozcloud/tools/mzcld/internal/cache"
)

// --- Auth mode ---------------------------------------------------------------

// AuthMode controls how mzcld obtains GCP credentials.
type AuthMode string

const (
	// AuthModeGcloud shells out to gcloud CLI for tokens. Default. RAPT-safe.
	AuthModeGcloud AuthMode = "gcloud"
	// AuthModeADC uses Application Default Credentials. For CI/service accounts.
	AuthModeADC AuthMode = "adc"
)

var activeAuthMode AuthMode = AuthModeGcloud

// SetAuthMode sets the global authentication mode.
func SetAuthMode(m AuthMode) { activeAuthMode = m }

// GetAuthMode returns the current authentication mode.
func GetAuthMode() AuthMode { return activeAuthMode }

// --- Token sources -----------------------------------------------------------

// gcloudTokenSource delegates token generation to the gcloud CLI,
// which handles RAPT and security-key reauthentication transparently.
type gcloudTokenSource struct{}

func (g *gcloudTokenSource) Token() (*oauth2.Token, error) {
	out, err := exec.Command("gcloud", "auth", "print-access-token").Output()
	if err != nil {
		return nil, fmt.Errorf("gcloud auth print-access-token failed: %w", err)
	}
	return &oauth2.Token{AccessToken: strings.TrimSpace(string(out))}, nil
}

// adcTokenSource uses Application Default Credentials.
type adcTokenSource struct{}

func (a *adcTokenSource) Token() (*oauth2.Token, error) {
	ctx := context.Background()
	creds, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return nil, fmt.Errorf("failed to find default credentials: %w", err)
	}
	return creds.TokenSource.Token()
}

// TokenSource returns the active token source based on the current auth mode.
func TokenSource() oauth2.TokenSource {
	if activeAuthMode == AuthModeADC {
		return &adcTokenSource{}
	}
	return &gcloudTokenSource{}
}

// ClientOption returns a google API client option using the active auth mode.
// All GCP SDK clients should use this for consistent authentication.
func ClientOption() option.ClientOption {
	return option.WithTokenSource(TokenSource())
}

// --- Auth flow ---------------------------------------------------------------

// EnsureAuthenticated returns the current user's email, running the appropriate
// gcloud auth flow interactively if credentials are missing or expired.
func EnsureAuthenticated() (string, error) {
	email, err := GetUserEmail()
	if err == nil {
		return email, nil
	}
	if !isAuthError(err) {
		return "", err
	}

	authCmd := "gcloud auth login"
	if activeAuthMode == AuthModeADC {
		authCmd = "gcloud auth application-default login"
	}

	var runAuth bool
	if promptErr := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Not authenticated").
				Description(fmt.Sprintf("Run `%s` now?", authCmd)).
				Value(&runAuth),
		),
	).Run(); promptErr != nil {
		return "", promptErr
	}
	if !runAuth {
		return "", fmt.Errorf("authentication required: run `%s`", authCmd)
	}

	args := strings.Fields(authCmd)
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("gcloud auth failed: %w", err)
	}

	return GetUserEmail()
}

// isAuthError reports whether err looks like a missing-credentials or
// expired-token error (as opposed to a network or domain error).
func isAuthError(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "could not find default credentials") ||
		strings.Contains(msg, "oauth2: token expired") ||
		strings.Contains(msg, "Unauthenticated") ||
		strings.Contains(msg, "reauth") ||
		strings.Contains(msg, "print-access-token failed") ||
		strings.Contains(msg, "not from mozilla.com domain")
}

// --- User info ---------------------------------------------------------------

type gcpUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Picture       string `json:"picture"`
	Hd            string `json:"hd"`
}

// GetUserEmail returns the email address for the currently authenticated GCP
// user. It validates that the account belongs to the mozilla.com domain.
// Uses the active auth mode's token source.
func GetUserEmail() (string, error) {
	token, err := TokenSource().Token()
	if err != nil {
		return "", fmt.Errorf("failed to get token: %w", err)
	}

	req, err := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v1/userinfo", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	var info gcpUserInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return "", fmt.Errorf("failed to unmarshal userinfo JSON: %w", err)
	}

	if info.Hd != "mozilla.com" {
		return "", fmt.Errorf("user is not from mozilla.com domain, got %s", info.Email)
	}
	return info.Email, nil
}

// --- Entitlement types -------------------------------------------------------

// UserEntitlement is a single entitlement available to the logged-in user.
type UserEntitlement struct {
	Appcode     string `json:"appcode"`
	ProjectID   string `json:"project_id"`
	Entitlement string `json:"entitlement"`
	Realm       string `json:"realm"`
}

// EntitlementData is the raw shape of the per-user entitlements JSON file
// stored in GCS.
type EntitlementData map[string]Environments

// Environments groups nonprod and prod environment details.
type Environments struct {
	NonProd EnvironmentDetails `json:"nonprod"`
	Prod    EnvironmentDetails `json:"prod"`
}

// EnvironmentDetails holds the GCP project ID and list of entitlement names
// for one environment.
type EnvironmentDetails struct {
	Entitlements []string `json:"entitlements"`
	ProjectID    string   `json:"project_id"`
}

// --- Project cache -----------------------------------------------------------

const dataCacheFile = "data.json"

// ProjectCache persists the last entitlement selection made by the user.
type ProjectCache struct {
	LastChoice UserEntitlement `json:"last_choice"`
}

// Save persists the ProjectCache to ~/.mzcld/data.json.
func (p *ProjectCache) Save() error {
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}
	return cache.Save(dataCacheFile, data)
}

// Load reads the ProjectCache from ~/.mzcld/data.json. Returns an empty cache
// (not an error) when the file does not exist.
func Load() (ProjectCache, error) {
	if !cache.Exists(dataCacheFile) {
		return ProjectCache{}, nil
	}
	data, err := cache.Load(dataCacheFile)
	if err != nil {
		return ProjectCache{}, err
	}
	var pc ProjectCache
	if err := json.Unmarshal(data, &pc); err != nil {
		return ProjectCache{}, err
	}
	return pc, nil
}

// --- GCS helpers -------------------------------------------------------------

// LoadEntitlements always downloads the user's entitlement list fresh from GCS.
func LoadEntitlements(ctx context.Context, bucketName, userEmail string) ([]UserEntitlement, error) {
	client, err := storage.NewClient(ctx, ClientOption())
	if err != nil {
		return nil, fmt.Errorf("failed to create storage client: %w", err)
	}
	defer client.Close() //nolint:errcheck

	objectName := fmt.Sprintf("%s.json", userEmail)
	reader, err := client.Bucket(bucketName).Object(objectName).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read entitlements object: %w", err)
	}
	defer reader.Close() //nolint:errcheck

	raw, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read entitlements: %w", err)
	}

	var data EntitlementData
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("failed to parse entitlements JSON: %w", err)
	}

	return reformatEntitlementData(data), nil
}

// reformatEntitlementData flattens EntitlementData into a []UserEntitlement slice.
func reformatEntitlementData(data EntitlementData) []UserEntitlement {
	var list []UserEntitlement
	for appcode, envs := range data {
		for _, e := range envs.NonProd.Entitlements {
			list = append(list, UserEntitlement{
				Appcode:     appcode,
				ProjectID:   envs.NonProd.ProjectID,
				Entitlement: e,
				Realm:       "nonprod",
			})
		}
		for _, e := range envs.Prod.Entitlements {
			list = append(list, UserEntitlement{
				Appcode:     appcode,
				ProjectID:   envs.Prod.ProjectID,
				Entitlement: e,
				Realm:       "prod",
			})
		}
	}
	return list
}
