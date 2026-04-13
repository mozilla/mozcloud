# Troubleshooting Common Issues

## ArgoCD sync fails with "spec.selector is immutable"

**Symptoms:**
```
Deployment.apps "myapp" is invalid: spec.selector: Invalid value: ...: field is immutable
```

**Cause:**
mozcloud changes the pod selector labels from the original chart's convention (e.g. `app.kubernetes.io/name: myapp`, `app.kubernetes.io/instance: dev`) to its own convention (e.g. `app.kubernetes.io/name: <app_code>`, `env_code: dev`). Kubernetes does not allow patching `spec.selector` on an existing Deployment — the resource must be deleted and recreated.

**Important — selector labels are immutable and should be preserved when possible:**
`spec.selector` is set once at Deployment creation time and can never be changed. Any migration that modifies selector labels forces a Deployment delete-and-recreate, causing a period of unavailability. When designing or modifying a chart, always check whether selector labels will change before merging — if they can be preserved, preserve them. For mozcloud migrations, the selector change is unavoidable because mozcloud enforces its own label convention (`app.kubernetes.io/name: <app_code>`, `env_code: <env>`), but this should be called out explicitly in the migration plan and communicated to the team before the sync.

**Fix:**
Delete the affected Deployments via the ArgoCD web interface and re-trigger the sync. ArgoCD will recreate them immediately with the new selectors. There will be a brief period of unavailability during recreation.

In the ArgoCD UI: navigate to the application, find the affected Deployment resources, select each one, and use **Delete** to remove them. Then trigger a fresh sync — ArgoCD will recreate them with the new selectors.

**Note:** This is expected for every tenant migrating to mozcloud and should be communicated to the team before merging. Plan the sync during a low-traffic window for production environments.

**Zero-Downtime Alternative (Parallel Deployment Strategy):**

If the service cannot tolerate a brief outage during selector migration, use new permanent workload names that do not conflict with the existing Deployment names. This requires a two-phase approach.

**Critical constraint — workload key controls both Deployment and Service name:**

In mozcloud, the workload key becomes both the Deployment name and the Service name. There is no way to use the same workload key as the original Deployment while avoiding the immutable selector conflict. This means:

- The new Deployment name will be different from the original (permanent change)
- The new Service name will also be different from the original (permanent change)
- The new Ingress will point to the new Service — traffic does NOT automatically split between old and new pods

The old Service is NOT retained automatically. When `mozcloud.enabled: true`, the old Service template is gated out and removed from ArgoCD's desired state. For old pods to continue receiving traffic during Phase 1, the old Service must be kept alive through explicit template gating.

**Phase 1 — Run old and new simultaneously (migration PR)**

Gate out only the old **Deployments** (to avoid the immutable selector conflict), but keep the old **Service** and **Ingress** alive by gating them separately. This requires splitting the gate condition in the old templates — Deployment templates get the full `mozcloud.enabled` gate, while Service and Ingress templates temporarily remain ungated or use a separate flag.

Add the new mozcloud workloads with permanent new names alongside:

```yaml
# Example: original Deployments were myapp, myapp-worker, etc.
# New permanent names (prefix with app_code or another stable identifier):
mozcloud:
  workloads:
    <app_code>-myapp:
      component: app
      ...
    <app_code>-myapp-worker:
      component: worker
      ...
```

After Phase 1 merge:
- Old Service (`myapp`) remains, routing Ingress traffic to the old pods
- Old pods continue running as orphaned Deployments (not in desired state — ArgoCD prune is disabled, so they remain until explicitly pruned)
- New Deployments (`<app_code>-myapp`, etc.) start up alongside them
- New mozcloud Service (`<app_code>-myapp`) and Ingress are created but not yet the live traffic path

Verify the new Deployments are healthy before proceeding.

**Phase 2 — Cutover (follow-up PR)**

Once the new Deployments are confirmed healthy:

1. Update the Ingress (or DNS) to route traffic to the new Service
2. Apply the full `mozcloud.enabled` gate to remove the old Service and Ingress from desired state
3. Sync ArgoCD, then prune the old Deployments, Service, and Ingress via the **ArgoCD web interface** (navigate to the application, select each orphaned resource, and use the Delete action)

The new names are permanent going forward — no rename step needed.

**Trade-offs:**
- Deployment and Service names change permanently (update any internal service-to-service callers that reference the old Service name)
- Requires careful split of template gating between Deployment and Service/Ingress templates
- Worker/cron Deployments will process jobs in both old and new instances during the overlap window — verify idempotency before using this approach for them
- Monitoring dashboards and alerts keyed on old Deployment/Service names must be updated
- Two PRs/syncs required instead of one

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

## Unexpected PodMonitoring resource for telegraf (mozcloud ≥ 0.15.0)

**Symptoms:**
- `helm template` output includes a `PodMonitoring` resource named `{app}-telegraf`
- This resource was not present in the original chart

**Explanation:** mozcloud added a telegraf `PodMonitoring` resource that is enabled by default.

**Solution:** If the original chart did not have telegraf, disable it explicitly:
```yaml
mozcloud:
  telegraf:
    enabled: false
```

## Local file:// chart dependency version mismatch

**Context:** This issue only arises when testing changes to the mozcloud chart itself using a local `file://` repository reference (e.g., during mozcloud development or pre-release validation). Normal migration users pull from the OCI registry and won't encounter this.

**Symptoms:**
- `helm dependency update` fails with "can't get a valid version for subchart"
- Error references a `file:///path/to/local/chart` repository

**Cause:** The `version` in the app chart's `Chart.yaml` doesn't match the `version` declared in the local mozcloud chart's `Chart.yaml`.

**Solution:** Update the app chart's dependency version to match the local chart's actual version:
```bash
grep '^version:' /path/to/local/mozcloud/Chart.yaml
# Then update Chart.yaml accordingly and re-run helm dependency update
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
  - Use `<tenant>-<component>-worker`, NOT `<component>-worker`
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
