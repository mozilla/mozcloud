#!/bin/bash
#
# setup_migration.sh - Initialize migration directory structure and branch
#
# Usage: ./setup_migration.sh <environment> [<chart-name>]
#   environment: dev, stage, prod, preview, etc.
#   chart-name: (optional) chart name, defaults to name from Chart.yaml
#
# Example: ./setup_migration.sh dev
#          ./setup_migration.sh stage my-custom-chart
#
# Creates:
#   - Migration branch: claude-migration-<chart-name>-<env>[-<date>]
#   - .migration/ directory structure
#   - Downloads mozcloud chart to .migration/mozcloud/
#
# Exit codes:
#   0 - Success
#   1 - Error (missing Chart.yaml, invalid input, etc.)

set -e

# Parse arguments
ENV=$1
CHART_NAME_OVERRIDE=$2

if [[ -z "$ENV" ]]; then
    echo "Error: Environment argument required"
    echo "Usage: $0 <environment> [<chart-name>]"
    echo "Example: $0 dev"
    exit 1
fi

# Step 1: Capture and verify chart root directory
CHART_DIR=$(pwd)
echo "Chart root: $CHART_DIR"

if [[ ! -f "$CHART_DIR/Chart.yaml" ]]; then
    echo "ERROR: Chart.yaml not found. Not in a chart directory."
    exit 1
fi

# Get chart name from Chart.yaml or use override
if [[ -n "$CHART_NAME_OVERRIDE" ]]; then
    CHART_NAME="$CHART_NAME_OVERRIDE"
else
    CHART_NAME=$(grep "^name:" "$CHART_DIR/Chart.yaml" | awk '{print $2}' | tr -d '"' | tr -d "'")
fi

if [[ -z "$CHART_NAME" ]]; then
    echo "ERROR: Could not determine chart name from Chart.yaml"
    exit 1
fi

echo "Chart name: $CHART_NAME"
echo "Environment: $ENV"
echo

# Step 2: Create migration branch
BRANCH_NAME="claude-migration-${CHART_NAME}-${ENV}"

# Check if branch already exists
if git rev-parse --verify "$BRANCH_NAME" &> /dev/null; then
    echo "Branch '$BRANCH_NAME' already exists."
    echo "Appending date for uniqueness..."
    BRANCH_NAME="${BRANCH_NAME}-$(date +%Y%m%d-%H%M%S)"
fi

echo "Creating migration branch: $BRANCH_NAME"
git checkout -b "$BRANCH_NAME"
echo "[OK] Created and checked out branch: $BRANCH_NAME"
echo

# Step 3: Create migration directory structure
echo "Creating migration directory structure..."
mkdir -p "$CHART_DIR/.migration/manifests/dev"
mkdir -p "$CHART_DIR/.migration/manifests/stage"
mkdir -p "$CHART_DIR/.migration/manifests/prod"
mkdir -p "$CHART_DIR/.migration/manifests/preview"

# Verify directories were created
if [[ -d "$CHART_DIR/.migration" ]]; then
    echo "[OK] Created .migration directory structure:"
    ls -la "$CHART_DIR/.migration/"
else
    echo "ERROR: Failed to create .migration directory"
    exit 1
fi
echo

# Step 4: Download mozcloud reference chart
echo "Downloading latest mozcloud chart from OCI registry..."

# Get the latest version
MOZCLOUD_REGISTRY="oci://us-west1-docker.pkg.dev/moz-fx-platform-artifacts/mozcloud-charts/mozcloud"
LATEST_VERSION=$(helm show chart "$MOZCLOUD_REGISTRY" --version latest 2>/dev/null | grep "^version:" | awk '{print $2}')

if [[ -z "$LATEST_VERSION" ]]; then
    echo "ERROR: Could not determine latest mozcloud chart version"
    echo "Check OCI registry access: $MOZCLOUD_REGISTRY"
    exit 1
fi

echo "Latest mozcloud version: $LATEST_VERSION"

# Download and extract the chart
helm pull "$MOZCLOUD_REGISTRY" \
    --version "$LATEST_VERSION" \
    --untar \
    --untardir "$CHART_DIR/.migration/"

# IMPORTANT: Return to chart root (helm operations may change directory)
cd "$CHART_DIR"
pwd

# Verify mozcloud chart was downloaded correctly
if [[ -d "$CHART_DIR/.migration/mozcloud" ]]; then
    echo "[OK] Downloaded mozcloud chart to .migration/mozcloud/"
    ls -la "$CHART_DIR/.migration/mozcloud/" | head -n 10
else
    echo "ERROR: Failed to download mozcloud chart"
    exit 1
fi
echo

# Step 5: Check if .gitignore includes .migration
if [[ -f "$CHART_DIR/.gitignore" ]]; then
    if grep -q "^\.migration/$" "$CHART_DIR/.gitignore"; then
        echo "[OK] .migration/ already in .gitignore"
    else
        echo "[WARNING]  NOTE: Consider adding .migration/ to .gitignore"
        echo "   The .migration directory contains local documentation and should be git-ignored"
    fi
else
    echo "[WARNING]  NOTE: No .gitignore found. Consider creating one and adding .migration/"
fi
echo

# Step 6: Initialize README.md
README_PATH="$CHART_DIR/.migration/README.md"
if [[ ! -f "$README_PATH" ]]; then
    echo "Creating initial .migration/README.md..."
    cat > "$README_PATH" << EOF
# Migration Status: $CHART_NAME

## Current State
- **Branch**: $BRANCH_NAME
- **Environment**: $ENV
- **Status**: Setup complete, ready to start migration
- **Started**: $(date +%Y-%m-%d)

## Next Steps
1. Review values files (values.yaml, values-$ENV.yaml)
2. Capture original manifests
3. Create migration plan
4. Review plan with user
5. Execute migration
6. Validate with render-diff

## Notes
- Migration directory structure created
- Mozcloud chart version $LATEST_VERSION downloaded to .migration/mozcloud/
- Ready to begin analysis and planning
EOF
    echo "[OK] Created .migration/README.md"
else
    echo "[WARNING]  .migration/README.md already exists, not overwriting"
fi
echo

# Final summary
echo "=== Setup Complete ==="
echo "Branch: $BRANCH_NAME"
echo "Chart: $CHART_NAME"
echo "Environment: $ENV"
echo "Mozcloud version: $LATEST_VERSION"
echo
echo "Next steps:"
echo "1. Capture original manifests with: ./scripts/capture_manifests.sh $ENV"
echo "2. Begin migration planning"
