---
name: mozcloud-chart-migration
description: Assist with migrating custom tenant Helm charts to the mozcloud shared Helm chart
user-invocable: true
disable-model-invocation: true
---

## General

We are migrating custom Helm charts in the provided directory to use a common shared `mozcloud` chart.
This chart is stored in an OCI repository `oci://us-west1-docker.pkg.dev/moz-fx-platform-artifacts/mozcloud-charts` and should be added as a dependency of the chart being migrated.
We want to use the latest version of the `mozcloud` chart.
   - Check the OCI repository to ensure we are using the latest version of the mozcloud chart before starting any changes.
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

- Download the latest version of the `mozcloud` chart from this OCI repository: `oci://us-west1-docker.pkg.dev/moz-fx-platform-artifacts/mozcloud-charts`
- Use the most recent values.yaml and values.schema.json from the `mozcloud` chart when converting charts to the mozcloud format
- The mozcloud chart follows a specific schema that must be adhered to for successful migration

## Custom Chart Context

Custom charts will generally have templates and values files.
We will have a shared `values.yaml` file and one for each environment: `values-dev.yaml`, `values-stage.yaml`, `values-prod.yaml`.
  - We have several multi-region charts that will contain a regional values file like `values-stage-europe-west1.yaml`.
  - These values files are used along with the matching environment values file. For example `helm template -f values-stage.yaml -f values-stage-europe-west1.yaml`.
  - If regional values files are found, we want to migrate both the `values-stage.yaml` on its own as well as with the regional values file.
We want to convert environments one at a time starting with `values-dev.yaml` if it exists.

## Determine Target Chart to Migrate

- This skill should be run while in the target chart directory.
  - If it is not, ask the user to change to the target chart directory and run again.
- Ensure the current directory has a `Chart.yaml` file and prompt for confirmation that this is the intended chart to migrate.
- List all available values files in the directory (values-*.yaml):
  - Use `ls values-*.yaml` to find all environment values files
  - Display them to the user with a clear list
  - If the user provided a specific values file as an argument, confirm that's the one to use
  - If no argument was provided, ask the user which environment to migrate:
    - Present the list and ask them to select (e.g., "dev", "stage", "preview")
    - Map their response to the corresponding values file (e.g., "dev" → "values-dev.yaml")
  - Default to values-dev.yaml only if it exists and no other selection is made

## Pre-flight Checks

Before starting the migration, verify the following:

1. **Git Status**: Verify no uncommitted changes before starting (unless user confirms they want to proceed with dirty working directory)
2. **Helm Version**: Ensure the `helm` command is available and check that it is not version 4
   ```bash
   helm version --short
   ```
3. **render-diff Tool**: Ensure `render-diff` is available or prompt to install from `https://github.com/mozilla/mozcloud/tree/main/tools/render-diff`
   ```bash
   which render-diff
   ```
4. **OCI Registry Access**: Verify mozcloud chart is accessible from OCI registry:
   ```bash
   helm show chart oci://us-west1-docker.pkg.dev/moz-fx-platform-artifacts/mozcloud-charts/application --version <latest>
   ```

## Setup

- Using `git` create a new branch called `claude-migration-$CHART_NAME-$ENV` where `$CHART_NAME` and `$ENV` match the target chart and environment values file.
  - Example: `claude-migration-cicd-demos-dev`
  - For uniqueness, you may append a date if needed: `claude-migration-$CHART_NAME-$ENV-$(date +%Y%m%d)`
- Create a `.migration` directory inside the chart that we are migrating if it does not already exist.
  - Store any temporary files, migration documentation, and related files in this directory for easy cleanup after the migration is complete.

## Migration Directory Structure

The `.migration` directory will maintain all migration-related documentation and artifacts:

```
.migration/
├── README.md                    # Current status, next steps, decisions (ENTRY POINT)
├── STATUS.md                    # Multi-environment progress tracker
├── SUMMARY.md                   # High-level migration overview (optional)
├── MIGRATION_PLAN_$ENV.md       # Detailed plan for each environment
├── CHANGES_$ENV.md              # Resource name changes per environment
├── DIFF_ANALYSIS_$ENV.md        # Structured diff analysis per environment
└── manifests/                   # Original rendered manifests for comparison
    ├── dev/
    ├── stage/
    └── prod/
```

## Resuming a Migration

**ALWAYS start by reading `.migration/README.md` first** - it is the entry point for understanding current state.

When continuing work on an existing migration:

1. **Read migration context first:**
   - `.migration/README.md` (current status, decisions, next steps)
   - `.migration/MIGRATION_PLAN_$ENV.md` (detailed plan for current environment)
   - `.migration/CHANGES_$ENV.md` (detailed change log)
   - `.migration/SUMMARY.md` (if exists - high level overview)

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
   - Render the helm chart with the current values files using `helm template`
   - Store the rendered manifests in `.migration/manifests/$ENV/` directory
   ```bash
   helm template . -f values.yaml -f values-$ENV.yaml > .migration/manifests/$ENV/original.yaml
   ```

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
   ```bash
   render-diff -f values-$ENV.yaml -su
   ```
   - Must show same number of resources
   - Document any semantic differences in `.migration/CHANGES_$ENV.md`

2. **Enhanced Diff Analysis**:
   - Render the new chart: `helm template . -f values.yaml -f values-$ENV.yaml > .migration/manifests/$ENV/migrated.yaml`
   - Perform detailed comparison and populate `.migration/DIFF_ANALYSIS_$ENV.md`:
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
   - For each non-migrated environment, verify no changes:
   ```bash
   render-diff -f values-stage.yaml
   render-diff -f values-prod.yaml
   ```
   - These should show zero differences

4. **Visual Manifest Comparison**:
   - Compare with original: `diff .migration/manifests/$ENV/original.yaml .migration/manifests/$ENV/migrated.yaml`
   - Review any differences and ensure they are intentional

5. **Regional Values Files** (if applicable):
   - Test combinations: `render-diff -f values-stage.yaml -f values-stage-europe-west1.yaml -su`

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
4. The `.migration` directory can be removed or archived

## Security

- **Never commit any changes to git** - user will review and commit
- **Never run `helm upgrade`, `helm install`, `helm delete`** or any other destructive commands
- Do not write to any directories outside of the chart being migrated
  - Any writes should be to the values files, templates and helpers we're migrating or in the local `.migration` directory
- Always verify changes with render-diff before suggesting the user commit

## Troubleshooting Common Issues

### Render-diff shows missing resources
- Check that mozcloud dependency is enabled in values
- Verify workload names match original deployment names
- Review template gating conditions

### Chart.lock conflicts
- Run `helm dependency update` after modifying Chart.yaml
- Ensure helm-charts repository is accessible

### Resource names changing unexpectedly
- Verify mozcloud workload key matches full original name
- Check for nameOverride or fullnameOverride in values
- Review mozcloud chart's naming conventions in values.schema.json

### Non-migrated environments showing differences
- Ensure template gating is working correctly
- Verify conditional logic in templates
- Check that mozcloud dependency is disabled for those environments

## Summary

1. Always read `.migration/README.md` first - it is the entry point
2. Use `.migration` directory for all migration artifacts
3. Preserve resource names as PRIMARY goal
4. Test thoroughly with `render-diff`
   - Use `helm template` to compare manifests if `render-diff` fails or is unavailable
5. Document all decisions and changes
6. Never commit - user reviews first
7. Update README.md at milestones for clear handoff
