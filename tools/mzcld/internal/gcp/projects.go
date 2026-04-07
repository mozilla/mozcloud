package gcp

import (
	"context"
	"fmt"
	"sort"
	"strings"

	resourcemanager "cloud.google.com/go/resourcemanager/apiv3"
	"cloud.google.com/go/resourcemanager/apiv3/resourcemanagerpb"
	"github.com/mozilla/mozcloud/tools/mzcld/internal/gsm"
	"google.golang.org/api/iterator"
)

// ListAccessibleProjects returns project IDs the caller has access to,
// sorted alphabetically. Uses the gcloud token source to handle RAPT.
func ListAccessibleProjects(ctx context.Context) ([]string, error) {
	client, err := resourcemanager.NewProjectsClient(ctx, gsm.ClientOption())
	if err != nil {
		return nil, fmt.Errorf("failed to create resource manager client: %w", err)
	}
	defer client.Close() //nolint:errcheck

	it := client.SearchProjects(ctx, &resourcemanagerpb.SearchProjectsRequest{})

	var projects []string
	for {
		p, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list projects: %w", err)
		}
		if strings.EqualFold(p.State.String(), "ACTIVE") {
			projects = append(projects, p.ProjectId)
		}
	}
	sort.Strings(projects)
	return projects, nil
}
