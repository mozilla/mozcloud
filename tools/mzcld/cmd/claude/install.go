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

const mcpModule = "github.com/mozilla/mozcloud/tools/mozcloud-mcp"

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install Claude skills, agents, and MCP servers from the MozCloud repo",
	Long: `install discovers available skills, agents, and MCP servers in the claude/
directory of the MozCloud repository and lets you choose which to install.

Skills and agents are symlinked into your Claude configuration directory.
The mozcloud-mcp server is installed via go install and registered with Claude.

Run this command from anywhere inside the MozCloud repository.`,
	RunE: runInstall,
}

var (
	scopeFlag  string
	updateFlag bool
	allFlag    bool
)

func init() {
	installCmd.Flags().StringVar(&scopeFlag, "scope", "", "Scope: user or project (skips prompt)")
	installCmd.Flags().BoolVar(&updateFlag, "update", false, "Update the mozcloud-mcp binary to the latest published version")
	installCmd.Flags().BoolVar(&allFlag, "all", false, "Install all available items without prompting")
	installCmd.Flags().SortFlags = false
}

// installable represents a skill, agent, or MCP server that can be installed.
type installable struct {
	kind  string // "skill", "agent", "mcp"
	name  string
	label string // display label for the TUI
	src   string // source path (for symlinking); empty for mcp
}

func runInstall(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	claudeDir, err := findClaudeDir()
	if err != nil {
		return err
	}

	items, err := discover(claudeDir)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		ui.Warn("No skills, agents, or MCP servers found in " + claudeDir)
		return nil
	}

	// Determine scope
	scope := scopeFlag
	if scope == "" && !allFlag {
		scope, err = promptScope()
		if err != nil {
			return err
		}
	}
	if scope == "" {
		scope = "user"
	}

	// Determine which items to install
	selected := items
	if !allFlag {
		selected, err = promptItems(items)
		if err != nil {
			return err
		}
	}
	if len(selected) == 0 {
		ui.Info("Nothing selected.")
		return nil
	}

	// Determine target claude dir
	targetDir, err := targetClaudeDir(scope, claudeDir)
	if err != nil {
		return err
	}

	ui.Header("Installing...")

	for _, item := range selected {
		switch item.kind {
		case "skill", "agent":
			if err := installSymlink(item, targetDir); err != nil {
				ui.Error(fmt.Sprintf("%s: %s", item.name, err))
			}
		case "mcp":
			if err := installMCP(ctx, scope, updateFlag); err != nil {
				ui.Error("mozcloud-mcp: " + err.Error())
			}
		}
	}

	fmt.Println()
	return nil
}

// findClaudeDir locates the claude/ directory in the MozCloud repo by walking
// up from the CWD using git to find the repo root.
func findClaudeDir() (string, error) {
	out, err := executil.Output(context.Background(), "git", "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("not inside a git repository: run this command from within the mozcloud repository")
	}
	root := strings.TrimSpace(out)
	claudeDir := filepath.Join(root, "claude")
	if _, err := os.Stat(claudeDir); os.IsNotExist(err) {
		return "", fmt.Errorf("claude/ directory not found at %s: are you in the mozcloud repository?", claudeDir)
	}
	return claudeDir, nil
}

// discover returns all installable items found in claudeDir.
func discover(claudeDir string) ([]installable, error) {
	var items []installable

	// Skills: each subdirectory under claude/skills/
	skillsDir := filepath.Join(claudeDir, "skills")
	if entries, err := os.ReadDir(skillsDir); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				items = append(items, installable{
					kind:  "skill",
					name:  e.Name(),
					label: "skill  / " + e.Name(),
					src:   filepath.Join(skillsDir, e.Name()),
				})
			}
		}
	}

	// Agents: each .md file under claude/agents/
	agentsDir := filepath.Join(claudeDir, "agents")
	if entries, err := os.ReadDir(agentsDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
				items = append(items, installable{
					kind:  "agent",
					name:  e.Name(),
					label: "agent  / " + strings.TrimSuffix(e.Name(), ".md"),
					src:   filepath.Join(agentsDir, e.Name()),
				})
			}
		}
	}

	// MCP server: always offer mozcloud-mcp
	items = append(items, installable{
		kind:  "mcp",
		name:  "mozcloud-mcp",
		label: "mcp    / mozcloud-mcp",
	})

	return items, nil
}

func promptScope() (string, error) {
	var scope string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Scope").
				Description("Where should skills and agents be installed?").
				Options(
					huh.NewOption("User  — ~/.claude/  (available in all projects)", "user"),
					huh.NewOption("Project  — .claude/  (this repository only)", "project"),
				).
				Value(&scope),
		),
	)
	if err := form.Run(); err != nil {
		return "", err
	}
	return scope, nil
}

func promptItems(items []installable) ([]installable, error) {
	options := make([]huh.Option[string], len(items))
	for i, item := range items {
		options[i] = huh.NewOption(item.label, item.name)
	}

	var selected []string
	// Default all selected
	defaults := make([]string, len(items))
	for i, item := range items {
		defaults[i] = item.name
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select items to install").
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

func targetClaudeDir(scope, claudeDir string) (string, error) {
	if scope == "project" {
		// Project root is one level up from claude/
		repoRoot := filepath.Dir(claudeDir)
		return filepath.Join(repoRoot, ".claude"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".claude"), nil
}

func installSymlink(item installable, targetDir string) error {
	var subdir string
	switch item.kind {
	case "skill":
		subdir = "skills"
	case "agent":
		subdir = "agents"
	}

	dir := filepath.Join(targetDir, subdir)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("cannot create %s: %w", dir, err)
	}

	dst := filepath.Join(dir, item.name)
	switch {
	case isSymlink(dst):
		ui.Dim("already linked: " + dst)
	case exists(dst):
		ui.Warn("skipped (file exists): " + dst)
	default:
		if err := os.Symlink(item.src, dst); err != nil {
			return err
		}
		ui.Success("linked: " + dst)
	}
	return nil
}

func installMCP(ctx context.Context, scope string, update bool) error {
	alreadyRegistered := func() bool {
		out, _ := executil.Output(ctx, "claude", "mcp", "list")
		return strings.Contains(out, "mozcloud")
	}

	if update {
		ui.Info("Updating mozcloud-mcp...")
		if out, err := executil.Combined(ctx, "go", "install", mcpModule+"@latest"); err != nil {
			return fmt.Errorf("go install failed: %s", out)
		}
		ui.Success("mozcloud-mcp updated")
		return nil
	}

	if alreadyRegistered() {
		ui.Dim("mozcloud-mcp already registered (use --update to upgrade)")
		return nil
	}

	ui.Info("Installing mozcloud-mcp...")
	if out, err := executil.Combined(ctx, "go", "install", mcpModule+"@latest"); err != nil {
		return fmt.Errorf("go install failed: %s", out)
	}
	if out, err := executil.Combined(ctx, "claude", "mcp", "add",
		"--scope", scope, "mozcloud", "mozcloud-mcp", "--", "--transport", "stdio"); err != nil {
		return fmt.Errorf("claude mcp add failed: %s", out)
	}
	ui.Success("mozcloud-mcp installed and registered (" + scope + " scope)")
	return nil
}

func isSymlink(path string) bool {
	fi, err := os.Lstat(path)
	return err == nil && fi.Mode()&os.ModeSymlink != 0
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
