# Mozcloud Chart Migration Skill

A Claude Code skill that assists with migrating custom tenant Helm charts to the shared `mozcloud` Helm chart.

## Overview

This skill automates and guides the process of migrating custom Helm charts to use Mozilla's shared `mozcloud` chart, which is stored in the OCI repository `oci://us-west1-docker.pkg.dev/moz-fx-platform-artifacts/mozcloud-charts`.

The skill handles:
- Multi-environment migrations (dev, stage, prod)
- Resource name preservation
- Semantic diff validation
- Migration documentation and tracking
- Template gating and values file conversion
- Regional values file support

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
   helm show chart oci://us-west1-docker.pkg.dev/moz-fx-platform-artifacts/mozcloud-charts/application --version <latest>
   ```

### Setup Instructions

1. **Copy the skill to your project's Claude skills directory:**

   ```bash
   # From the sandbox-infra repository root
   mkdir -p .claude/skills
   cp -r /path/to/mozcloud/claude/skills/mozcloud-chart-migration .claude/skills/
   ```

   Or if you're already in the mozcloud repository:

   ```bash
   # Navigate to your sandbox-infra project
   cd /path/to/sandbox-infra

   # Copy the skill
   mkdir -p .claude/skills
   cp -r /path/to/mozcloud/claude/skills/mozcloud-chart-migration .claude/skills/
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

1. **Pre-flight Checks**: Verifies git status, required tools, and OCI registry access
2. **Setup**: Creates migration branch and `.migration/` directory
3. **Planning**: Analyzes current chart and creates detailed migration plan
4. **Execution**: Converts values files, updates Chart.yaml, gates templates
5. **Testing**: Runs render-diff and validates resource equivalence
6. **Documentation**: Updates migration tracking files

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

- **No automatic commits**: All changes require user review
- **No destructive commands**: Never runs `helm install`, `helm upgrade`, or `helm delete`
- **Non-environment isolation**: Verifies other environments show no changes
- **ArgoCD-aware**: Designed for ArgoCD deployment workflow with safe rollback

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

### Resource count mismatch
Check `.migration/DIFF_ANALYSIS_$ENV.md` for detailed resource comparison and verify mozcloud dependency is enabled.

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

## Additional Resources

- **Mozcloud Chart Repository**: https://github.com/mozilla/helm-charts
- **Render-diff Tool**: https://github.com/mozilla/mozcloud/tree/main/tools/render-diff
- **Migration Documentation**: See `.migration/README.md` in any migrated chart for detailed progress and decisions

## Support

For issues or questions:
1. Check `.migration/README.md` in your chart for migration-specific context
2. Review the skill's `SKILL.md` for detailed workflow documentation
3. Consult with the platform team for mozcloud chart schema questions
