---
name: mozcloud-chart-migration
description: Assist with migrating custom tenant Helm charts to the mozcloud shared Helm chart
user-invocable: true
disable-model-invocation: true
---

## Requirements

This skill requires the **`mozcloud` MCP server** to be registered with Claude Code. It provides all Helm and migration tooling used throughout this workflow.

If MCP tools (`migration_preflight_check`, `helm_template`, etc.) are unavailable, ask the user to run from the repository root:

```bash
./claude/install.sh --mcp-project
```

## General

**CRITICAL: This skill NEVER commits changes.** The skill prepares migration files, validates changes, and creates documentation. The user is always responsible for reviewing changes and running git commands (add, commit, push, PR creation).

We are migrating custom Helm charts in the provided directory to use a common shared `mozcloud` chart.
This chart is stored in an OCI repository `oci://us-west1-docker.pkg.dev/moz-fx-platform-artifacts/mozcloud-charts` and should be added as a dependency of the chart being migrated.
We want to use the latest version of the `mozcloud` chart.
   - Call `helm_chart_latest_version` to confirm the latest version before starting any changes.
   - If the nginx image provided in the custom chart is `us-west1-docker.pkg.dev/moz-fx-platform-artifacts/platform-shared-images/nginx-unprivileged:1.22` we can ignore that version and use the latest from the `mozcloud` chart.
We do not want any loss of rendered resources.
  - If there are 10 rendered manifests with the custom chart, there should be 10 or more after our migration.
Only work on a single environment values file at one time.
Our main goal is to end up with a chart that has no templates if possible. The values files should contain all the configuration required to successfully render manifests with the `mozcloud` chart.

## Deployment Context

**ArgoCD-Based Workflow:**
- All charts are deployed via ArgoCD
- **Rollback strategy**: If migration has issues, simply delete the branch or push fixes - no complex rollback procedures needed
- Once merged to main, changes deploy to the target environment
- Engineers will manually verify deployments - focus on generating correct values and templates with minimal resource changes

**Migration Branch Strategy:**
- Each migration branch isolates changes from other environments
- Some tenants may have preview/testing environments available for validation before merging
- Non-merged changes don't affect production environments
- This allows for iterative refinement and testing

## Mozcloud Chart Context

- Download the latest version of the `mozcloud` chart from this OCI repository: `oci://us-west1-docker.pkg.dev/moz-fx-platform-artifacts/mozcloud-charts/mozcloud`
- Use the most recent values.yaml and values.schema.json from the `mozcloud` chart when converting charts to the mozcloud format
- The mozcloud chart follows a specific schema that must be adhered to for successful migration

**See [references/mozcloud-chart-reference.md](references/mozcloud-chart-reference.md) for complete chart details, schema information, and migration patterns.**

## Custom Chart Context

Custom charts will generally have templates and values files.
We will have a shared `values.yaml` file and one for each environment: `values-dev.yaml`, `values-stage.yaml`, `values-prod.yaml`.
  - We have several multi-region charts that will contain a regional values file like `values-stage-europe-west1.yaml`.
  - These values files are used along with the matching environment values file. For example `helm template -f values-stage.yaml -f values-stage-europe-west1.yaml`.
  - If regional values files are found, we want to migrate both the `values-stage.yaml` on its own as well as with the regional values file.
We want to convert environments one at a time starting with `values-dev.yaml` if it exists.

**Important Context for Preview Environments:**

Preview environments have different requirements than dev/stage. See [references/preview-environment-guide.md](references/preview-environment-guide.md) for complete details on:
- Resource naming differences (preserve names in dev/stage, prefix with PR# in preview)
- Preview-specific configuration patterns
- Critical validation points

**Important Context for Nginx Configuration:**

Mozcloud provides default nginx configuration. See [references/configuration-patterns.md](references/configuration-patterns.md) for guidance on:
- When to use mozcloud default nginx (recommended)
- How to handle custom nginx configurations
- Decision guide for static vs templated configs

## Determine Target Chart to Migrate

- This skill should be run while in the target chart directory.
  - If it is not, ask the user to change to the target chart directory and run again.
- Ensure the current directory has a `Chart.yaml` file and prompt for confirmation that this is the intended chart to migrate.

### Regular Migration (Environment-Specific)

- List all available values files using `values_list_environments` with `chart_path` set to the current directory.
  - Display the returned list to the user with a clear list
  - If the user provided a specific values file as an argument, confirm that's the one to use
  - If no argument was provided, ask the user which environment to migrate:
    - Present the list and ask them to select (e.g., "dev", "stage", "preview")
    - Map their response to the corresponding values file (e.g., "dev" → "values-dev.yaml")
  - Default to values-dev.yaml only if it exists and no other selection is made

### Cleanup Mode

If the user invokes with `cleanup` argument:
```bash
/mozcloud-chart-migration cleanup
```

**Prerequisites Check**:
- Verify all environment migrations are merged to main (check git branch history)
- Confirm Chart.yaml has mozcloud dependency with `condition: mozcloud.enabled`
- Ask user to confirm environments are deployed and stable

**Actions** (see [references/cleanup-phase.md](references/cleanup-phase.md) for detailed guide):
1. Consolidate duplicate values from `values-{env}.yaml` into `values.yaml`
2. Remove unused custom templates (those fully replaced by mozcloud)
3. Simplify Chart.yaml (remove `condition: mozcloud.enabled`)
4. Remove `mozcloud.enabled: true` from all values files
5. Call `helm_dependency_update` with `chart_path=$CHART_DIR`
6. Archive `.migration/` directory (move to `.migration-archive`)
7. Validate all changes with `render_diff` for each environment (must show `has_diff: false`)
8. Create cleanup branch for review

**Important**:
- Cleanup is optional and low-urgency
- Functionality remains identical (validated by render-diff)
- Makes future maintenance easier by reducing duplication

## Pre-flight Checks

Before starting the migration, call `migration_preflight_check` with `chart_path` set to the current directory.

This single tool checks all prerequisites:
1. **Git Status** — working directory must be clean
2. **Helm Version** — `helm` must be available and not version 4
3. **render-diff Tool** — `render-diff` binary must be on `$PATH`
4. **OCI Registry Auth** — credential store must have an entry for `us-west1-docker.pkg.dev`

If `all_passed` is `false`, surface the `blockers` list to the user and stop. Common remediations:
- Missing `render-diff`: install from `tools/render-diff` in this repository
- OCI auth failure: run `gcloud auth configure-docker us-west1-docker.pkg.dev`
- Dirty git state: commit or stash changes (or get explicit user confirmation to proceed)

## Working Directory Management

**CRITICAL**: Always use absolute paths with `$CHART_DIR` variable when running bash/git commands to prevent files from being created in wrong locations.

Quick reference:
```bash
# Capture chart root at start
CHART_DIR=$(pwd)

# Always use absolute paths for bash/git operations
mkdir -p $CHART_DIR/.migration/manifests/dev
```

MCP tools (`helm_template`, `render_diff`, `helm_pull`, etc.) accept a `chart_path` parameter — pass the absolute `$CHART_DIR` value directly. They do not change the shell working directory.

**See [references/working-directory-management.md](references/working-directory-management.md) for complete guidance on directory management, safety checks, and why this matters.**

## Setup

**Step 1: Capture chart root directory and read chart metadata:**

```bash
CHART_DIR=$(pwd)
echo "Chart root: $CHART_DIR"
```

Then call `chart_read_metadata` with `chart_path=$CHART_DIR` to parse `Chart.yaml`. If the tool returns an error (file not found), the directory is not a chart — ask the user to change to the correct chart directory and run again.

Use the returned metadata to confirm the chart name with the user before proceeding.

**Step 2: Create migration branch:**

Using `git` create a new branch called `claude-migration-$CHART_NAME-$ENV` where `$CHART_NAME` and `$ENV` match the target chart and environment values file.
- Example: `claude-migration-cicd-demos-dev`
- For uniqueness, you may append a date if needed: `claude-migration-$CHART_NAME-$ENV-$(date +%Y%m%d)`

**Step 3: Create migration directory structure:**

```bash
# Use absolute path with $CHART_DIR variable
mkdir -p $CHART_DIR/.migration/manifests/{dev,stage,prod}

# Verify directory was created in correct location
ls -la $CHART_DIR/.migration/
```

**Step 4: Download mozcloud reference chart:**

First call `helm_chart_latest_version` with `repository=us-west1-docker.pkg.dev/moz-fx-platform-artifacts/mozcloud-charts` and `chart_name=mozcloud` to get the latest version string.

Then call `helm_pull` with:
- `repository`: `us-west1-docker.pkg.dev/moz-fx-platform-artifacts/mozcloud-charts`
- `chart_name`: `mozcloud`
- `version`: the latest version returned above
- `destination`: `$CHART_DIR/.migration/`
- `untar`: `true`

To inspect the chart's default values and schema, call:
- `helm_show_values` with `repository=us-west1-docker.pkg.dev/moz-fx-platform-artifacts/mozcloud-charts`, `chart_name=mozcloud`, and `version` set to the latest version
- `helm_show_schema` with `repository=us-west1-docker.pkg.dev/moz-fx-platform-artifacts/mozcloud-charts`, `chart_name=mozcloud`, and `version` set to the latest version

The chart will be extracted to `$CHART_DIR/.migration/mozcloud/`.

**Note**: The `.migration` directory should be git-ignored (add to `.gitignore`) to avoid repository clutter. It stores migration documentation and artifacts locally, which can inform future tenant migrations on the same machine.

## Migration Directory Structure

The `.migration` directory maintains all migration-related documentation and artifacts:

```
.migration/
├── README.md                    # ENTRY POINT - current status, next steps
├── STATUS.md                    # Multi-environment progress tracker
├── MIGRATION_PLAN_$ENV.md       # Detailed plan per environment
├── CHANGES_$ENV.md              # Change log per environment
├── DIFF_ANALYSIS_$ENV.md        # Diff analysis per environment
├── mozcloud/                    # Downloaded mozcloud chart reference
└── manifests/$ENV/              # Original & migrated manifests
```

**See [references/migration-directory-structure.md](references/migration-directory-structure.md) for complete details, file purposes, and examples.**

## Resuming a Migration

**ALWAYS start by reading migration status first** — it is the entry point for understanding current state.

When continuing work on an existing migration:

1. **Read migration context first:**
   - Call `migration_read_status` with `chart_path=$CHART_DIR` — returns `.migration/README.md` and `.migration/STATUS.md` in a single call
   - Then read `.migration/MIGRATION_PLAN_$ENV.md` and `.migration/CHANGES_$ENV.md` for the current environment

2. **Verify current state:**
   - Check which environment we're migrating
   - Confirm which workloads are already done
   - Review any known issues or blockers
   - Check if team decisions are pending

3. **Continue where left off:**
   - Follow "Next Steps" from README.md
   - Don't re-do completed work
   - Build on previous decisions and patterns from other environment migrations
   - Update docs as you progress

4. **Respect previous decisions:**
   - Don't reverse decisions without explicit user request
   - If you disagree with previous approach, note it but continue consistently
   - Ask user before changing strategy

5. **Document as you go:**
   - Update README.md at major milestones
   - Keep CHANGES_$ENV.md current with any resource name changes
   - Note any blockers or pending decisions

## Learning from Other Tenant Migrations

**Important**: `.migration/` directories are git-ignored to avoid repository clutter and merge conflicts. This means migration context stays local to each developer's workstation but can inform other migrations on the same machine.

**Before starting a new migration**, search for other tenant migrations to learn from existing patterns and decisions:

1. **Search for other `.migration/` directories:**
   ```bash
   # From current chart directory, search parent directories for other tenant migrations
   find ../.. -type d -name ".migration" 2>/dev/null
   ```

2. **Read migration documentation from similar tenants:**
   - Look for `.migration/README.md` files in other charts
   - Check `.migration/CHANGES_*.md` for resource migration patterns
   - Review `.migration/DIFF_ANALYSIS_*.md` to understand common changes
   - Note any recurring patterns or solutions

3. **Identify applicable patterns:**
   - **Naming conventions**: How did other tenants handle resource names?
   - **Workload configuration**: Common patterns for Deployment → mozcloud workload conversion
   - **ExternalSecret mapping**: How were secret configurations migrated?
   - **ConfigMap handling**: Static files vs mozcloud config patterns
   - **Ingress/HTTPRoute**: Gateway and routing configurations
   - **Preview environments**: Special handling for preview vs regular environments

4. **Apply consistent patterns across tenants:**
   - If multiple tenants used the same approach, prefer that pattern
   - Document when you deviate from established patterns (and why)
   - Note new patterns you discover for future migrations

5. **Example search and analysis:**
   ```bash
   # Find all tenant migration READMEs
   find ../.. -path "*/.migration/README.md" -type f 2>/dev/null

   # Count how many tenants have completed migrations
   find ../.. -path "*/.migration/STATUS.md" -type f 2>/dev/null | wc -l
   ```

**When to skip this step:**
- This is the first migration on this machine (no other `.migration/` directories exist)
- User explicitly requests to start fresh without referencing other migrations
- The current tenant has very different requirements from others

**Benefits of cross-tenant learning:**
- Consistency across tenant migrations
- Faster migrations by reusing proven patterns
- Avoiding mistakes others already solved
- Building institutional knowledge locally

## Starting a New Migration Plan

1. **Initialize or update multi-environment status tracker:**
   - Create or update `.migration/STATUS.md` with format:
   ```markdown
   # Migration Progress

   | Environment | Status | Branch | Notes |
   |-------------|--------|--------|-------|
   | dev | Completed | merged to main | Merged on YYYY-MM-DD |
   | stage | In Progress | claude-migration-xxx-stage | Current work |
   | prod | Not Started | - | Blocked: waiting for stage |

   ## Current Focus
   - **Environment**: stage
   - **Branch**: claude-migration-xxx-stage
   - **Started**: YYYY-MM-DD
   ```

2. **Identify values files:**
   - Read the current `values.yaml` and target environment values file (e.g., `values-dev.yaml`)
   - If there is no `values-dev.yaml`, try `values-stage.yaml`
   - If neither exists, check which values files are available and prompt for selection

2. **Understand current state:**
   - Read the custom helm chart's templates
   - Note any custom resources, helpers, or special configurations
   - Identify dependencies in `Chart.yaml`

3. **Capture current manifests:**
   - Render the helm chart using `helm_template` with:
     - `chart_path`: `$CHART_DIR`
     - `values_files`: `["$CHART_DIR/values.yaml", "$CHART_DIR/values-$ENV.yaml"]`
   - Write the returned manifests to `$CHART_DIR/.migration/manifests/$ENV/original.yaml`
   - Note the `resource_count` from the tool response — this is the baseline to preserve

4. **Create initial diff analysis template:**
   - Create `.migration/DIFF_ANALYSIS_$ENV.md` with structure:
   ```markdown
   # Diff Analysis: $ENV Environment

   ## Resource Impact Summary
   - **Added**: 0 resources
   - **Modified**: TBD
   - **Deleted**: 0 resources
   - **Unchanged**: TBD

   ## Changes That May Trigger Pod Restarts
   - TBD (will be populated after migration)

   ## Stable Resources (No Changes)
   - TBD (will be populated after migration)

   ## Critical Changes Requiring Review
   - TBD (will be populated after migration)
   ```

5. **Create migration plan:**
   - Save the migration plan in `.migration/MIGRATION_PLAN_$ENV.md`
   - Include:
     - Current state analysis
     - Proposed changes to values files
     - Chart.yaml dependency updates
     - Template modifications or removals
     - Resource name mapping (if any names must change)
     - Testing strategy
   - **Prompt for user review before continuing**

6. **Execute migration:**

   **Resource Name Preservation (Mandatory)**

   Before writing any configuration, follow this procedure:

   1. List all current resource names from original manifests
   2. For each resource, verify how mozcloud will name it
   3. If any name differs from the original:
      - Stop immediately - do not implement the configuration
      - Ask the user - present the name change with:
        - Original name
        - Proposed mozcloud name
        - Technical reason (if name cannot be preserved)
        - Alternative options (if any exist)
      - Wait for approval - do not proceed until user explicitly approves

   This applies to all resources including:
   - Deployments/Rollouts
   - Services
   - ConfigMaps
   - Secrets/ExternalSecrets (both the ExternalSecret resource and the target Secret name)
   - Ingress/HTTPRoutes
   - Certificates
   - Any GCP-specific resources

   Common issues to watch for:
   - Mozcloud may auto-generate resources (like ExternalSecrets) - verify their names match originals
   - ConfigMap names in mozcloud may have different conventions - preserve original names
   - ExternalSecret resource name vs target Secret name - both must be considered

   ---

   After receiving approval for any name changes, proceed with implementation:
   - Convert the existing `values.yaml` and environment values file to the format required by the mozcloud schema
   - **Leave the existing values file configuration in place if possible**
     - Make a clear distinction between what is new and old
     - We want it to be easy to clean up after all environments have been migrated
     - We do not want other environments to break while we're migrating one
   - Update the `Chart.yaml` to include the mozcloud chart as a dependency
     - Ensure this dependency can be toggled in the values file
     - Enable only for the environment we're migrating or ones that have already been migrated
   - Ensure none of the custom chart templates are included in the environment we're migrating, unless there is a resource that cannot be generated by the shared mozcloud chart
     - We do not want to remove the templates entirely
     - Wrap any templates in a flag that can be disabled for the environment we're migrating
     - Prompt for input if a resource cannot be created by the shared chart
     - **CRITICAL**: Preserve original resource names exactly if possible
       - The mozcloud workload name becomes the Deployment name
       - Use the FULL original deployment name as the workload key (e.g., `gha-fxa-profile-worker`, not `profile-worker`)
       - Example:
         ```
         Original: Deployment "gha-fxa-profile-worker"
         Mozcloud: workloads.gha-fxa-profile-worker (NOT workloads.profile-worker)
         ```
       - **Minimizing resource name changes is a PRIMARY goal, not optional**

7. **Document changes:**
   - Show a summary of all changes made
   - Store detailed changes in `.migration/CHANGES_$ENV.md`
   - Include:
     - Resource name changes (if any)
     - New mozcloud configuration patterns used
     - Any workarounds or special cases
     - Template modifications

8. **Final environment migration:**
   - If we are migrating the last environment, ensure all duplicate values file entries live in the default `values.yaml`
   - Some entries may be moved temporarily to environment values files as we migrate non-production environments
   - Prompt for confirmation before making this change

## Testing (Required Steps)

After completing the migration, perform these validation steps in order:

1. **Semantic Diff Check**:
   Call `render_diff` with:
   - `chart_path`: `$CHART_DIR`
   - `values_files`: `["$CHART_DIR/values.yaml", "$CHART_DIR/values-$ENV.yaml"]`
   - `semantic`: `true`
   - `update_dependencies`: `true`

   The response includes `has_diff` (bool), `diff` text, and a `summary`. If `has_diff` is true, review the diff and document meaningful differences in `.migration/CHANGES_$ENV.md`.

2. **Enhanced Diff Analysis**:
   Call `render_manifests` with the same `chart_path` and `values_files` to get the migrated manifests. Write the returned manifests to `$CHART_DIR/.migration/manifests/$ENV/migrated.yaml`, then compare with `original.yaml` to populate `.migration/DIFF_ANALYSIS_$ENV.md`:
   ```markdown
   ## Resource Impact Summary
   - **Added**: X resources (list names)
   - **Modified**: Y resources (list names)
   - **Deleted**: Z resources (list names)
   - **Unchanged**: N resources

   ## Changes That May Trigger Pod Restarts
   - Deployment "foo": environment variable added
   - Deployment "bar": image tag changed

   ## Stable Resources (No Changes)
   - ConfigMap "app-config"
   - Secret "app-secret"
   - Service "app-service"

   ## Critical Changes Requiring Review
   - HTTPRoute gateway changed from X to Y
   - Resource limits increased by 50%
   ```

3. **Non-Migrated Environment Verification**:
   For each non-migrated environment, call `render_diff` with that environment's values files. The `has_diff` field must be `false` — these environments must show zero differences.

4. **Values Schema Validation**:
   Call `schema_validate_values` with:
   - `values_file`: `$CHART_DIR/values-$ENV.yaml`
   - `schema_file`: `$CHART_DIR/.migration/mozcloud/values.schema.json`

   Fix any validation errors before proceeding.

5. **Regional Values Files** (if applicable):
   Call `render_diff` with `values_files: ["$CHART_DIR/values-stage.yaml", "$CHART_DIR/values-stage-europe-west1.yaml"]` and `semantic: true`.

## Migration Completion Checklist

Before considering a migration complete, verify:

- [ ] All required resources are rendered (count matches or exceeds original)
- [ ] Resource names are preserved (or changes are documented and approved)
- [ ] Non-migrated environments show no changes
- [ ] render-diff shows semantic equivalence
- [ ] Chart.yaml includes mozcloud dependency with proper version
- [ ] Values files follow mozcloud schema
- [ ] Templates are either removed or properly gated
- [ ] `.migration/CHANGES_$ENV.md` is complete
- [ ] `.migration/DIFF_ANALYSIS_$ENV.md` is complete
- [ ] `.migration/STATUS.md` is updated
- [ ] `.migration/README.md` is updated with current status
- [ ] User has reviewed and approved the changes

## Post-Migration Cleanup

After ALL environments have been successfully migrated:

1. Remove custom templates that are no longer needed
2. Consolidate duplicate configuration into `values.yaml`
3. Remove environment-specific flags for mozcloud dependency
4. The `.migration` directory should be kept locally (it's git-ignored) to inform future tenant migrations

## Security

**Strictly Prohibited (Never Use, Even With Approval):**
- **Never access environment variables** (`env`, `printenv`, `echo $VAR`, `export`) - risk of exposing secrets
- **Never run Helm deployment commands** (`helm install`, `helm upgrade`, `helm delete`, `helm rollback`) - deployments happen via ArgoCD
- **Never run kubectl commands** - this skill prepares configurations, ArgoCD handles deployment
- If user requests these, politely refuse and explain why

**General Security Practices:**
- **Never commit changes** - user will review and commit
- **Scope file writes strictly** to current chart directory (`$CHART_DIR`) only:
  - Allowed: `$CHART_DIR/values.yaml`, `$CHART_DIR/.migration/*`, `$CHART_DIR/templates/*`
  - Prohibited: Parent directories (`../`), other tenant directories, absolute paths outside chart, system directories
  - If user requests writes outside current directory, refuse and explain scope restriction
- **Verify changes** with render-diff before suggesting the user commit
- **Preserve resource names** to avoid unintended service disruptions

## Troubleshooting Common Issues

Common issues and solutions:
- **Render-diff shows missing resources**: Check mozcloud dependency enabled, verify workload names match original
- **Chart.lock conflicts**: Run `helm dependency update` after modifying Chart.yaml
- **Resource names changing**: Verify workload key matches full original deployment name
- **Non-migrated environments showing differences**: Check template gating and mozcloud.enabled flag

**See [references/troubleshooting.md](references/troubleshooting.md) for complete troubleshooting guide with detailed solutions.**

## Summary

### Critical Principles
1. **Working Directory Management**: Always use `$CHART_DIR` variable with absolute paths (see [references/working-directory-management.md](references/working-directory-management.md))
2. **Entry Point**: Always read `.migration/README.md` first when resuming work
3. **Resource Names**: Preserve original names as PRIMARY goal - require approval for any changes
4. **Environment Isolation**: Migrate one environment at a time, verify others show no changes
5. **Validation**: Test thoroughly with `render-diff` before suggesting commit
6. **Documentation**: Update migration docs at each milestone for clear handoff
7. **Safety**: Never commit - user reviews first

### Reference Documentation
- [Mozcloud Chart Reference](references/mozcloud-chart-reference.md) - Chart details, schema, patterns
- [Working Directory Management](references/working-directory-management.md) - Absolute paths, safety checks
- [Migration Directory Structure](references/migration-directory-structure.md) - File purposes, examples
- [Troubleshooting](references/troubleshooting.md) - Common issues and solutions
