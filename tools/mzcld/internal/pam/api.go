package pam

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	pamsdk "cloud.google.com/go/privilegedaccessmanager/apiv1"
	pamv1pb "cloud.google.com/go/privilegedaccessmanager/apiv1/privilegedaccessmanagerpb"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/durationpb"
)

// Type aliases so callers use pam.Grant / pam.Entitlement without importing pamv1pb.
type Grant = pamv1pb.Grant
type Entitlement = pamv1pb.Entitlement

func newClient(ctx context.Context) (*pamsdk.Client, error) {
	return pamsdk.NewClient(ctx)
}

// entitlementResourceName returns the PAM resource name for an entitlement.
func entitlementResourceName(projectID, entitlement string) string {
	return fmt.Sprintf("projects/%s/locations/global/entitlements/%s", projectID, entitlement)
}

// GetEntitlement fetches a single PAM entitlement.
func GetEntitlement(ctx context.Context, projectID, entitlementName string) (*Entitlement, error) {
	client, err := newClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create PAM client: %w", err)
	}
	defer client.Close() //nolint:errcheck

	return client.GetEntitlement(ctx, &pamv1pb.GetEntitlementRequest{
		Name: entitlementResourceName(projectID, entitlementName),
	})
}

// CreateGrant requests a new PAM grant against the given entitlement.
func CreateGrant(ctx context.Context, projectID, entitlement, justification string, duration time.Duration) (*Grant, error) {
	client, err := newClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create PAM client: %w", err)
	}
	defer client.Close() //nolint:errcheck

	return client.CreateGrant(ctx, &pamv1pb.CreateGrantRequest{
		Parent: entitlementResourceName(projectID, entitlement),
		Grant: &pamv1pb.Grant{
			RequestedDuration: durationpb.New(duration),
			Justification: &pamv1pb.Justification{
				Justification: &pamv1pb.Justification_UnstructuredJustification{
					UnstructuredJustification: justification,
				},
			},
		},
	})
}

// GetGrant fetches a PAM grant by its full resource name.
func GetGrant(ctx context.Context, grantName string) (*Grant, error) {
	client, err := newClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create PAM client: %w", err)
	}
	defer client.Close() //nolint:errcheck

	return client.GetGrant(ctx, &pamv1pb.GetGrantRequest{
		Name: grantName,
	})
}

// RevokeGrant revokes a PAM grant by its full resource name.
func RevokeGrant(ctx context.Context, grantName string) error {
	client, err := newClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create PAM client: %w", err)
	}
	defer client.Close() //nolint:errcheck

	op, err := client.RevokeGrant(ctx, &pamv1pb.RevokeGrantRequest{
		Name: grantName,
	})
	if err != nil {
		return err
	}
	_, err = op.Wait(ctx)
	return err
}

// --- Grant helpers -----------------------------------------------------------

// activationTime returns the time the grant was activated by scanning the
// timeline for an Activated event. Falls back to UpdateTime if no activation
// event is found (e.g. auto-approved grants).
func activationTime(g *Grant) time.Time {
	if tl := g.GetTimeline(); tl != nil {
		for _, ev := range tl.GetEvents() {
			if ev.GetActivated() != nil && ev.GetEventTime() != nil {
				return ev.GetEventTime().AsTime()
			}
		}
	}
	if g.GetUpdateTime() != nil {
		return g.GetUpdateTime().AsTime()
	}
	return time.Time{}
}

// IsExpired reports whether the grant has passed its end time.
func IsExpired(g *Grant) bool {
	start := activationTime(g)
	if start.IsZero() || g.GetRequestedDuration() == nil {
		return false
	}
	end := start.Add(g.GetRequestedDuration().AsDuration())
	return time.Now().After(end)
}

// IsApprovalAwaited reports whether the grant is waiting for approval.
func IsApprovalAwaited(g *Grant) bool {
	return g.GetState() == pamv1pb.Grant_APPROVAL_AWAITED
}

// TimeRemaining returns the duration until the grant expires.
func TimeRemaining(g *Grant) time.Duration {
	start := activationTime(g)
	if start.IsZero() || g.GetRequestedDuration() == nil {
		return 0
	}
	end := start.Add(g.GetRequestedDuration().AsDuration())
	return time.Until(end)
}

// ExtractParts parses the grant Name and returns (projectID, entitlementName, error).
func ExtractParts(g *Grant) (string, string, error) {
	re := regexp.MustCompile(`^projects/([^/]+)/locations/[^/]+/entitlements/([^/]+)/grants/[^/]+$`)
	matches := re.FindStringSubmatch(g.GetName())
	if len(matches) < 3 {
		return "", "", fmt.Errorf("grant name does not match expected pattern: %s", g.GetName())
	}
	return matches[1], matches[2], nil
}

// GetJustification returns the unstructured justification text from a grant.
func GetJustification(g *Grant) string {
	j := g.GetJustification()
	if j == nil {
		return ""
	}
	if u, ok := j.Justification.(*pamv1pb.Justification_UnstructuredJustification); ok {
		return u.UnstructuredJustification
	}
	return ""
}

// StateLabel maps the grant state enum to a display string.
func StateLabel(g *Grant) string {
	switch g.GetState() {
	case pamv1pb.Grant_ACTIVE, pamv1pb.Grant_SCHEDULED:
		return "Active"
	case pamv1pb.Grant_APPROVAL_AWAITED:
		return "Awaiting Approval"
	case pamv1pb.Grant_DENIED:
		return "Denied"
	case pamv1pb.Grant_REVOKED:
		return "Revoked"
	case pamv1pb.Grant_EXPIRED:
		return "Expired"
	default:
		return strings.TrimPrefix(g.GetState().String(), "Grant_")
	}
}

// Roles returns the IAM role names from the grant's role bindings.
func Roles(g *Grant) []string {
	pa := g.GetPrivilegedAccess()
	if pa == nil {
		return nil
	}
	iam := pa.GetGcpIamAccess()
	if iam == nil {
		return nil
	}
	roles := make([]string, 0, len(iam.GetRoleBindings()))
	for _, rb := range iam.GetRoleBindings() {
		roles = append(roles, rb.GetRole())
	}
	return roles
}

// --- Entitlement helpers -----------------------------------------------------

// MaxDuration returns the maximum request duration for an entitlement.
func MaxDuration(e *Entitlement) time.Duration {
	if d := e.GetMaxRequestDuration(); d != nil {
		return d.AsDuration()
	}
	return 0
}

// --- Serialization -----------------------------------------------------------

// MarshalGrants serializes grants using protojson via ListGrantsResponse.
func MarshalGrants(grants []*Grant) ([]byte, error) {
	resp := &pamv1pb.ListGrantsResponse{Grants: grants}
	return protojson.Marshal(resp)
}

// UnmarshalGrants deserializes grants from MarshalGrants output.
func UnmarshalGrants(data []byte) ([]*Grant, error) {
	resp := &pamv1pb.ListGrantsResponse{}
	if err := protojson.Unmarshal(data, resp); err != nil {
		return nil, err
	}
	return resp.GetGrants(), nil
}
