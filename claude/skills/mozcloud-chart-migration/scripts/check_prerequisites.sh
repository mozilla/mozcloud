#!/bin/bash
#
# check_prerequisites.sh - Verify all required tools and access for mozcloud migration
#
# Usage: ./check_prerequisites.sh [--skip-git-check]
#
# Exit codes:
#   0 - All prerequisites met
#   1 - One or more prerequisites failed

set -e

SKIP_GIT_CHECK=false
if [[ "$1" == "--skip-git-check" ]]; then
    SKIP_GIT_CHECK=true
fi

echo "=== Mozcloud Migration Prerequisites Check ==="
echo

# Track overall status
ALL_CHECKS_PASSED=true

# 1. Git Status Check
if [[ "$SKIP_GIT_CHECK" == "false" ]]; then
    echo "1. Checking git status..."
    if [[ -n $(git status --porcelain) ]]; then
        echo "   [WARNING]  WARNING: Uncommitted changes detected"
        echo "   You may want to commit or stash changes before starting migration"
        echo "   (This is a warning, not a blocker)"
    else
        echo "   [OK] Working directory is clean"
    fi
else
    echo "1. Skipping git status check (--skip-git-check flag provided)"
fi
echo

# 2. Helm Version Check
echo "2. Checking Helm version..."
if ! command -v helm &> /dev/null; then
    echo "   [FAIL] FAILED: helm command not found"
    echo "   Please install Helm: https://helm.sh/docs/intro/install/"
    ALL_CHECKS_PASSED=false
else
    HELM_VERSION=$(helm version --short 2>/dev/null || helm version 2>/dev/null | head -n 1)
    echo "   Found: $HELM_VERSION"

    # Check if it's version 4
    if echo "$HELM_VERSION" | grep -q "v4\."; then
        echo "   [FAIL] FAILED: Helm v4 detected"
        echo "   Helm v4 has breaking changes. Please use Helm v3 for compatibility"
        ALL_CHECKS_PASSED=false
    else
        echo "   [OK] Helm version is compatible"
    fi
fi
echo

# 3. render-diff Tool Check
echo "3. Checking render-diff availability..."
if ! command -v render-diff &> /dev/null; then
    echo "   [FAIL] FAILED: render-diff command not found"
    echo "   Please install from: https://github.com/mozilla/mozcloud/tree/main/tools/render-diff"
    ALL_CHECKS_PASSED=false
else
    RENDER_DIFF_PATH=$(which render-diff)
    echo "   [OK] Found at: $RENDER_DIFF_PATH"
fi
echo

# 4. OCI Registry Access Check
echo "4. Checking mozcloud chart OCI registry access..."
MOZCLOUD_REGISTRY="oci://us-west1-docker.pkg.dev/moz-fx-platform-artifacts/mozcloud-charts/mozcloud"

# Try to get the latest version
if helm show chart "$MOZCLOUD_REGISTRY" --version latest &> /dev/null; then
    LATEST_VERSION=$(helm show chart "$MOZCLOUD_REGISTRY" --version latest 2>/dev/null | grep "^version:" | awk '{print $2}')
    echo "   [OK] OCI registry is accessible"
    echo "   Latest mozcloud chart version: $LATEST_VERSION"
else
    echo "   [FAIL] FAILED: Cannot access OCI registry"
    echo "   You may need to authenticate:"
    echo "   gcloud auth configure-docker us-west1-docker.pkg.dev"
    ALL_CHECKS_PASSED=false
fi
echo

# 5. Chart.yaml Check
echo "5. Checking for Chart.yaml in current directory..."
if [[ ! -f "Chart.yaml" ]]; then
    echo "   [FAIL] FAILED: Chart.yaml not found in current directory"
    echo "   Please run this script from a Helm chart directory"
    ALL_CHECKS_PASSED=false
else
    CHART_NAME=$(grep "^name:" Chart.yaml | awk '{print $2}')
    echo "   [OK] Found Chart.yaml"
    echo "   Chart name: $CHART_NAME"
fi
echo

# Final Summary
echo "=== Prerequisites Check Summary ==="
if [[ "$ALL_CHECKS_PASSED" == "true" ]]; then
    echo "[OK] All prerequisites met. Ready to start migration!"
    exit 0
else
    echo "[FAIL] Some prerequisites failed. Please address the issues above before continuing."
    exit 1
fi
