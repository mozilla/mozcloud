# Cleanup Phase Guide

## Overview

After all environments are migrated to mozcloud and merged to main, run a final cleanup phase to:
1. Consolidate duplicate values into `values.yaml`
2. Remove unused custom templates
3. Simplify Chart.yaml dependencies
4. Archive migration documentation

## When to Run Cleanup

**Prerequisites**:
- All environment migrations merged to main
- All environments deployed and validated
- Team confirms environments are stable

**Do NOT run cleanup**:
- While environments are on separate branches
- Before all environments are deployed
- During active rollback or troubleshooting

## Invocation

```bash
cd /path/to/chart/directory
/mozcloud-chart-migration cleanup
```

## Cleanup Actions

### 1. Values Consolidation

**Goal**: Move common configuration from environment files to base `values.yaml`

**Process**:
1. Identify configuration duplicated across all `values-{env}.yaml` files
2. Move common config to `values.yaml`
3. Keep only environment-specific overrides in `values-{env}.yaml`
4. Validate with render-diff (must show zero changes)

**Common configuration to consolidate**:
- `mozcloud.app.name`, `realm`, `projectId`
- Base workload configuration (container image, ports, common env vars)
- ServiceAccount annotations (if using templating for env-specific values)
- Common HPA settings
- ConfigMap and ExternalSecret base configuration
- BackendConfig/FrontendConfig common settings

**Environment-specific overrides to keep**:
- `mozcloud.enabled` flag (if environments at different migration stages)
- Ingress/HTTPRoute hostnames
- Container image tags
- Resource limits/requests (if different per environment)
- Environment-specific environment variables
- Preview-specific configuration (`preview.enabled`, `preview.httpRoute`, etc.)

### 2. Template Removal

**Goal**: Remove custom templates fully replaced by mozcloud

**Templates typically removed**:
- `templates/rollout.yaml` or `templates/deployment.yaml`
- `templates/service.yaml`
- `templates/configmap.yaml`
- `templates/externalsecret.yaml`
- `templates/serviceaccount.yaml`
- `templates/hpa.yaml`
- `templates/ingress.yaml`
- `templates/backendconfig.yaml`
- `templates/frontendconfig.yaml`
- `templates/managedcert.yaml`
- `templates/preview-environments/httproute.yaml`
- `templates/preview-environments/endpoint-check.yaml`

**Templates to review (may keep)**:
- AnalysisTemplate resources (if planning to re-enable with Rollouts)
- Custom monitoring resources not provided by mozcloud
- Application-specific resources not part of workload

**Process**:
1. Review each template file
2. Verify mozcloud provides equivalent functionality
3. Delete template file if fully replaced
4. Run render-diff to confirm no changes

### 3. Chart.yaml Simplification

**Current state** (during migration):
```yaml
dependencies:
  - name: mozcloud
    version: "0.7.x"
    repository: "oci://us-west1-docker.pkg.dev/moz-fx-platform-artifacts/mozcloud-charts"
    condition: mozcloud.enabled  # Conditional during migration
```

**After cleanup**:
```yaml
dependencies:
  - name: mozcloud
    version: "0.7.x"
    repository: "oci://us-west1-docker.pkg.dev/moz-fx-platform-artifacts/mozcloud-charts"
    # No condition - mozcloud always enabled
```

**Process**:
1. Remove `condition: mozcloud.enabled` from Chart.yaml
2. Remove `mozcloud.enabled: true` from all values files (no longer needed)
3. Run `helm dependency update` to regenerate Chart.lock
4. Validate with render-diff

### 4. Migration Directory Cleanup

**Options**:

**Option A: Archive** (Recommended for first 1-3 months)
```bash
mv .migration .migration-archive
# Keep in git for reference
```

**Option B: Remove Completely**
```bash
rm -rf .migration
# Migration docs preserved in git history
```

**What's in .migration/**:
- Migration plans and documentation
- Original and migrated manifests for comparison
- Diff analyses
- Change logs
- Downloaded mozcloud chart reference

**Recommendation**: Archive initially, remove after team confirms no longer needed

## Validation Strategy

After each cleanup step, run comprehensive validation:

### 1. Render-diff for All Environments
```bash
render-diff -f values-dev.yaml -su
render-diff -f values-stage.yaml -su
render-diff -f values-prod.yaml -su
render-diff -f values-preview.yaml -su
```

**Expected**: Zero differences (semantic equivalence)

### 2. Resource Count Check
```bash
helm template . -f values.yaml -f values-dev.yaml | grep "^kind:" | wc -l
helm template . -f values.yaml -f values-stage.yaml | grep "^kind:" | wc -l
helm template . -f values.yaml -f values-prod.yaml | grep "^kind:" | wc -l
```

**Expected**: Same resource counts as before cleanup

### 3. Visual Inspection
```bash
helm template . -f values.yaml -f values-dev.yaml > /tmp/dev-post-cleanup.yaml
diff .migration/manifests/dev/migrated.yaml /tmp/dev-post-cleanup.yaml
```

**Expected**: No differences in actual resource definitions

## Cleanup Branch Strategy

Create a cleanup branch for review:
```bash
git checkout -b cleanup-mozcloud-migration-$(date +%Y%m%d)
```

This allows:
- Team review before applying
- Easy rollback if issues found
- Clear separation from environment migrations

## Success Criteria

- [ ] Values consolidated (no unnecessary duplication)
- [ ] Custom templates removed (only mozcloud or custom resources remain)
- [ ] Chart.yaml simplified (no conditional mozcloud enablement)
- [ ] All environments render identical manifests post-cleanup
- [ ] render-diff shows zero changes for all environments
- [ ] Migration directory archived or removed
- [ ] Cleanup changes committed on separate branch
- [ ] Team review completed
- [ ] Cleanup branch merged to main

## Rollback Plan

If cleanup causes unexpected issues:

1. **Git revert** the cleanup commit
2. **Redeploy** via ArgoCD sync
3. **Review** what went wrong
4. **Fix** and re-attempt

Cleanup is purely cosmetic - rollback should be simple.

## Example: Values Consolidation

### Before Cleanup

`values.yaml`:
```yaml
# Minimal base configuration
```

`values-dev.yaml`:
```yaml
mozcloud:
  enabled: true
  app:
    name: myapp
    realm: nonprod
  workloads:
    myapp:
      replicas: 2
      containers:
        - name: app
          image: myregistry/myapp
          tag: dev-latest
          resources:
            limits:
              cpu: 100m
              memory: 128Mi
```

`values-stage.yaml`:
```yaml
mozcloud:
  enabled: true
  app:
    name: myapp          # Duplicate
    realm: nonprod       # Duplicate
  workloads:
    myapp:
      replicas: 2        # Duplicate
      containers:
        - name: app      # Duplicate
          image: myregistry/myapp  # Duplicate
          tag: stage-latest
          resources:
            limits:
              cpu: 200m
              memory: 128Mi
```

### After Cleanup

`values.yaml`:
```yaml
mozcloud:
  app:
    name: myapp
    realm: nonprod

  workloads:
    myapp:
      replicas: 2
      containers:
        - name: app
          image: myregistry/myapp
          # tag: defined per environment
          # resources: defined per environment
```

`values-dev.yaml`:
```yaml
mozcloud:
  workloads:
    myapp:
      containers:
        - name: app
          tag: dev-latest
          resources:
            limits:
              cpu: 100m
              memory: 128Mi
```

`values-stage.yaml`:
```yaml
mozcloud:
  workloads:
    myapp:
      containers:
        - name: app
          tag: stage-latest
          resources:
            limits:
              cpu: 200m
              memory: 128Mi
```

**Result**: Cleaner separation of common vs environment-specific config

## Important Notes

- **Cleanup is optional** - Functionality remains identical with or without cleanup
- **No urgency** - Can defer cleanup if team bandwidth is limited
- **Low risk** - All changes validated with render-diff before applying
- **Makes maintenance easier** - Reduces duplication and simplifies future updates
- **Team decision** - Timing of cleanup is up to the team, not prescribed

## Troubleshooting

**Issue**: render-diff shows differences after values consolidation

**Solution**: Review the diff carefully - ensure environment-specific overrides are preserved in the right files

---

**Issue**: Templates removed but resources missing

**Solution**: Verify mozcloud provides the resource type - may need to keep custom template

---

**Issue**: Chart.lock conflicts after Chart.yaml changes

**Solution**: Run `helm dependency update` to regenerate Chart.lock

---

**Issue**: Team wants to keep templates for future use

**Solution**: Archive templates in a separate directory (e.g., `templates-archive/`) instead of deleting
