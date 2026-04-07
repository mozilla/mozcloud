package gsm

import (
	"context"
	"encoding/json"
	"fmt"

	"charm.land/huh/v2"
	"charm.land/huh/v2/spinner"
	"github.com/mozilla/mozcloud/tools/mzcld/internal/cache"
	"github.com/mozilla/mozcloud/tools/mzcld/internal/gcp"
	"github.com/mozilla/mozcloud/tools/mzcld/internal/gsm"
	"github.com/mozilla/mozcloud/tools/mzcld/internal/ui"
)

// runWithSpinner runs action with a spinner title, or silently if stdout is piped.
func runWithSpinner(ctx context.Context, title string, action func()) {
	if ui.IsPiped() {
		action()
		return
	}
	_ = spinner.New().
		Title(title).
		Context(ctx).
		Action(action).
		Run()
}

const (
	gsmCacheFile  = "gsm-cache.json"
	maxRecentProj = 10
)

// gsmCache persists recent project+secret selections.
type gsmCache struct {
	RecentProjects []string `json:"recent_projects"`
	Secret         string   `json:"secret"`
}

func loadGSMCache() gsmCache {
	data, err := cache.Load(gsmCacheFile)
	if err != nil {
		return gsmCache{}
	}
	var c gsmCache
	if err := json.Unmarshal(data, &c); err != nil {
		return gsmCache{}
	}
	return c
}

func saveGSMCache(c gsmCache) {
	data, err := json.Marshal(c)
	if err != nil {
		return
	}
	_ = cache.Save(gsmCacheFile, data)
}

// cacheSelection loads the cache, pushes the project to recents, updates the
// secret, and saves.
func cacheSelection(projectID, secret string) {
	c := loadGSMCache()
	pushRecentProject(&c, projectID)
	c.Secret = secret
	saveGSMCache(c)
}

// pushRecentProject adds projectID to the front of the recents list,
// deduplicating and capping at maxRecentProj.
func pushRecentProject(c *gsmCache, projectID string) {
	filtered := []string{projectID}
	for _, p := range c.RecentProjects {
		if p != projectID {
			filtered = append(filtered, p)
		}
	}
	if len(filtered) > maxRecentProj {
		filtered = filtered[:maxRecentProj]
	}
	c.RecentProjects = filtered
}

// selectProject returns a GCP project ID. It uses the flag value if provided,
// otherwise shows a filterable picker with recent projects + all accessible projects.
func selectProject(ctx context.Context, flagProject string) (string, error) {
	if flagProject != "" {
		return flagProject, nil
	}

	cached := loadGSMCache()

	// Build the option list: recents first (marked), then all accessible projects.
	var allProjects []string
	var fetchErr error
	runWithSpinner(ctx, "Loading projects...", func() {
		allProjects, fetchErr = gcp.ListAccessibleProjects(ctx)
	})
	if fetchErr != nil {
		ui.Warn("Could not list projects: " + fetchErr.Error())
	}

	seen := make(map[string]struct{})
	var opts []huh.Option[string]

	// Add recent projects at the top.
	for _, p := range cached.RecentProjects {
		label := fmt.Sprintf("★ %s", p)
		opts = append(opts, huh.NewOption(label, p))
		seen[p] = struct{}{}
	}

	// Add remaining accessible projects.
	for _, p := range allProjects {
		if _, ok := seen[p]; !ok {
			opts = append(opts, huh.NewOption(p, p))
			seen[p] = struct{}{}
		}
	}

	// If we have projects to choose from, show the filterable picker.
	if len(opts) > 0 {
		var selected string
		if err := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Project").
					Options(opts...).
					Filtering(true).
					Height(15).
					Value(&selected),
			),
		).Run(); err != nil {
			return "", err
		}
		return selected, nil
	}

	// Fallback: manual input if no projects could be listed.
	var projectID string
	if err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("GCP Project ID").
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("project ID is required")
					}
					return nil
				}).
				Value(&projectID),
		),
	).Run(); err != nil {
		return "", err
	}
	return projectID, nil
}

// selectSecret interactively picks a secret from the project.
// If flagSecret is non-empty it is returned directly.
// When allowCreate is true, a "Create new..." option is prepended.
func selectSecret(ctx context.Context, client *gsm.Client, projectID, flagSecret string, allowCreate bool) (string, bool, error) {
	if flagSecret != "" {
		return flagSecret, false, nil
	}

	var names []string
	var secretErr error
	runWithSpinner(ctx, "Loading secrets...", func() {
		names, secretErr = client.ListSecrets(ctx, projectID)
	})
	if secretErr != nil {
		return "", false, secretErr
	}

	const createNew = "__create_new__"
	var opts []huh.Option[string]
	if allowCreate {
		opts = append(opts, huh.NewOption("+ Create new secret...", createNew))
	}
	for _, n := range names {
		opts = append(opts, huh.NewOption(n, n))
	}

	if len(opts) == 0 {
		return "", false, fmt.Errorf("no secrets found in project %s", projectID)
	}

	var selected string
	if err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Secret").
				Options(opts...).
				Filtering(true).
				Value(&selected),
		),
	).Run(); err != nil {
		return "", false, err
	}

	if selected == createNew {
		var name string
		if err := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("New secret name").
					Validate(func(s string) error {
						if s == "" {
							return fmt.Errorf("name is required")
						}
						return nil
					}).
					Value(&name),
			),
		).Run(); err != nil {
			return "", false, err
		}
		return name, true, nil
	}

	return selected, false, nil
}
