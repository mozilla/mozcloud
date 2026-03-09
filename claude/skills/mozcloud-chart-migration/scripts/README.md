# Migration Helper Scripts

This directory contains helper scripts that automate common operations during mozcloud chart migrations. These scripts are designed to be both:
- **Executed directly** by the skill (reducing token usage and improving reliability)
- **Run manually** by users for ad-hoc operations

## Scripts Overview

### 1. check_prerequisites.sh
Verifies all required tools and access before starting a migration.

**Usage:**
```bash
./scripts/check_prerequisites.sh [--skip-git-check]
```

**Checks:**
- Git status (uncommitted changes warning)
- Helm version (ensures not v4)
- render-diff availability
- OCI registry access
- Chart.yaml presence

**Exit codes:**
- `0` - All prerequisites met
- `1` - One or more prerequisites failed

**Example:**
```bash
# Run full check
./scripts/check_prerequisites.sh

# Skip git status check
./scripts/check_prerequisites.sh --skip-git-check
```

---

### 2. setup_migration.sh
Initializes the migration directory structure and creates the migration branch.

**Usage:**
```bash
./scripts/setup_migration.sh <environment> [<chart-name>]
```

**Arguments:**
- `environment` (required): dev, stage, prod, preview, etc.
- `chart-name` (optional): Override chart name from Chart.yaml

**Creates:**
- Migration branch: `claude-migration-<chart-name>-<env>[-<date>]`
- `.migration/` directory structure
- `.migration/manifests/{dev,stage,prod,preview}/` directories
- Downloads mozcloud chart to `.migration/mozcloud/`
- Initial `.migration/README.md`

**Exit codes:**
- `0` - Success
- `1` - Error (missing Chart.yaml, OCI access failure, etc.)

**Example:**
```bash
# Setup dev environment migration
./scripts/setup_migration.sh dev

# Setup with custom chart name
./scripts/setup_migration.sh stage my-custom-name
```

---

### 3. capture_manifests.sh
Renders Helm chart manifests and saves them for comparison during migration.

**Usage:**
```bash
./scripts/capture_manifests.sh <environment> [--regional-values <file>] [--output-suffix <suffix>]
```

**Arguments:**
- `environment` (required): dev, stage, prod, preview, etc.
- `--regional-values` (optional): Additional regional values file (e.g., `values-stage-europe-west1.yaml`)
- `--output-suffix` (optional): Custom suffix for output file (default: "original" or "regional-original")

**Outputs:**
- `.migration/manifests/<env>/<suffix>.yaml`

**Exit codes:**
- `0` - Success
- `1` - Error (missing values files, render failure, etc.)

**Examples:**
```bash
# Capture dev environment manifests
./scripts/capture_manifests.sh dev

# Capture stage with regional values
./scripts/capture_manifests.sh stage --regional-values values-stage-europe-west1.yaml

# Capture migrated manifests for comparison
./scripts/capture_manifests.sh prod --output-suffix migrated
```

---

### 4. validate_migration.sh
Validates migration using render-diff and manifest comparison.

**Usage:**
```bash
./scripts/validate_migration.sh <environment> [--all-environments] [--regional-values <file>]
```

**Arguments:**
- `environment` (required): The environment being migrated
- `--all-environments` (optional): Also validate non-migrated environments show no changes
- `--regional-values` (optional): Additional regional values file for multi-region deployments

**Performs:**
1. Semantic diff check with render-diff
2. Resource count validation (ensures no resources lost)
3. Optional: Validates non-migrated environments unchanged
4. Generates diff analysis report

**Outputs:**
- `.migration/DIFF_ANALYSIS_<env>.md`
- `.migration/manifests/<env>/migrated.yaml`

**Exit codes:**
- `0` - Validation passed
- `1` - Validation failed
- `2` - Validation passed with warnings

**Examples:**
```bash
# Validate dev environment only
./scripts/validate_migration.sh dev

# Validate stage and check all other environments
./scripts/validate_migration.sh stage --all-environments

# Validate with regional values
./scripts/validate_migration.sh prod --regional-values values-prod-europe-west1.yaml
```

---

## Typical Workflow

Here's how these scripts work together in a typical migration:

```bash
# 1. Navigate to chart directory
cd /path/to/charts/my-chart

# 2. Check prerequisites
./scripts/check_prerequisites.sh

# 3. Setup migration for dev environment
./scripts/setup_migration.sh dev

# 4. Capture original manifests
./scripts/capture_manifests.sh dev

# 5. [Make migration changes to values files and Chart.yaml]
# ... edit values-dev.yaml, update Chart.yaml, etc. ...

# 6. Validate the migration
./scripts/validate_migration.sh dev --all-environments

# 7. Review diff analysis
cat .migration/DIFF_ANALYSIS_dev.md

# 8. If validation passes, commit and push
git add -A
git commit -m "Migrate dev environment to mozcloud chart"
git push -u origin $(git branch --show-current)
```

## Multi-Region Example

For charts with regional values files:

```bash
# Setup
./scripts/setup_migration.sh stage
./scripts/capture_manifests.sh stage

# Capture regional variant
./scripts/capture_manifests.sh stage \
  --regional-values values-stage-europe-west1.yaml \
  --output-suffix regional-original

# [Make migration changes]

# Validate both configurations
./scripts/validate_migration.sh stage
./scripts/validate_migration.sh stage \
  --regional-values values-stage-europe-west1.yaml
```

## Benefits of Using Scripts

### For the Skill
- **Reduced token usage**: Execute scripts instead of loading full bash commands
- **Improved reliability**: Tested, debugged code vs generated bash
- **Faster execution**: Pre-written scripts run immediately
- **Consistent behavior**: Same logic every time

### For Users
- **Manual operations**: Run scripts independently for troubleshooting
- **Reproducibility**: Standardized commands across migrations
- **Documentation**: Clear usage examples and help text
- **Debugging**: Scripts include verbose output for transparency

## Script Design Principles

All scripts follow these principles:
1. **Absolute paths**: Use `$CHART_DIR` variable to prevent file creation in wrong locations
2. **Safety checks**: Verify Chart.yaml, directory structure, required files
3. **Clear output**: Verbose logging with [OK]/[FAIL]/[WARNING] indicators
4. **Error handling**: Exit codes and error messages for failures
5. **Idempotent**: Safe to run multiple times (where possible)
6. **Documentation**: Built-in help text and usage examples

## Troubleshooting

### "Permission denied" errors
Scripts need execute permission:
```bash
chmod +x scripts/*.sh
```

### Scripts fail to find Chart.yaml
Ensure you're running from the chart root directory:
```bash
pwd
ls Chart.yaml
```

### OCI registry access errors
Authenticate to GCP artifact registry:
```bash
gcloud auth configure-docker us-west1-docker.pkg.dev
```

### render-diff command not found
Install render-diff tool:
```bash
# From mozcloud repository
cd tools/render-diff
go install
```

## Contributing

When adding new scripts:
1. Follow the existing naming convention (`verb_noun.sh`)
2. Include usage documentation in header comments
3. Use consistent error handling and exit codes
4. Add verbose output with status indicators
5. Update this README with the new script
6. Make script executable: `chmod +x scripts/new_script.sh`
