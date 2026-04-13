#!/bin/bash
#
# capture_manifests.sh - Render and save Helm chart manifests for migration comparison
#
# Usage: ./capture_manifests.sh <environment> [--regional-values <regional-file>] [--output-suffix <suffix>]
#
# Arguments:
#   environment: dev, stage, prod, preview, etc.
#   --regional-values: (optional) Additional regional values file (e.g., values-stage-europe-west1.yaml)
#   --output-suffix: (optional) Suffix for output file (default: "original" or "regional-original")
#
# Examples:
#   ./capture_manifests.sh dev
#   ./capture_manifests.sh stage --regional-values values-stage-europe-west1.yaml
#   ./capture_manifests.sh prod --output-suffix migrated
#
# Outputs:
#   .migration/manifests/<env>/original.yaml (or <suffix>.yaml)
#
# Exit codes:
#   0 - Success
#   1 - Error

set -e

# Parse arguments
ENV=""
REGIONAL_VALUES=""
OUTPUT_SUFFIX=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --regional-values)
            REGIONAL_VALUES="$2"
            shift 2
            ;;
        --output-suffix)
            OUTPUT_SUFFIX="$2"
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
    echo "Usage: $0 <environment> [--regional-values <file>] [--output-suffix <suffix>]"
    exit 1
fi

# Capture chart root directory
CHART_DIR=$(pwd)

if [[ ! -f "$CHART_DIR/Chart.yaml" ]]; then
    echo "ERROR: Chart.yaml not found. Not in a chart directory."
    exit 1
fi

# Verify values files exist
VALUES_FILE="$CHART_DIR/values-${ENV}.yaml"
if [[ ! -f "$CHART_DIR/values.yaml" ]]; then
    echo "ERROR: values.yaml not found in $CHART_DIR"
    exit 1
fi

if [[ ! -f "$VALUES_FILE" ]]; then
    echo "WARNING: $VALUES_FILE not found. Will only use values.yaml"
    VALUES_FILE=""
fi

# Determine output file
if [[ -z "$OUTPUT_SUFFIX" ]]; then
    if [[ -n "$REGIONAL_VALUES" ]]; then
        OUTPUT_SUFFIX="regional-original"
    else
        OUTPUT_SUFFIX="original"
    fi
fi

OUTPUT_DIR="$CHART_DIR/.migration/manifests/$ENV"
OUTPUT_FILE="$OUTPUT_DIR/${OUTPUT_SUFFIX}.yaml"

# Create output directory if needed
mkdir -p "$OUTPUT_DIR"

echo "=== Capturing Manifests for $ENV environment ==="
echo "Chart directory: $CHART_DIR"
echo "Base values: values.yaml"

# Build helm template command
HELM_CMD="helm template ."
HELM_CMD="$HELM_CMD -f $CHART_DIR/values.yaml"

if [[ -n "$VALUES_FILE" ]]; then
    echo "Environment values: values-${ENV}.yaml"
    HELM_CMD="$HELM_CMD -f $VALUES_FILE"
fi

if [[ -n "$REGIONAL_VALUES" ]]; then
    REGIONAL_PATH="$CHART_DIR/$REGIONAL_VALUES"
    if [[ ! -f "$REGIONAL_PATH" ]]; then
        echo "ERROR: Regional values file not found: $REGIONAL_PATH"
        exit 1
    fi
    echo "Regional values: $REGIONAL_VALUES"
    HELM_CMD="$HELM_CMD -f $REGIONAL_PATH"
fi

echo "Output: $OUTPUT_FILE"
echo

# Render the chart
echo "Rendering chart..."
echo "Command: $HELM_CMD"
eval "$HELM_CMD" > "$OUTPUT_FILE"

# Return to chart root (safety check)
cd "$CHART_DIR"

# Verify output file was created
if [[ ! -f "$OUTPUT_FILE" ]]; then
    echo "ERROR: Failed to create output file: $OUTPUT_FILE"
    exit 1
fi

# Count resources and lines
RESOURCE_COUNT=$(grep -c "^kind:" "$OUTPUT_FILE" || echo "0")
LINE_COUNT=$(wc -l < "$OUTPUT_FILE" | tr -d ' ')

echo "[OK] Manifests captured successfully"
echo
echo "=== Summary ==="
echo "Output file: $OUTPUT_FILE"
echo "File size: $(ls -lh "$OUTPUT_FILE" | awk '{print $5}')"
echo "Line count: $LINE_COUNT"
echo "Resource count: $RESOURCE_COUNT resources"
echo

# Show resource breakdown
echo "Resource types:"
grep "^kind:" "$OUTPUT_FILE" | sort | uniq -c | sort -rn

echo
echo "[OK] Capture complete!"
