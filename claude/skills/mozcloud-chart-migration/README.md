# Mozcloud Chart Migration Skill

A Claude Code skill that assists with migrating custom tenant Helm charts to the shared `mozcloud` Helm chart.

## Overview

This skill automates and guides the process of migrating custom Helm charts to use Mozilla's shared `mozcloud` chart, which is stored in the OCI repository `oci://us-west1-docker.pkg.dev/moz-fx-platform-artifacts/mozcloud-charts/mozcloud`.

The migration follows an **ArgoCD-based deployment workflow**:
- All charts are deployed via ArgoCD
- Changes are isolated per environment using migration branches
- Rollback is simple: delete the branch or push fixes
- Once merged to main, changes deploy to the target environment

The skill handles:
- Multi-environment migrations (dev, stage, prod) with isolated branches
- Resource name preservation (primary goal)
- Semantic diff validation with render-diff
- Comprehensive migration documentation and tracking
- Template gating and values file conversion
- Regional values file support (multi-region deployments)

## Installation

### Prerequisites

Before installing this skill, ensure you have:

1. **Claude Code CLI** installed and configured
2. **Helm** (not version 4)
   ```bash
   helm version --short
   ```
3. **render-diff tool** from https://github.com/mozilla/mozcloud/tree/main/tools/render-diff
   ```bash
   which render-diff
   ```
4. **Access to Mozilla's OCI registry** to pull mozcloud charts
   ```bash
   helm show chart oci://us-west1-docker.pkg.dev/moz-fx-platform-artifacts/mozcloud-charts/mozcloud --version <latest>
   ```

### Setup Instructions

1. **Add the skill to your infra project's Claude skills directory:**

   You can either copy or symlink the skill. Symlinking is recommended if you want the skill to automatically stay in sync with updates in the mozcloud repository.

   **Option A: Symlink (Recommended)**
   ```bash
   # From your infra repository root (e.g., sandbox-infra)
   mkdir -p .claude/skills
   ln -s /path/to/mozcloud/claude/skills/mozcloud-chart-migration .claude/skills/mozcloud-chart-migration
   ```

   **Option B: Copy**
   ```bash
   # From your infra repository root (e.g., sandbox-infra)
   mkdir -p .claude/skills
   cp -r /path/to/mozcloud/claude/skills/mozcloud-chart-migration .claude/skills/
   ```

   If you're already in the mozcloud repository and want to set up the symlink:
   ```bash
   # Navigate to your infra project
   cd /path/to/sandbox-infra

   # Create the symlink (adjust the path to mozcloud as needed)
   mkdir -p .claude/skills
   ln -s ../mozcloud/claude/skills/mozcloud-chart-migration .claude/skills/mozcloud-chart-migration
   ```

2. **Verify installation:**

   ```bash
   # Check that the skill is present
   ls .claude/skills/mozcloud-chart-migration/SKILL.md
   ```

3. **Restart Claude Code** or run from the project directory to pick up the new skill.

## Usage

### Starting a Migration

1. **Navigate to the chart directory you want to migrate:**
   ```bash
   cd charts/your-custom-chart
   ```

2. **Invoke the skill:**
   ```bash
   /mozcloud-chart-migration
   ```

   Or provide a specific environment:
   ```bash
   /mozcloud-chart-migration dev
   ```

3. **Follow the guided workflow:**
   - The skill will verify prerequisites
   - Create a migration branch
   - Identify available values files
   - Generate a migration plan for your review
   - Execute the migration with your approval
   - Run validation tests

### Resuming an Existing Migration

If a migration was previously started, the skill automatically detects this by reading `.migration/README.md` and continues from where it left off.

```bash
cd charts/your-custom-chart
/mozcloud-chart-migration
```

The skill will:
- Read the current migration status
- Identify which environment is being worked on
- Continue with the next steps
- Respect previous decisions and patterns

### Migration Workflow

The skill follows this process:

1. **Pre-flight Checks**: Verifies git status, Helm version (not v4), render-diff availability, and OCI registry access
2. **Setup**: Creates environment-specific migration branch and `.migration/` directory structure
3. **Planning**: Analyzes current chart, identifies values files, creates detailed migration plan for user review
4. **Execution**:
   - Converts values files to mozcloud schema format
   - Updates Chart.yaml with mozcloud dependency
   - Gates custom templates (preserves for non-migrated environments)
   - **Preserves original resource names** (requires approval for any changes)
5. **Testing**:
   - Runs `render-diff` for semantic equivalence
   - Validates resource count matches original
   - Verifies non-migrated environments show no changes
   - Creates detailed diff analysis
6. **Documentation**: Updates migration tracking files (STATUS.md, CHANGES_$ENV.md, DIFF_ANALYSIS_$ENV.md)

### Key Features

#### Resource Name Preservation
The skill prioritizes preserving original resource names to avoid disruptions. Any name changes require explicit user approval.

#### Multi-Environment Support
Migrate one environment at a time (dev → stage → prod) with isolated branches and comprehensive testing.

#### Migration Directory Structure
All migration artifacts are stored in `.migration/`:
```
.migration/
├── README.md                    # Current status and next steps (ENTRY POINT)
├── STATUS.md                    # Multi-environment progress tracker
├── MIGRATION_PLAN_$ENV.md       # Detailed plan per environment
├── CHANGES_$ENV.md              # Resource changes per environment
├── DIFF_ANALYSIS_$ENV.md        # Structured diff analysis
└── manifests/                   # Original manifests for comparison
    ├── dev/
    ├── stage/
    └── prod/
```

#### Regional Values Support
Automatically handles multi-region deployments with files like `values-stage-europe-west1.yaml`.

## Safety Guarantees

The skill includes several safety mechanisms:

- **No automatic commits**: All changes require user review before committing
- **No destructive commands**: Never runs `helm install`, `helm upgrade`, `helm delete`, or other destructive operations
- **Environment isolation**: Verifies non-migrated environments show no changes
- **Scoped file writes**: Only writes to the chart being migrated and `.migration/` directory
- **ArgoCD-aware**: Designed for ArgoCD deployment workflow with simple rollback (delete branch or push fixes)
- **Resource name preservation**: Requires explicit approval for any resource name changes
- **Semantic validation**: Uses `render-diff` to verify resource equivalence before suggesting commit

### Permission Requests

**Important**: Claude Code will prompt you to approve certain operations during migration. Please carefully review these permission requests:

- **What to expect**: The skill may request permission for file operations (reading/writing files), git commands (creating branches, checking status), and shell commands (running helm, render-diff, etc.)
- **Review carefully**: Before approving, verify the operation makes sense for the current migration step
- **Deny if unexpected**: If a request seems unusual or outside the scope of chart migration (e.g., accessing environment variables, running deployment commands), deny it and ask for clarification
- **The skill will explain**: Each operation should be clearly described - if it's not clear what the skill is doing or why, ask before approving

The skill is designed to work within the constraints of your permission settings and will explain what it's doing at each step.

## Troubleshooting

### "render-diff not found"
Install the render-diff tool from https://github.com/mozilla/mozcloud/tree/main/tools/render-diff

### "Cannot access OCI registry"
Ensure you're authenticated to the GCP artifact registry:
```bash
gcloud auth configure-docker us-west1-docker.pkg.dev
```

### "Helm version 4 detected"
Helm v4 has breaking changes. Please use Helm v3 for compatibility.

### render-diff shows missing resources
- Check that mozcloud dependency is enabled in values for the migrated environment
- Verify workload names match original deployment names exactly
- Review template gating conditions to ensure custom templates are properly disabled

### Chart.lock conflicts
- Run `helm dependency update` after modifying Chart.yaml
- Ensure OCI registry is accessible

### Resource names changing unexpectedly
- Verify mozcloud workload key matches full original deployment name
- Check for `nameOverride` or `fullnameOverride` in values files
- Review mozcloud chart's naming conventions in `values.schema.json`
- The skill will ask for approval before implementing name changes

### Non-migrated environments showing differences
- Ensure template gating is working correctly (templates wrapped in conditional logic)
- Verify mozcloud dependency is disabled for non-migrated environments
- Run `render-diff -f values-<env>.yaml` for each non-migrated environment

### Files created in wrong location
This should be prevented by the Working Directory Management implementation. If encountered:
- Check that `$CHART_DIR` variable is set correctly
- Verify all file operations use absolute paths with `$CHART_DIR`
- See Technical References section above

## Examples

### Simple Migration
```bash
cd charts/my-app
/mozcloud-chart-migration dev
# Follow prompts, review plan, approve execution
```

### Multi-Region Migration
```bash
cd charts/my-multi-region-app
/mozcloud-chart-migration stage
# Skill automatically detects and handles regional values files
```

### Resuming Work
```bash
cd charts/my-app
/mozcloud-chart-migration
# Skill reads .migration/README.md and continues where left off
```

## Technical References

Detailed implementation documentation is available in the `references/` directory:

### [Working Directory Management](references/working-directory-management.md)
**Critical for preventing files from being created in wrong locations.**

The skill maintains a consistent working directory by:
- Capturing `$CHART_DIR` at the start of migration
- Using absolute paths for all file operations
- Verifying location after helm operations (which may change directory context)
- Safety checks before creating files

### [Mozcloud Chart Reference](references/mozcloud-chart-reference.md)
Complete documentation about the mozcloud chart:
- **OCI Repository**: `oci://us-west1-docker.pkg.dev/moz-fx-platform-artifacts/mozcloud-charts/mozcloud`
- Chart structure and schema details
- Common migration patterns
- Values file conventions
- Troubleshooting chart-specific issues

### [Migration Directory Structure](references/migration-directory-structure.md)
Detailed breakdown of the `.migration/` directory:
- Purpose of each file (README.md, STATUS.md, CHANGES_*.md, etc.)
- Example content for each file type
- Best practices for documentation
- When and how to update files

### [Troubleshooting Guide](references/troubleshooting.md)
Comprehensive troubleshooting reference:
- Render-diff issues
- OCI registry authentication
- Resource naming problems
- Template gating issues
- Helm version compatibility
- And more...

## Additional Resources

- **Mozcloud Chart Repository**: https://github.com/mozilla/helm-charts
- **Render-diff Tool**: https://github.com/mozilla/mozcloud/tree/main/tools/render-diff
- **Migration Documentation**: See `.migration/README.md` in any migrated chart for detailed progress and decisions
- **Complete Skill Documentation**: See `SKILL.md` for full implementation details

## Support

For issues or questions:
1. Check `.migration/README.md` in your chart for migration-specific context
2. Review the skill's `SKILL.md` for detailed workflow documentation
3. Consult with the platform team for mozcloud chart schema questions
