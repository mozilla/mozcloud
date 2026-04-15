package pam

import (
	"testing"
	"time"

	pamv1pb "cloud.google.com/go/privilegedaccessmanager/apiv1/privilegedaccessmanagerpb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestExtractParts(t *testing.T) {
	g := &Grant{Name: "projects/my-proj/locations/global/entitlements/my-ent/grants/abc123"}
	proj, ent, err := ExtractParts(g)
	if err != nil {
		t.Fatal(err)
	}
	if proj != "my-proj" {
		t.Errorf("project = %q, want my-proj", proj)
	}
	if ent != "my-ent" {
		t.Errorf("entitlement = %q, want my-ent", ent)
	}
}

func TestExtractParts_invalid(t *testing.T) {
	g := &Grant{Name: "invalid/name"}
	_, _, err := ExtractParts(g)
	if err == nil {
		t.Error("expected error for invalid grant name")
	}
}

func TestIsExpired(t *testing.T) {
	now := time.Now()

	t.Run("active grant", func(t *testing.T) {
		g := makeGrant(now.Add(-30*time.Minute), 1*time.Hour, nil)
		if IsExpired(g) {
			t.Error("expected not expired")
		}
	})

	t.Run("expired grant", func(t *testing.T) {
		g := makeGrant(now.Add(-2*time.Hour), 1*time.Hour, nil)
		if !IsExpired(g) {
			t.Error("expected expired")
		}
	})
}

func TestTimeRemaining(t *testing.T) {
	now := time.Now()
	g := makeGrant(now.Add(-30*time.Minute), 1*time.Hour, nil)
	remaining := TimeRemaining(g)
	if remaining < 29*time.Minute || remaining > 31*time.Minute {
		t.Errorf("remaining = %v, expected ~30m", remaining)
	}
}

func TestIsExpired_usesActivationTime(t *testing.T) {
	now := time.Now()

	// Grant was updated 2 hours ago (e.g. approved), but activated 30 minutes ago.
	// With 1 hour duration, it should NOT be expired.
	activated := now.Add(-30 * time.Minute)
	g := makeGrant(now.Add(-2*time.Hour), 1*time.Hour, &activated)
	if IsExpired(g) {
		t.Error("expected not expired when using activation time")
	}
}

func makeGrant(updateTime time.Time, duration time.Duration, activatedAt *time.Time) *Grant {
	g := &Grant{
		UpdateTime:        timestamppb.New(updateTime),
		RequestedDuration: durationpb.New(duration),
	}
	if activatedAt != nil {
		g.Timeline = &pamv1pb.Grant_Timeline{
			Events: []*pamv1pb.Grant_Timeline_Event{
				{
					Event:     &pamv1pb.Grant_Timeline_Event_Requested_{},
					EventTime: timestamppb.New(updateTime),
				},
				{
					Event:     &pamv1pb.Grant_Timeline_Event_Activated_{Activated: &pamv1pb.Grant_Timeline_Event_Activated{}},
					EventTime: timestamppb.New(*activatedAt),
				},
			},
		}
	}
	return g
}
