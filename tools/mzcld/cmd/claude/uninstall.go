package claude

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/mozilla/mozcloud/tools/mzcld/internal/executil"
	"github.com/mozilla/mozcloud/tools/mzcld/internal/ui"
	"github.com/spf13/cobra"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove installed Claude skills, agents, and MCP servers",
	Long: `uninstall scans your Claude configuration directory for installed MozCloud
skills and agents (symlinks) and lets you choose which to remove.

Run this command from anywhere inside the MozCloud repository, or pass
--scope explicitly if running outside it.`,
	RunE: runUninstall,
}

var uninstallScopeFlag string

func init() {
	uninstallCmd.Flags().StringVar(&uninstallScopeFlag, "scope", "", "Scope to uninstall from: user or project (skips prompt)")
}

func runUninstall(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	scope := uninstallScopeFlag
	if scope == "" {
		var err error
		scope, err = promptScope()
		if err != nil {
			return err
		}
	}

	targetDir, err := resolveTargetDir(scope, cmd)
	if err != nil {
		return err
	}

	installed, err := scanInstalled(ctx, targetDir)
	if err != nil {
		return err
	}
	if len(installed) == 0 {
		ui.Info("Nothing installed at " + scope + " scope.")
		return nil
	}

	selected, err := promptUninstall(installed)
	if err != nil {
		return err
	}
	if len(selected) == 0 {
		ui.Info("Nothing selected.")
		return nil
	}

	ui.Header("Uninstalling...")

	for _, item := range selected {
		switch item.kind {
		case "skill", "agent":
			if err := os.Remove(item.src); err != nil {
				ui.Error(fmt.Sprintf("%s: %s", item.name, err))
			} else {
				ui.Success("removed: " + item.src)
			}
		case "mcp":
			if out, err := executil.Combined(ctx, "claude", "mcp", "remove", "mozcloud"); err != nil {
				ui.Error("mozcloud-mcp: " + out)
			} else {
				ui.Success("mozcloud-mcp unregistered")
			}
		}
	}

	fmt.Println()
	return nil
}

// resolveTargetDir returns the claude config dir for the given scope.
// If scope is "project" it tries to find the repo root; falls back to CWD/.claude.
func resolveTargetDir(scope string, _ *cobra.Command) (string, error) {
	if scope == "user" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot determine home directory: %w", err)
		}
		return filepath.Join(home, ".claude"), nil
	}

	// project scope — try to find repo root, fall back to CWD
	root, err := executil.Output(context.Background(), "git", "rev-parse", "--show-toplevel")
	if err != nil {
		cwd, _ := os.Getwd()
		return filepath.Join(cwd, ".claude"), nil
	}
	return filepath.Join(strings.TrimSpace(root), ".claude"), nil
}

// scanInstalled returns all MozCloud-installed items found in targetDir.
func scanInstalled(ctx context.Context, targetDir string) ([]installable, error) {
	var items []installable

	for _, sub := range []struct{ kind, dir string }{
		{"skill", "skills"},
		{"agent", "agents"},
	} {
		dir := filepath.Join(targetDir, sub.dir)
		entries, err := os.ReadDir(dir)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			path := filepath.Join(dir, e.Name())
			fi, err := os.Lstat(path)
			if err != nil || fi.Mode()&os.ModeSymlink == 0 {
				continue
			}
			items = append(items, installable{
				kind:  sub.kind,
				name:  e.Name(),
				label: sub.kind + "  / " + strings.TrimSuffix(e.Name(), ".md"),
				src:   path, // for uninstall, src is the symlink path to remove
			})
		}
	}

	// Check if mozcloud-mcp is registered
	out, _ := executil.Output(ctx, "claude", "mcp", "list")
	if strings.Contains(out, "mozcloud") {
		items = append(items, installable{
			kind:  "mcp",
			name:  "mozcloud-mcp",
			label: "mcp    / mozcloud-mcp",
		})
	}

	return items, nil
}

func promptUninstall(items []installable) ([]installable, error) {
	options := make([]huh.Option[string], len(items))
	for i, item := range items {
		options[i] = huh.NewOption(item.label, item.name)
	}

	var selected []string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select items to uninstall").
				Options(options...).
				Value(&selected),
		),
	)
	if err := form.Run(); err != nil {
		return nil, err
	}

	selectedSet := make(map[string]bool, len(selected))
	for _, s := range selected {
		selectedSet[s] = true
	}

	var result []installable
	for _, item := range items {
		if selectedSet[item.name] {
			result = append(result, item)
		}
	}
	return result, nil
}
