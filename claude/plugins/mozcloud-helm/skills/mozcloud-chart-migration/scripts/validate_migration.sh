#!/bin/bash
#
# validate_migration.sh - Validate migration using render-diff and manifest comparison
#
# Usage: ./validate_migration.sh <environment> [--all-environments] [--regional-values <file>]
#
# Arguments:
#   environment: The environment being migrated (dev, stage, prod, etc.)
#   --all-environments: Also validate non-migrated environments show no changes
#   --regional-values: Additional regional values file for multi-region deployments
#
# Examples:
#   ./validate_migration.sh dev
#   ./validate_migration.sh stage --all-environments
#   ./validate_migration.sh prod --regional-values values-prod-europe-west1.yaml
#
# Performs:
#   1. Semantic diff check with render-diff for migrated environment
#   2. Resource count validation
#   3. Optional: Validate non-migrated environments unchanged
#   4. Generate diff analysis report
#
# Exit codes:
#   0 - Validation passed
#   1 - Validation failed or errors encountered
#   2 - Validation passed with warnings

set -e

# Parse arguments
ENV=""
CHECK_ALL_ENVS=false
REGIONAL_VALUES=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --all-environments)
            CHECK_ALL_ENVS=true
            shift
            ;;
        --regional-values)
            REGIONAL_VALUES="$2"
            shift 2
            ;;
        *)
            if [[ -z "$ENV" ]]; then
                ENV="$1"
            else
                echo "Error: Unknown argument: $1"
                exit 1
            fi
            shift
            ;;
    esac
done

if [[ -z "$ENV" ]]; then
    echo "Error: Environment argument required"
    echo "Usage: $0 <environment> [--all-environments] [--regional-values <file>]"
    exit 1
fi

# Capture chart root
CHART_DIR=$(pwd)

if [[ ! -f "$CHART_DIR/Chart.yaml" ]]; then
    echo "ERROR: Chart.yaml not found. Not in a chart directory."
    exit 1
fi

CHART_NAME=$(grep "^name:" "$CHART_DIR/Chart.yaml" | awk '{print $2}' | tr -d '"' | tr -d "'")

echo "=== Migration Validation for $ENV environment ==="
echo "Chart: $CHART_NAME"
echo "Environment: $ENV"
echo

VALIDATION_FAILED=false
VALIDATION_WARNINGS=false

# Step 1: Semantic diff check with render-diff
echo "Step 1: Running semantic diff check..."
echo "Command: render-diff -f values-${ENV}.yaml -su"

if [[ -n "$REGIONAL_VALUES" ]]; then
    echo "Including regional values: $REGIONAL_VALUES"
fi

RENDER_DIFF_OUTPUT=$(mktemp)

# Run render-diff
if [[ -n "$REGIONAL_VALUES" ]]; then
    render-diff -f "values-${ENV}.yaml" -f "$REGIONAL_VALUES" -su > "$RENDER_DIFF_OUTPUT" 2>&1 || true
else
    render-diff -f "values-${ENV}.yaml" -su > "$RENDER_DIFF_OUTPUT" 2>&1 || true
fi

# Display output
cat "$RENDER_DIFF_OUTPUT"

# Check for differences
if grep -q "No differences found" "$RENDER_DIFF_OUTPUT"; then
    echo "[OK] Semantic diff check passed: No differences"
elif grep -q "differences found" "$RENDER_DIFF_OUTPUT"; then
    echo "[WARNING]  WARNING: Semantic differences detected"
    VALIDATION_WARNINGS=true
else
    echo "[WARNING]  WARNING: Could not parse render-diff output"
    VALIDATION_WARNINGS=true
fi

rm -f "$RENDER_DIFF_OUTPUT"
echo

# Step 2: Resource count validation
echo "Step 2: Validating resource count..."

ORIGINAL_MANIFEST="$CHART_DIR/.migration/manifests/$ENV/original.yaml"
if [[ ! -f "$ORIGINAL_MANIFEST" ]]; then
    echo "[WARNING]  WARNING: Original manifest not found at $ORIGINAL_MANIFEST"
    echo "   Cannot validate resource count. Run capture_manifests.sh first."
    VALIDATION_WARNINGS=true
else
    ORIGINAL_COUNT=$(grep -c "^kind:" "$ORIGINAL_MANIFEST" || echo "0")
    echo "Original resource count: $ORIGINAL_COUNT"

    # Capture current manifests
    TEMP_MANIFEST=$(mktemp)
    if [[ -n "$REGIONAL_VALUES" ]]; then
        helm template . -f "$CHART_DIR/values.yaml" -f "$CHART_DIR/values-${ENV}.yaml" -f "$CHART_DIR/$REGIONAL_VALUES" > "$TEMP_MANIFEST"
    else
        helm template . -f "$CHART_DIR/values.yaml" -f "$CHART_DIR/values-${ENV}.yaml" > "$TEMP_MANIFEST"
    fi

    CURRENT_COUNT=$(grep -c "^kind:" "$TEMP_MANIFEST" || echo "0")
    echo "Current resource count: $CURRENT_COUNT"

    if [[ "$CURRENT_COUNT" -lt "$ORIGINAL_COUNT" ]]; then
        echo "[FAIL] FAILED: Resource count decreased ($ORIGINAL_COUNT → $CURRENT_COUNT)"
        echo "  Missing resources! Check mozcloud.enabled flag and workload configuration."
        VALIDATION_FAILED=true
    elif [[ "$CURRENT_COUNT" -eq "$ORIGINAL_COUNT" ]]; then
        echo "[OK] Resource count matches: $CURRENT_COUNT resources"
    else
        echo "[OK] Resource count increased: $ORIGINAL_COUNT → $CURRENT_COUNT"
        echo "  (This may be expected if mozcloud adds resources like ExternalSecrets)"
    fi

    rm -f "$TEMP_MANIFEST"
fi
echo

# Step 3: Check non-migrated environments (if requested)
if [[ "$CHECK_ALL_ENVS" == "true" ]]; then
    echo "Step 3: Validating non-migrated environments..."

    ALL_ENVS=("dev" "stage" "prod" "preview")
    for CHECK_ENV in "${ALL_ENVS[@]}"; do
        if [[ "$CHECK_ENV" == "$ENV" ]]; then
            continue  # Skip the environment we're migrating
        fi

        VALUES_FILE="$CHART_DIR/values-${CHECK_ENV}.yaml"
        if [[ ! -f "$VALUES_FILE" ]]; then
            continue  # Skip if values file doesn't exist
        fi

        echo "Checking $CHECK_ENV environment..."
        CHECK_OUTPUT=$(mktemp)
        render-diff -f "values-${CHECK_ENV}.yaml" > "$CHECK_OUTPUT" 2>&1 || true

        if grep -q "No differences found" "$CHECK_OUTPUT"; then
            echo "  [OK] $CHECK_ENV: No changes (correct)"
        else
            echo "  [FAIL] $CHECK_ENV: Unexpected changes detected!"
            cat "$CHECK_OUTPUT"
            VALIDATION_FAILED=true
        fi

        rm -f "$CHECK_OUTPUT"
    done
    echo
fi

# Step 4: Generate diff analysis summary
echo "Step 4: Generating diff analysis..."
DIFF_ANALYSIS="$CHART_DIR/.migration/DIFF_ANALYSIS_${ENV}.md"

if [[ -f "$ORIGINAL_MANIFEST" ]]; then
    # Capture current manifests
    MIGRATED_MANIFEST="$CHART_DIR/.migration/manifests/$ENV/migrated.yaml"
    if [[ -n "$REGIONAL_VALUES" ]]; then
        helm template . -f "$CHART_DIR/values.yaml" -f "$CHART_DIR/values-${ENV}.yaml" -f "$CHART_DIR/$REGIONAL_VALUES" > "$MIGRATED_MANIFEST"
    else
        helm template . -f "$CHART_DIR/values.yaml" -f "$CHART_DIR/values-${ENV}.yaml" > "$MIGRATED_MANIFEST"
    fi

    # Analyze differences
    echo "Comparing manifests: original vs migrated"
    ADDED_RESOURCES=$(comm -13 <(grep "^kind:" "$ORIGINAL_MANIFEST" | sort) <(grep "^kind:" "$MIGRATED_MANIFEST" | sort) | wc -l | tr -d ' ')
    REMOVED_RESOURCES=$(comm -23 <(grep "^kind:" "$ORIGINAL_MANIFEST" | sort) <(grep "^kind:" "$MIGRATED_MANIFEST" | sort) | wc -l | tr -d ' ')

    echo "  Added resources: $ADDED_RESOURCES"
    echo "  Removed resources: $REMOVED_RESOURCES"

    if [[ "$REMOVED_RESOURCES" -gt 0 ]]; then
        echo "  [WARNING]  WARNING: Resources were removed!"
        VALIDATION_FAILED=true
    fi

    # Save summary
    cat > "$DIFF_ANALYSIS" << EOF
# Diff Analysis: $ENV Environment

Generated: $(date)

## Resource Impact Summary
- **Original count**: $(grep -c "^kind:" "$ORIGINAL_MANIFEST" || echo "0") resources
- **Migrated count**: $(grep -c "^kind:" "$MIGRATED_MANIFEST" || echo "0") resources
- **Added**: $ADDED_RESOURCES resources
- **Removed**: $REMOVED_RESOURCES resources

## Resource Breakdown

### Original Resources
\`\`\`
$(grep "^kind:" "$ORIGINAL_MANIFEST" | sort | uniq -c | sort -rn)
\`\`\`

### Migrated Resources
\`\`\`
$(grep "^kind:" "$MIGRATED_MANIFEST" | sort | uniq -c | sort -rn)
\`\`\`

## Next Steps
- Review the manifest diff manually: \`diff .migration/manifests/$ENV/original.yaml .migration/manifests/$ENV/migrated.yaml\`
- Document any intentional changes in CHANGES_${ENV}.md
- Ensure all critical resources are present
EOF

    echo "[OK] Diff analysis saved to: $DIFF_ANALYSIS"
else
    echo "[WARNING]  Skipping diff analysis: original manifest not found"
fi
echo

# Final summary
echo "=== Validation Summary ==="
if [[ "$VALIDATION_FAILED" == "true" ]]; then
    echo "[FAIL] VALIDATION FAILED"
    echo "  Please review the errors above and fix issues before proceeding."
    exit 1
elif [[ "$VALIDATION_WARNINGS" == "true" ]]; then
    echo "[WARNING]  VALIDATION PASSED WITH WARNINGS"
    echo "  Review warnings above. Changes may be intentional."
    exit 2
else
    echo "[OK] VALIDATION PASSED"
    echo "  Migration appears successful. Review diff analysis and commit when ready."
    exit 0
fi
