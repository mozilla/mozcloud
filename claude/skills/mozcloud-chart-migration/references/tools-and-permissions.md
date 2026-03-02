# Tools and Permissions

This document lists tools that can be used **without prompting** the user for permission during migration operations.

## Read-Only Commands

These basic shell commands are safe to use freely:

- `pwd` - Print working directory
- `ls`, `ls -la`, `ls -lh` - List files and directories
- `wc -l` - Count lines in files
- `which` - Locate command binaries
- `find` - Find files (though prefer Glob tool for better performance)
- `grep` - Search content (though prefer Grep tool for better performance)

**Note**: `echo` is **NOT** in this list due to security concerns - it could expose sensitive environment variables. Use explicit text output instead, or ask for approval first.

## Git Commands (Read-Only)

Git commands that only read repository state:

- `git status` - Check repository status (staged, unstaged, untracked files)
- `git branch` - List branches (local and remote)
- `git log` - View commit history
- `git diff` - View differences between commits, working tree, etc.
- `git show` - Show commits, tags, and other objects
- `git rev-parse` - Parse git revision information

**Examples:**
```bash
# Check if working directory is clean
git status

# View recent commits
git log --oneline -10

# See what changed in a specific file
git diff HEAD -- values.yaml
```

## Helm Commands (Read-Only)

Helm commands that only read or render locally without modifying state:

- `helm version` - Check Helm version
- `helm show chart <chart>` - Display chart metadata (Chart.yaml)
- `helm show values <chart>` - Display default values from chart
- `helm show all <chart>` - Display all chart information
- `helm template <chart>` - Render templates locally (does not install)
- `helm lint <chart>` - Validate chart structure and syntax
- `helm dependency list` - List chart dependencies
- `helm dependency build` - Build dependencies (downloads, does not install)
- `helm pull <chart>` - Download chart (does not install)

**Examples:**
```bash
# Check Helm version
helm version --short

# View chart metadata
helm show chart oci://us-west1-docker.pkg.dev/.../mozcloud

# Render templates locally
helm template . -f values.yaml -f values-dev.yaml

# Validate chart
helm lint .
```

## Validation Tools

Migration-specific validation tools:

- `render-diff` - Compare rendered manifests (all flags and options)
  - `render-diff -f values-dev.yaml` - Compare original vs migrated
  - `render-diff -f values-dev.yaml -su` - Show semantic differences
  - `render-diff -f values.yaml -f values-regional.yaml` - Multi-file comparison
- `diff` - Compare text files
- `kubectl diff` - Compare manifests (if cluster access available, read-only)

**Examples:**
```bash
# Check semantic differences
render-diff -f values-dev.yaml -su

# Manual file comparison
diff .migration/manifests/dev/original.yaml .migration/manifests/dev/migrated.yaml
```

## File Operations (Read-Only)

Commands for viewing file contents (prefer dedicated tools when available):

- `head` - View first lines of file (prefer Read tool)
- `tail` - View last lines of file (prefer Read tool)
- `cat` - Display file contents (prefer Read tool)
- `stat` - Show file information (size, permissions, timestamps)
- `file` - Determine file type

**Note**: For reading files during migration, prefer using the Read tool over these commands, as it provides better integration with Claude Code.

## Directory Operations

Commands for inspecting directory structure:

- `tree` - Display directory tree (if available)
- `du -sh` - Show directory size
- `find <dir> -type f -name "*.yaml"` - Find specific files (prefer Glob tool)

## Strictly Prohibited Commands (NEVER Use)

**CRITICAL**: The following commands should **NEVER** be used, even if the user requests or approves them. The skill must refuse and explain why.

### Environment Variable Access (Security Risk)
- `env` - Lists all environment variables (exposes secrets)
- `printenv` - Prints environment variables (exposes secrets)
- `echo $VAR` - Prints variable values (exposes secrets)
- `export` - Shows/sets environment variables (exposes secrets)
- `set` - Shows all variables including environment (exposes secrets)

**Why Prohibited**: Environment variables often contain sensitive credentials (API keys, passwords, database credentials, tokens). Exposing these could compromise security.

**What to do instead**: If the user needs to check a value, ask them to verify it themselves in their terminal.

### Helm Deployment Commands (Out of Scope)
- `helm install <release> <chart>` - Installs charts to cluster
- `helm upgrade <release> <chart>` - Upgrades releases
- `helm delete` / `helm uninstall` - Deletes releases
- `helm rollback` - Rolls back releases
- Any `helm` command with `--install` or `--atomic` flags

**Why Prohibited**: This skill prepares migration configurations. **Deployments happen via ArgoCD**, not directly via Helm. Running these commands would:
- Bypass the ArgoCD deployment workflow
- Skip team review and approval processes
- Potentially deploy untested configurations
- Create state conflicts with ArgoCD

**What to do instead**: Prepare the configuration files and let the user commit them. ArgoCD will handle deployment after merge to main.

### Kubernetes Commands (Out of Scope)
- `kubectl apply` - Apply configuration
- `kubectl create` - Create resources
- `kubectl delete` - Delete resources
- `kubectl edit` - Edit resources
- `kubectl patch` - Patch resources
- `kubectl replace` - Replace resources
- `kubectl scale` - Scale resources
- `kubectl set` - Set resource properties
- `kubectl exec` - Execute commands in pods
- `kubectl port-forward` - Forward ports
- **Any `kubectl` command that modifies cluster state**

**Why Prohibited**: Same reasoning as Helm commands - this skill prepares configurations, it does not deploy them. All deployments go through ArgoCD.

**What to do instead**: Focus on generating correct values files and templates. Let ArgoCD deploy after merge.

### Process Information (Security Risk)
- `ps aux` - Shows running processes (may reveal secrets in arguments)
- `top`, `htop` - Process viewers (may show sensitive information)

**Why Prohibited**: Process lists may reveal secrets passed as command-line arguments.

## How to Handle User Requests for Prohibited Commands

If a user asks you to run a prohibited command:

1. **Politely refuse**: Explain that the command is outside the scope of this skill
2. **Explain why**: Reference the specific reason (security risk, deployment workflow, etc.)
3. **Suggest alternatives**:
   - For env/secrets: "Please check this value in your terminal"
   - For deployments: "This migration prepares the configuration. After you commit and merge, ArgoCD will handle deployment"
   - For kubectl: "This skill focuses on preparing Helm charts, not deploying them"

**Example Response:**
```
I cannot run `helm install` as this skill is designed to prepare migration
configurations, not deploy them. Your deployment workflow uses ArgoCD, which
will automatically deploy once you commit these changes and merge to main.

This ensures proper review and follows your established deployment process.
```

## Commands That Require User Approval

The following commands modify state and require explicit user approval before use:

### Git Commands (Modifying)
- `git add` - Stage changes
- `git commit` - Create commits
- `git push` - Push to remote
- `git pull` - Pull from remote
- `git merge` - Merge branches
- `git rebase` - Rebase commits
- `git reset` - Reset state
- `git checkout` - Switch branches or restore files
- `git rm` - Remove files
- `git clean` - Clean untracked files

**Note**: While these require approval, they are within scope for migration workflow (e.g., creating migration branches, staging files). Use judiciously and only when needed.

### File System Commands (Modifying)
- `rm` - Remove files (dangerous, use with extreme caution)
- `mv` - Move files (prefer Edit/Write tools)
- `cp` - Copy files (only for migration artifacts)
- `mkdir` - Create directories (typically only for `.migration/` structure)

**CRITICAL RESTRICTION**: All file writes (whether via bash commands or Write/Edit tools) must be within the current chart directory (`$CHART_DIR`).

**Prohibited**:
- Writing to parent directories (`../`)
- Writing to absolute paths outside chart directory (`/tmp/`, `/home/`, etc.)
- Writing to other tenant directories
- Writing to system directories

**Allowed**:
- Chart files: `$CHART_DIR/values.yaml`, `$CHART_DIR/Chart.yaml`, `$CHART_DIR/templates/*`
- Migration directory: `$CHART_DIR/.migration/*`

**If user requests writes outside chart directory**: Refuse and explain that the skill is scoped to the current tenant chart only.

**Note**: Prefer using Claude Code's Write, Edit, and other file tools instead of shell commands when possible.

### Package Management (Rarely Needed)
- `brew install` / `apt install` / `yum install` - Install packages
- Any package manager commands

**Note**: Only if user is missing required tools (render-diff, helm, etc.) and requests installation assistance.

## Best Practices

1. **Use dedicated tools first**: Prefer Glob over `find`, Grep over `grep`, Read over `cat`
2. **Read-only by default**: Always use read-only commands unless modification is explicitly required
3. **User approval for changes**: Any command that modifies files, git state, or remote systems requires approval
4. **Verify before acting**: Use read-only commands to verify state before suggesting modifications
5. **Document tool usage**: When using specialized tools like `render-diff`, explain what they're checking

## Adding New Tools

If additional tools are needed during migration:

1. **Verify it's read-only**: Ensure the tool doesn't modify state
2. **Document its purpose**: Add to this reference with examples
3. **Update SKILL.md**: Add a brief mention in the main skill file
4. **Get user approval first**: If unsure, ask the user before using a new tool
