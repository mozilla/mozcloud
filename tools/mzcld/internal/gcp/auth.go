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
	"github.com/charmbracelet/huh"
	"golang.org/x/oauth2/google"

	"github.com/mozilla/mozcloud/tools/mzcld/internal/cache"
)

// --- Auth --------------------------------------------------------------------

// EnsureAuthenticated returns the current user's email, running the gcloud
// auth flow interactively if credentials are missing or expired.
func EnsureAuthenticated() (string, error) {
	email, err := GetUserEmail()
	if err == nil {
		return email, nil
	}
	if !isAuthError(err) {
		return "", err
	}

	var runAuth bool
	if promptErr := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Not authenticated").
				Description("Run `gcloud auth application-default login` now?").
				Value(&runAuth),
		),
	).Run(); promptErr != nil {
		return "", promptErr
	}
	if !runAuth {
		return "", fmt.Errorf("authentication required: run `gcloud auth application-default login`")
	}

	cmd := exec.Command("gcloud", "auth", "application-default", "login")
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
func GetUserEmail() (string, error) {
	ctx := context.Background()
	creds, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return "", fmt.Errorf("failed to find default credentials: %w", err)
	}
	token, err := creds.TokenSource.Token()
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
	client, err := storage.NewClient(ctx)
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
