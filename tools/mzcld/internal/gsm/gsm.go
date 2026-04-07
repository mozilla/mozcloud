// Package gsm provides helpers for interacting with Google Secret Manager.
package gsm

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/mozilla/mozcloud/tools/mzcld/internal/gcp"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// VersionInfo holds metadata about a single secret version.
type VersionInfo struct {
	Name    string // full resource name
	Version string // just the version number (e.g. "1", "2")
	State   string // ENABLED, DISABLED, DESTROYED
	Created time.Time
}

// Client wraps a Secret Manager client for reuse across operations.
type Client struct {
	sm *secretmanager.Client
}

// NewClient creates a new GSM client using gcloud token auth.
func NewClient(ctx context.Context) (*Client, error) {
	sm, err := secretmanager.NewClient(ctx, gcp.ClientOption())
	if err != nil {
		return nil, fmt.Errorf("failed to create secretmanager client: %w", err)
	}
	return &Client{sm: sm}, nil
}

// Close releases resources held by the client.
func (c *Client) Close() error {
	return c.sm.Close()
}

// ListSecrets returns the short names of all secrets in a project.
func (c *Client) ListSecrets(ctx context.Context, projectID string) ([]string, error) {
	parent := fmt.Sprintf("projects/%s", projectID)
	it := c.sm.ListSecrets(ctx, &secretmanagerpb.ListSecretsRequest{Parent: parent})

	var names []string
	for {
		s, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list secrets: %w", err)
		}
		parts := strings.Split(s.Name, "/")
		names = append(names, parts[len(parts)-1])
	}
	sort.Strings(names)
	return names, nil
}

// GetSecretVersion returns the payload bytes for a specific version of a secret.
// Pass "latest" as version to get the most recent version.
func (c *Client) GetSecretVersion(ctx context.Context, projectID, secretName, version string) ([]byte, error) {
	name := fmt.Sprintf("projects/%s/secrets/%s/versions/%s", projectID, secretName, version)
	resp, err := c.sm.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{Name: name})
	if err != nil {
		return nil, fmt.Errorf("failed to access secret version: %w", err)
	}
	return resp.Payload.Data, nil
}

// AddSecretVersion adds a new version with the given payload data.
func (c *Client) AddSecretVersion(ctx context.Context, projectID, secretName string, data []byte) error {
	parent := fmt.Sprintf("projects/%s/secrets/%s", projectID, secretName)
	_, err := c.sm.AddSecretVersion(ctx, &secretmanagerpb.AddSecretVersionRequest{
		Parent:  parent,
		Payload: &secretmanagerpb.SecretPayload{Data: data},
	})
	if err != nil {
		return fmt.Errorf("failed to add secret version: %w", err)
	}
	return nil
}

// CreateSecret creates a new secret with automatic replication.
func (c *Client) CreateSecret(ctx context.Context, projectID, secretName string) error {
	parent := fmt.Sprintf("projects/%s", projectID)
	_, err := c.sm.CreateSecret(ctx, &secretmanagerpb.CreateSecretRequest{
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
func (c *Client) ListVersions(ctx context.Context, projectID, secretName string) ([]VersionInfo, error) {
	parent := fmt.Sprintf("projects/%s/secrets/%s", projectID, secretName)
	it := c.sm.ListSecretVersions(ctx, &secretmanagerpb.ListSecretVersionsRequest{Parent: parent})

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

	// Sort newest first by version number (descending, numeric).
	sort.Slice(versions, func(i, j int) bool {
		vi, _ := strconv.Atoi(versions[i].Version)
		vj, _ := strconv.Atoi(versions[j].Version)
		return vi > vj
	})

	return versions, nil
}

// SecretExists checks whether a secret with the given name exists in the project.
func (c *Client) SecretExists(ctx context.Context, projectID, secretName string) (bool, error) {
	name := fmt.Sprintf("projects/%s/secrets/%s", projectID, secretName)
	_, err := c.sm.GetSecret(ctx, &secretmanagerpb.GetSecretRequest{Name: name})
	if err != nil {
		if st, ok := status.FromError(err); ok && st.Code() == codes.NotFound {
			return false, nil
		}
		return false, fmt.Errorf("failed to check secret existence: %w", err)
	}
	return true, nil
}
