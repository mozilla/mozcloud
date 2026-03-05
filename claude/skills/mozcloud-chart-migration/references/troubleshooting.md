# Troubleshooting Common Issues

## Render-diff shows missing resources

**Symptoms:**
- `render-diff` reports fewer resources in migrated chart
- Expected resources not appearing in output

**Solutions:**
- Check that mozcloud dependency is enabled in values
- Verify workload names match original deployment names
- Review template gating conditions
- Ensure custom templates are properly disabled for migrated environment
- Run `helm template` manually to see what's being rendered:
  ```bash
  helm template . -f values.yaml -f values-dev.yaml | grep -E "^kind:"
  ```

## Chart.lock conflicts

**Symptoms:**
- `helm template` fails with dependency errors
- Chart.lock shows wrong versions
- "dependency update needed" messages

**Solutions:**
- Run `helm dependency update` after modifying Chart.yaml
- Ensure OCI registry is accessible
- Delete Chart.lock and regenerate:
  ```bash
  rm Chart.lock
  helm dependency update
  ```
- Verify dependency versions match Chart.yaml

## Resource names changing unexpectedly

**Symptoms:**
- Deployment names differ from original
- Service names don't match expected values
- ConfigMap/Secret references break

**Solutions:**
- Verify mozcloud workload key matches full original name
  - Use `gha-fxa-profile-worker`, NOT `profile-worker`
- Check for `nameOverride` or `fullnameOverride` in values
- Review mozcloud chart's naming conventions in `values.schema.json`
- Compare rendered manifests side-by-side:
  ```bash
  diff <(kubectl get all --dry-run -o yaml) .migration/manifests/dev/migrated.yaml
  ```

## Non-migrated environments showing differences

**Symptoms:**
- `render-diff -f values-stage.yaml` shows changes when only dev was migrated
- Other environments rendering mozcloud resources unexpectedly

**Solutions:**
- Ensure template gating is working correctly
- Verify conditional logic in templates:
  ```yaml
  {{- if not .Values.mozcloud.enabled }}
  # original template content
  {{- end }}
  ```
- Check that mozcloud dependency is disabled for those environments:
  ```yaml
  # values-stage.yaml
  mozcloud:
    enabled: false
  ```
- Verify Chart.yaml dependency condition:
  ```yaml
  dependencies:
    - name: mozcloud
      condition: mozcloud.enabled  # Must be present
  ```

## Files created in wrong location

**Symptoms:**
- `.migration/` directory appears in wrong place
- Migration artifacts scattered across repo
- Can't find generated files

**Solutions:**
- Verify `$CHART_DIR` variable is set correctly:
  ```bash
  echo $CHART_DIR
  ls -la $CHART_DIR/Chart.yaml  # Should exist
  ```
- Always use absolute paths with `$CHART_DIR`
- Return to chart root after helm operations:
  ```bash
  cd $CHART_DIR
  pwd
  ```
- See [Working Directory Management](./working-directory-management.md) for complete guidance

## Helm version 4 compatibility issues

**Symptoms:**
- `helm version` shows v4.x.x
- Unexpected behavior or errors

**Solutions:**
- Helm v4 has breaking changes
- Use Helm v3 for compatibility:
  ```bash
  # Check version
  helm version --short

  # If v4, install v3 (example for macOS)
  brew install helm@3
  ```

## OCI registry authentication failures

**Symptoms:**
- "unauthorized" errors when pulling mozcloud chart
- "failed to authorize" messages
- Cannot access artifact registry

**Solutions:**
- Authenticate to GCP Artifact Registry:
  ```bash
  gcloud auth configure-docker us-west1-docker.pkg.dev
  ```
- Verify you have access to the project:
  ```bash
  gcloud projects list
  ```
- Check active account:
  ```bash
  gcloud auth list
  ```
- Try pulling chart manually:
  ```bash
  helm pull oci://us-west1-docker.pkg.dev/moz-fx-platform-artifacts/mozcloud-charts/mozcloud
  ```

## render-diff tool not found

**Symptoms:**
- `which render-diff` returns nothing
- Command not found errors

**Solutions:**
- Install from mozcloud repository:
  ```bash
  cd /path/to/mozcloud/tools/render-diff
  go install
  ```
- Verify installation:
  ```bash
  which render-diff
  render-diff --help
  ```
- Add to PATH if needed:
  ```bash
  export PATH="$PATH:$(go env GOPATH)/bin"
  ```

## Schema validation errors

**Symptoms:**
- `helm lint` fails
- Schema validation errors during template rendering
- "does not match schema" messages

**Solutions:**
- Review `.migration/mozcloud/values.schema.json`
- Compare your values structure with schema requirements
- Check for:
  - Required fields that are missing
  - Incorrect value types (string vs number)
  - Invalid enum values
- Run detailed validation:
  ```bash
  helm lint . -f values.yaml -f values-dev.yaml
  ```

## Resource count mismatch

**Symptoms:**
- Original chart renders 10 resources, migrated chart renders 8
- Missing resources in diff

**Solutions:**
- List resources in both versions:
  ```bash
  # Original
  helm template . -f values.yaml -f values-dev.yaml | grep -E "^kind:" | sort

  # Migrated (ensure mozcloud enabled)
  helm template . -f values.yaml -f values-dev.yaml | grep -E "^kind:" | sort
  ```
- Check `.migration/DIFF_ANALYSIS_dev.md` for detailed comparison
- Verify all resource types are configured in mozcloud values
- Check if mozcloud auto-generates resources (like ExternalSecrets)

## Regional values files not working

**Symptoms:**
- Multi-region deployments failing
- `values-stage-europe-west1.yaml` not being applied

**Solutions:**
- Test with both files:
  ```bash
  render-diff -f values-stage.yaml -f values-stage-europe-west1.yaml -su
  ```
- Ensure proper layering (base → environment → region):
  ```bash
  helm template . \
    -f values.yaml \
    -f values-stage.yaml \
    -f values-stage-europe-west1.yaml
  ```
- Verify regional overrides are properly structured

## Git merge conflicts in Chart.lock

**Symptoms:**
- Merge conflicts when updating Chart.lock
- Lock file out of sync

**Solutions:**
- Always regenerate Chart.lock after merging:
  ```bash
  git checkout --theirs Chart.lock  # or --ours
  helm dependency update
  git add Chart.lock
  ```
- Never manually edit Chart.lock
- Ensure Chart.yaml is correct before regenerating lock

## Migration documentation out of sync

**Symptoms:**
- `.migration/README.md` doesn't reflect current state
- Confusion about what's been completed

**Solutions:**
- Always read `.migration/README.md` first before making changes
- Update documentation after each major milestone
- Keep STATUS.md current with environment progress
- Use CHANGES_*.md to track all modifications
- Review and update before handing off or pausing work
