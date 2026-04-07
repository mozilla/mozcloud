// Package gsm provides helpers for interacting with Google Secret Manager.
package gsm

import (
	"context"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"golang.org/x/oauth2"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// GcloudTokenSource delegates token generation to the gcloud CLI,
// which handles RAPT and security-key reauthentication transparently.
type GcloudTokenSource struct{}

func (g *GcloudTokenSource) Token() (*oauth2.Token, error) {
	out, err := exec.Command("gcloud", "auth", "print-access-token").Output()
	if err != nil {
		return nil, fmt.Errorf("gcloud auth print-access-token failed: %w", err)
	}
	return &oauth2.Token{AccessToken: strings.TrimSpace(string(out))}, nil
}

// ClientOption returns a google API client option that uses gcloud for auth.
func ClientOption() option.ClientOption {
	return option.WithTokenSource(&GcloudTokenSource{})
}

// VersionInfo holds metadata about a single secret version.
type VersionInfo struct {
	Name    string // full resource name
	Version string // just the version number (e.g. "1", "2")
	State   string // ENABLED, DISABLED, DESTROYED
	Created time.Time
}

// ListSecrets returns the short names of all secrets in a project.
func ListSecrets(ctx context.Context, projectID string) ([]string, error) {
	client, err := secretmanager.NewClient(ctx, ClientOption())
	if err != nil {
		return nil, fmt.Errorf("failed to create secretmanager client: %w", err)
	}
	defer client.Close() //nolint:errcheck

	parent := fmt.Sprintf("projects/%s", projectID)
	it := client.ListSecrets(ctx, &secretmanagerpb.ListSecretsRequest{Parent: parent})

	var names []string
	for {
		s, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list secrets: %w", err)
		}
		// name is "projects/PROJECT/secrets/NAME" — extract the short name
		parts := strings.Split(s.Name, "/")
		names = append(names, parts[len(parts)-1])
	}
	sort.Strings(names)
	return names, nil
}

// GetSecretVersion returns the payload bytes for a specific version of a secret.
// Pass "latest" as version to get the most recent version.
func GetSecretVersion(ctx context.Context, projectID, secretName, version string) ([]byte, error) {
	client, err := secretmanager.NewClient(ctx, ClientOption())
	if err != nil {
		return nil, fmt.Errorf("failed to create secretmanager client: %w", err)
	}
	defer client.Close() //nolint:errcheck

	name := fmt.Sprintf("projects/%s/secrets/%s/versions/%s", projectID, secretName, version)
	resp, err := client.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{Name: name})
	if err != nil {
		return nil, fmt.Errorf("failed to access secret version: %w", err)
	}
	return resp.Payload.Data, nil
}

// AddSecretVersion adds a new version with the given payload data.
func AddSecretVersion(ctx context.Context, projectID, secretName string, data []byte) error {
	client, err := secretmanager.NewClient(ctx, ClientOption())
	if err != nil {
		return fmt.Errorf("failed to create secretmanager client: %w", err)
	}
	defer client.Close() //nolint:errcheck

	parent := fmt.Sprintf("projects/%s/secrets/%s", projectID, secretName)
	_, err = client.AddSecretVersion(ctx, &secretmanagerpb.AddSecretVersionRequest{
		Parent:  parent,
		Payload: &secretmanagerpb.SecretPayload{Data: data},
	})
	if err != nil {
		return fmt.Errorf("failed to add secret version: %w", err)
	}
	return nil
}

// CreateSecret creates a new secret with automatic replication.
func CreateSecret(ctx context.Context, projectID, secretName string) error {
	client, err := secretmanager.NewClient(ctx, ClientOption())
	if err != nil {
		return fmt.Errorf("failed to create secretmanager client: %w", err)
	}
	defer client.Close() //nolint:errcheck

	parent := fmt.Sprintf("projects/%s", projectID)
	_, err = client.CreateSecret(ctx, &secretmanagerpb.CreateSecretRequest{
		Parent:   parent,
		SecretId: secretName,
		Secret: &secretmanagerpb.Secret{
			Replication: &secretmanagerpb.Replication{
				Replication: &secretmanagerpb.Replication_Automatic_{
					Automatic: &secretmanagerpb.Replication_Automatic{},
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create secret: %w", err)
	}
	return nil
}

// ListVersions returns metadata for all versions of a secret (newest first).
func ListVersions(ctx context.Context, projectID, secretName string) ([]VersionInfo, error) {
	client, err := secretmanager.NewClient(ctx, ClientOption())
	if err != nil {
		return nil, fmt.Errorf("failed to create secretmanager client: %w", err)
	}
	defer client.Close() //nolint:errcheck

	parent := fmt.Sprintf("projects/%s/secrets/%s", projectID, secretName)
	it := client.ListSecretVersions(ctx, &secretmanagerpb.ListSecretVersionsRequest{Parent: parent})

	var versions []VersionInfo
	for {
		v, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list versions: %w", err)
		}
		parts := strings.Split(v.Name, "/")
		ver := parts[len(parts)-1]

		var created time.Time
		if v.CreateTime != nil {
			created = v.CreateTime.AsTime()
		}

		versions = append(versions, VersionInfo{
			Name:    v.Name,
			Version: ver,
			State:   v.State.String(),
			Created: created,
		})
	}

	// Sort newest first by version number (descending).
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Version > versions[j].Version
	})

	return versions, nil
}

// SecretExists checks whether a secret with the given name exists in the project.
func SecretExists(ctx context.Context, projectID, secretName string) (bool, error) {
	client, err := secretmanager.NewClient(ctx, ClientOption())
	if err != nil {
		return false, fmt.Errorf("failed to create secretmanager client: %w", err)
	}
	defer client.Close() //nolint:errcheck

	name := fmt.Sprintf("projects/%s/secrets/%s", projectID, secretName)
	_, err = client.GetSecret(ctx, &secretmanagerpb.GetSecretRequest{Name: name})
	if err != nil {
		if strings.Contains(err.Error(), "NotFound") {
			return false, nil
		}
		return false, fmt.Errorf("failed to check secret existence: %w", err)
	}
	return true, nil
}
