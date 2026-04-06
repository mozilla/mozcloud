package gsm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/mozilla/mozcloud/tools/mzcld/internal/cache"
	"github.com/mozilla/mozcloud/tools/mzcld/internal/gsm"
	"github.com/mozilla/mozcloud/tools/mzcld/internal/ui"
)

const gsmCacheFile = "gsm-cache.json"

// gsmCache persists the last project+secret selection.
type gsmCache struct {
	ProjectID string `json:"project_id"`
	Secret    string `json:"secret"`
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

// selectProject returns a GCP project ID. It uses the flag value if provided,
// offers the cached last-choice, or prompts for manual input.
func selectProject(_ context.Context, flagProject string) (string, error) {
	if flagProject != "" {
		return flagProject, nil
	}

	cached := loadGSMCache()

	// Offer last-choice shortcut if we have one.
	if cached.ProjectID != "" {
		const optLast = "last"
		const optNew = "new"
		var quickChoice string

		lastLabel := fmt.Sprintf("↩ last: %s", cached.ProjectID)
		if err := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Project").
					Options(
						huh.NewOption(lastLabel, optLast),
						huh.NewOption("Enter different...", optNew),
					).
					Value(&quickChoice),
			),
		).Run(); err != nil {
			return "", err
		}
		if quickChoice == optLast {
			return cached.ProjectID, nil
		}
	}

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
func selectSecret(ctx context.Context, projectID, flagSecret string, allowCreate bool) (string, bool, error) {
	if flagSecret != "" {
		return flagSecret, false, nil
	}

	ui.Info("Loading secrets...")
	names, err := gsm.ListSecrets(ctx, projectID)
	if err != nil {
		return "", false, err
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
