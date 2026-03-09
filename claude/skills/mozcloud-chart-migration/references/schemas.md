# Migration Documentation Schemas

This document defines the expected structure and format for all migration documentation files in the `.migration/` directory. Following these schemas ensures consistency across migrations and makes it easy for Claude to generate and update documentation correctly.

---

## Table of Contents

1. [README.md Schema](#readmemd-schema)
2. [STATUS.md Schema](#statusmd-schema)
3. [MIGRATION_PLAN_$ENV.md Schema](#migration_plan_envmd-schema)
4. [CHANGES_$ENV.md Schema](#changes_envmd-schema)
5. [DIFF_ANALYSIS_$ENV.md Schema](#diff_analysis_envmd-schema)

---

## README.md Schema

**Purpose**: Entry point for understanding the current state of the migration. Always read this file first when resuming work.

**Location**: `.migration/README.md`

**Template**:
```markdown
# Migration Status: <CHART_NAME>

## Current State
- **Branch**: <branch-name>
- **Environment**: <current-env>
- **Status**: <brief status description>
- **Started**: <YYYY-MM-DD>
- **Last Updated**: <YYYY-MM-DD>

## What's Been Done
- [OK] Setup complete (branch created, .migration/ directory initialized)
- [OK] Original manifests captured (.migration/manifests/dev/original.yaml)
- [OK] Migration plan created and approved
- [ ] Values files converted to mozcloud format
- [ ] Chart.yaml updated with mozcloud dependency
- [ ] Templates gated for non-migrated environments
- [ ] Validation tests passed

## Current Issues / Blockers
- <issue-1>: <description>
- <issue-2>: <description>

## Next Steps
1. <next-action-1>
2. <next-action-2>
3. <next-action-3>

## Important Decisions Made
- **Resource Naming**: <decision about resource names>
- **Template Handling**: <decision about which templates to keep/remove>
- **Configuration Approach**: <decision about static vs dynamic configs>

## Notes
- <Any additional context, warnings, or information>
- <Links to relevant documentation>
```

**Example**:
```markdown
# Migration Status: fxa-profile-server

## Current State
- **Branch**: claude-migration-fxa-profile-server-dev
- **Environment**: dev
- **Status**: Migration in progress, values files converted, awaiting validation
- **Started**: 2024-03-06
- **Last Updated**: 2024-03-06

## What's Been Done
- [OK] Setup complete (branch created, .migration/ directory initialized)
- [OK] Original manifests captured (.migration/manifests/dev/original.yaml)
- [OK] Migration plan created and approved
- [OK] Values files converted to mozcloud format
- [OK] Chart.yaml updated with mozcloud dependency
- [OK] Templates gated for non-migrated environments
- [ ] Validation tests passed

## Current Issues / Blockers
None currently

## Next Steps
1. Run render-diff validation: `./scripts/validate_migration.sh dev --all-environments`
2. Review DIFF_ANALYSIS_dev.md for any unexpected changes
3. If validation passes, commit changes and create PR

## Important Decisions Made
- **Resource Naming**: All resource names preserved exactly (workload name = deployment name)
- **Template Handling**: All templates gated with `{{ if not .Values.mozcloud.enabled }}` condition
- **Configuration Approach**: Using mozcloud default nginx (no custom nginx config needed)

## Notes
- Mozcloud chart version 1.2.3 downloaded to .migration/mozcloud/
- Original chart had 12 resources, expecting same count after migration
- Non-migrated environments (stage, prod) should show zero changes
```

---

## STATUS.md Schema

**Purpose**: Track progress across all environments in a multi-environment migration.

**Location**: `.migration/STATUS.md`

**Template**:
```markdown
# Migration Progress

Last Updated: <YYYY-MM-DD>

| Environment | Status | Branch | Merged | Notes |
|-------------|--------|--------|--------|-------|
| dev | <status> | <branch-name> | <date or empty> | <notes> |
| stage | <status> | <branch-name> | <date or empty> | <notes> |
| prod | <status> | <branch-name> | <date or empty> | <notes> |
| preview | <status> | <branch-name> | <date or empty> | <notes> |

## Status Values
- **Not Started**: No work begun
- **In Progress**: Migration underway
- **Testing**: Awaiting validation
- **Ready for Review**: PR created
- **Completed**: Merged to main
- **Blocked**: Waiting on dependency

## Current Focus
- **Environment**: <current-env>
- **Branch**: <current-branch>
- **Started**: <YYYY-MM-DD>
- **Expected Completion**: <target-date>

## Migration Strategy
- Order: dev → stage → prod → preview
- Each environment isolated in separate branch
- Testing in lower environments before proceeding to production

## Blockers
- <blocker-1>: <description>

## Notes
- <overall migration context>
```

**Example**:
```markdown
# Migration Progress

Last Updated: 2024-03-06

| Environment | Status | Branch | Merged | Notes |
|-------------|--------|--------|--------|-------|
| dev | Completed | claude-migration-fxa-profile-dev | 2024-03-05 | All tests passed, deployed successfully |
| stage | In Progress | claude-migration-fxa-profile-stage | - | Validation in progress |
| prod | Not Started | - | - | Blocked: awaiting stage completion |
| preview | Not Started | - | - | Will migrate after prod |

## Status Values
- **Not Started**: No work begun
- **In Progress**: Migration underway
- **Testing**: Awaiting validation
- **Ready for Review**: PR created
- **Completed**: Merged to main
- **Blocked**: Waiting on dependency

## Current Focus
- **Environment**: stage
- **Branch**: claude-migration-fxa-profile-stage
- **Started**: 2024-03-06
- **Expected Completion**: 2024-03-07

## Migration Strategy
- Order: dev → stage → prod → preview
- Each environment isolated in separate branch
- Testing in lower environments before proceeding to production
- Stage has regional values file (values-stage-europe-west1.yaml) requiring special handling

## Blockers
None currently

## Notes
- Dev migration took 2 days (including learning curve)
- Stage and prod expected to be faster using patterns from dev
- Preview environment has different naming requirements (see preview-environment-guide.md)
```

---

## MIGRATION_PLAN_$ENV.md Schema

**Purpose**: Detailed plan for migrating a specific environment. Created before implementation begins, serves as roadmap.

**Location**: `.migration/MIGRATION_PLAN_<env>.md` (e.g., `MIGRATION_PLAN_dev.md`)

**Template**:
```markdown
# Migration Plan: <CHART_NAME> - <ENV> Environment

Created: <YYYY-MM-DD>
Environment: <env>
Mozcloud Version: <version>

## Current State Analysis

### Existing Resources
<List all resources currently rendered by the custom chart>

**Resource Count**: <N> resources
**Resource Types**:
- <count> Deployments
- <count> Services
- <count> ConfigMaps
- <count> Secrets/ExternalSecrets
- <count> Ingress/HTTPRoutes
- <count> Other resources

### Current Values Structure
**values.yaml**:
- <key sections and their purpose>

**values-<env>.yaml**:
- <key sections and their purpose>

**Regional Values** (if applicable):
- <regional values file>: <purpose>

### Custom Templates
- `templates/<template-1>.yaml`: <description>
- `templates/<template-2>.yaml`: <description>

### Dependencies
- <dependency-1>: <version>

## Proposed Changes

### 1. Chart.yaml Updates
```yaml
dependencies:
  - name: mozcloud
    version: <version>
    repository: oci://us-west1-docker.pkg.dev/moz-fx-platform-artifacts/mozcloud-charts
    condition: mozcloud.enabled
```

**Rationale**: <why this approach>

### 2. Values File Conversion

#### values.yaml Changes
<Describe changes to shared values file>

**New mozcloud configuration**:
```yaml
mozcloud:
  enabled: false  # Will be enabled per-environment

workloads:
  <workload-name>:
    <configuration>
```

**Existing configuration** (preserved for non-migrated environments):
```yaml
<existing-config>
```

#### values-<env>.yaml Changes
<Describe environment-specific changes>

**Enable mozcloud**:
```yaml
mozcloud:
  enabled: true
```

**Workload configuration**:
```yaml
workloads:
  <workload-name>:
    type: deployment
    replicas: <N>
    image: <image>
    # ... other config
```

### 3. Template Modifications

**Templates to Gate** (disable for migrated environments):
- `templates/<template-1>.yaml`: Wrap with `{{ if not .Values.mozcloud.enabled }}`
- `templates/<template-2>.yaml`: Wrap with `{{ if not .Values.mozcloud.enabled }}`

**Templates to Keep** (cannot be replaced by mozcloud):
- `templates/<special-template>.yaml`: <reason why it's needed>

**Templates to Remove** (fully replaced by mozcloud):
- None at this stage (will remove during cleanup phase)

### 4. Resource Name Mapping

**CRITICAL: Preserve resource names to avoid downtime**

| Original Resource | Type | Original Name | Mozcloud Name | Action |
|-------------------|------|---------------|---------------|--------|
| Deployment | Deployment | `app-worker` | `app-worker` | [OK] Preserved (workload key = `app-worker`) |
| Service | Service | `app-service` | `app-service` | [OK] Preserved (auto-generated by mozcloud) |
| ConfigMap | ConfigMap | `app-config` | `app-config` | [OK] Preserved (using mozcloud.configs) |

**Name Changes** (requires user approval):
- None expected - all names should be preserved

### 5. Configuration Mapping

**Deployment → Workload**:
- Replicas: `<original>` → `workloads.<name>.replicas: <value>`
- Image: `<original>` → `workloads.<name>.image: <value>`
- Env vars: `<original>` → `workloads.<name>.env: <mapping>`
- Resources: `<original>` → `workloads.<name>.resources: <mapping>`

**ConfigMap → mozcloud.configs**:
- `<configmap-name>`: `<approach (static file or inline data)>`

**ExternalSecret → mozcloud.externalSecrets**:
- `<externalsecret-name>`: `<mapping to mozcloud schema>`

**Ingress → HTTPRoute**:
- Gateway: `<gateway-name>`
- Hostnames: `<hostnames>`
- Routes: `<route mappings>`

## Testing Strategy

### Pre-Migration
1. Capture original manifests: `./scripts/capture_manifests.sh <env>`
2. Document resource count and types

### Post-Migration
1. Run semantic diff: `render-diff -f values-<env>.yaml -su`
2. Validate resource count matches original
3. Verify non-migrated environments unchanged: `render-diff -f values-<other-env>.yaml`
4. Review manifest diff manually
5. Check for unintended name changes

### Success Criteria
- [OK] Resource count matches or exceeds original
- [OK] All resource names preserved (or approved changes documented)
- [OK] render-diff shows semantic equivalence
- [OK] Non-migrated environments show zero changes
- [OK] No unexpected configuration changes

## Risks and Mitigations

### Risk: Resource Names Change
- **Mitigation**: Use full deployment name as workload key
- **Verification**: Compare resource names before and after

### Risk: Configuration Drift
- **Mitigation**: Carefully map all configuration fields
- **Verification**: Manual manifest diff review

### Risk: Breaking Non-Migrated Environments
- **Mitigation**: Gate all templates with `mozcloud.enabled` condition
- **Verification**: render-diff for each non-migrated environment

## Rollback Plan
If issues arise:
1. Delete migration branch: `git branch -D <branch-name>`
2. Or push fixes to same branch (no complex rollback needed with ArgoCD)
3. Non-merged changes don't affect deployed environments

## Timeline
- **Planning**: <date range>
- **Implementation**: <date range>
- **Testing**: <date range>
- **Review**: <date range>
- **Deployment**: <target date>

## Notes
- <Any additional context>
- <Special considerations>
- <Questions for user>
```

---

## CHANGES_$ENV.md Schema

**Purpose**: Detailed change log documenting all modifications made during migration. Updated as changes are implemented.

**Location**: `.migration/CHANGES_<env>.md` (e.g., `CHANGES_dev.md`)

**Template**:
```markdown
# Migration Changes: <CHART_NAME> - <ENV> Environment

Environment: <env>
Migration Branch: <branch-name>
Date: <YYYY-MM-DD>

## Summary
<Brief overview of changes made>

## Chart.yaml Changes

### Added Dependencies
```yaml
dependencies:
  - name: mozcloud
    version: <version>
    repository: oci://us-west1-docker.pkg.dev/moz-fx-platform-artifacts/mozcloud-charts
    condition: mozcloud.enabled
```

## Values File Changes

### values.yaml
<List changes to shared values file>

**Added**:
```yaml
<new configuration>
```

**Modified**:
```yaml
# Before
<old config>

# After
<new config>
```

**Removed**:
```yaml
<removed config>
```

### values-<env>.yaml
<List changes to environment-specific values file>

**Added**:
```yaml
mozcloud:
  enabled: true

workloads:
  <workload-config>
```

## Template Changes

### Templates Gated
<List templates wrapped with conditions>

**templates/<template-1>.yaml**:
- Added condition: `{{ if not .Values.mozcloud.enabled }}`
- Reason: Replaced by mozcloud workload

**templates/<template-2>.yaml**:
- Added condition: `{{ if not .Values.mozcloud.enabled }}`
- Reason: Replaced by mozcloud service

### Templates Kept
<List templates not modified>

**templates/<special-template>.yaml**:
- Reason: Cannot be replaced by mozcloud (custom resource)

### Templates Removed
None (cleanup phase will remove gated templates after all environments migrated)

## Resource Name Changes

**All resource names preserved** [OK]

| Resource Type | Original Name | New Name | Status |
|---------------|---------------|----------|--------|
| Deployment | `app-worker` | `app-worker` | [OK] Preserved |
| Service | `app-service` | `app-service` | [OK] Preserved |
| ConfigMap | `app-config` | `app-config` | [OK] Preserved |

**Name Changes** (if any, with user approval):
- None

## Configuration Mapping Details

### Workload: <workload-name>
```yaml
# Original deployment spec
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: app
        image: us-west1-docker.pkg.dev/project/repo/app:v1.2.3
        env:
        - name: ENV_VAR
          value: "value"

# Mozcloud workload config
workloads:
  app-worker:
    type: deployment
    replicas: 3
    image: us-west1-docker.pkg.dev/project/repo/app:v1.2.3
    env:
      ENV_VAR: "value"
```

### ConfigMap: <configmap-name>
**Approach**: <Static file / Inline data>

```yaml
# Original
<original configmap yaml>

# Mozcloud config
<mozcloud config yaml>
```

### ExternalSecret: <externalsecret-name>
```yaml
# Original
<original externalsecret yaml>

# Mozcloud config
<mozcloud externalsecret yaml>
```

### HTTPRoute/Ingress: <route-name>
```yaml
# Original Ingress
<original ingress yaml>

# Mozcloud HTTPRoute config
<mozcloud httproute config>
```

## Special Handling

### Preview Environment Considerations
<If preview environment, note special handling>

### Regional Values
<If multi-region, note regional handling>

### Custom Nginx Configuration
<If custom nginx, note approach>

## Issues Encountered and Solutions

### Issue 1: <description>
- **Problem**: <what went wrong>
- **Solution**: <how it was fixed>
- **Prevention**: <how to avoid in future>

## Validation Results

### render-diff Output
```
<output from render-diff command>
```

**Result**: [OK] Semantic equivalence confirmed / [WARNING] Differences detected

### Resource Count
- Original: <N> resources
- Migrated: <M> resources
- Status: [OK] Match / [WARNING] Difference

### Non-Migrated Environments
- **stage**: [OK] No changes
- **prod**: [OK] No changes

## Files Modified
- `Chart.yaml`
- `values.yaml`
- `values-<env>.yaml`
- `templates/<template-1>.yaml`
- `templates/<template-2>.yaml`

## Next Steps for Other Environments
<Lessons learned and recommendations for next environment migration>

- Pattern established for <something>
- Watch out for <gotcha>
- Reuse <approach> for similar resources
```

---

## DIFF_ANALYSIS_$ENV.md Schema

**Purpose**: Structured analysis of differences between original and migrated manifests. Generated automatically during validation.

**Location**: `.migration/DIFF_ANALYSIS_<env>.md` (e.g., `DIFF_ANALYSIS_dev.md`)

**Template**:
```markdown
# Diff Analysis: <ENV> Environment

Generated: <YYYY-MM-DD HH:MM:SS>
Chart: <chart-name>
Environment: <env>
Mozcloud Version: <version>

## Resource Impact Summary
- **Original count**: <N> resources
- **Migrated count**: <M> resources
- **Added**: <X> resources
- **Modified**: <Y> resources (semantically equivalent: <Z>, changed: <W>)
- **Deleted**: <0> resources
- **Unchanged**: <U> resources

## Added Resources
<List resources that were added, if any>

| Resource | Type | Name | Reason |
|----------|------|------|--------|
| <resource-1> | <type> | <name> | Auto-generated by mozcloud |

## Modified Resources

### Semantically Equivalent (Expected Changes)
<List resources that changed but are semantically equivalent>

| Resource | Type | Name | Changes |
|----------|------|------|---------|
| <resource-1> | <type> | <name> | Label additions, annotation changes |

**Details**:
- **Labels**: mozcloud adds standard labels (app.kubernetes.io/*)
- **Annotations**: mozcloud adds management annotations
- **Field order**: YAML field ordering may differ
- **Defaults**: Explicit default values may be added

### Semantic Changes (Require Review)
<List resources with actual semantic changes>

| Resource | Type | Name | Changes | Impact |
|----------|------|------|---------|--------|
| <resource-1> | <type> | <name> | <change-description> | <impact-description> |

## Deleted Resources
<Should always be empty - flag if any resources were deleted>

**Status**: [OK] No resources deleted / [WARNING] ALERT: Resources deleted!

## Unchanged Resources
<List resources that didn't change at all>

| Resource | Type | Name |
|----------|------|------|
| <resource-1> | <type> | <name> |

## Changes That May Trigger Pod Restarts
<List changes that could cause deployments to roll>

| Deployment | Change | Will Restart? |
|------------|--------|---------------|
| <deployment-1> | Environment variable added | Yes |
| <deployment-2> | Label added | No (metadata only) |
| <deployment-3> | Resource limits changed | Yes |

## Stable Resources (No Changes Expected)
<List resources that should remain stable>

- [OK] ConfigMap "app-config": No changes
- [OK] Secret "app-secret": No changes
- [OK] Service "app-service": Labels added only (semantically equivalent)

## Critical Changes Requiring Review
<Flag any unexpected or potentially problematic changes>

### Change 1: <description>
- **Resource**: <resource-name>
- **Change**: <what changed>
- **Impact**: <potential impact>
- **Action**: <recommended action>

## Resource Breakdown

### Original Resources
```
<count> Deployment
<count> Service
<count> ConfigMap
<count> ExternalSecret
<count> HTTPRoute
<count> Certificate
<count> <other-types>
---
Total: <N> resources
```

### Migrated Resources
```
<count> Deployment
<count> Service
<count> ConfigMap
<count> ExternalSecret
<count> HTTPRoute
<count> Certificate
<count> <other-types>
---
Total: <M> resources
```

## Manual Review Checklist
- [ ] Review all "Modified" resources for expected changes
- [ ] Verify all "Added" resources are intentional
- [ ] Confirm no "Deleted" resources (should be 0)
- [ ] Check pod restart triggers align with expectations
- [ ] Validate resource names preserved
- [ ] Review critical changes flagged above

## Recommendation
<Overall assessment: Ready to deploy / Needs fixes / Requires further review>

## Next Steps
1. <action-1>
2. <action-2>
3. <action-3>
```

**Example**:
```markdown
# Diff Analysis: dev Environment

Generated: 2024-03-06 14:30:22
Chart: fxa-profile-server
Environment: dev
Mozcloud Version: 1.2.3

## Resource Impact Summary
- **Original count**: 12 resources
- **Migrated count**: 13 resources
- **Added**: 1 resource (ExternalSecret auto-generated)
- **Modified**: 8 resources (semantically equivalent: 8, changed: 0)
- **Deleted**: 0 resources
- **Unchanged**: 4 resources

## Added Resources

| Resource | Type | Name | Reason |
|----------|------|------|--------|
| fxa-profile-secrets | ExternalSecret | fxa-profile-secrets | Auto-generated by mozcloud from workload.secrets config |

## Modified Resources

### Semantically Equivalent (Expected Changes)

| Resource | Type | Name | Changes |
|----------|------|------|---------|
| fxa-profile-worker | Deployment | fxa-profile-worker | Labels added (app.kubernetes.io/*) |
| fxa-profile-service | Service | fxa-profile-service | Labels added, selector updated |
| fxa-profile-config | ConfigMap | fxa-profile-config | Labels added |
| fxa-profile-route | HTTPRoute | fxa-profile-route | Annotations added |

**Details**:
- **Labels**: mozcloud adds standard labels (app.kubernetes.io/name, app.kubernetes.io/managed-by)
- **Annotations**: mozcloud adds management annotations
- **Field order**: YAML field ordering differs but content is equivalent
- **Defaults**: Some explicit default values added (e.g., restartPolicy: Always)

### Semantic Changes (Require Review)
None - all changes are label/annotation additions only.

## Deleted Resources

**Status**: [OK] No resources deleted

## Unchanged Resources

| Resource | Type | Name |
|----------|------|------|
| fxa-profile-cert | Certificate | fxa-profile-cert |
| fxa-profile-backup-config | ConfigMap | fxa-profile-backup-config |
| fxa-profile-monitoring | ServiceMonitor | fxa-profile-monitoring |
| fxa-profile-pdb | PodDisruptionBudget | fxa-profile-pdb |

## Changes That May Trigger Pod Restarts

| Deployment | Change | Will Restart? |
|------------|--------|---------------|
| fxa-profile-worker | Labels added to pod template | Yes (pod template changed) |

**Note**: Label changes to pod templates will trigger a rolling update. This is expected and safe.

## Stable Resources (No Changes Expected)

- [OK] ConfigMap "fxa-profile-config": Only labels added (semantically equivalent)
- [OK] Service "fxa-profile-service": Only labels added (semantically equivalent)
- [OK] HTTPRoute "fxa-profile-route": Only annotations added (semantically equivalent)

## Critical Changes Requiring Review

None - all changes are expected mozcloud additions (labels, annotations, auto-generated ExternalSecret).

## Resource Breakdown

### Original Resources
```
1 Deployment
1 Service
2 ConfigMap
1 HTTPRoute
1 Certificate
1 ServiceMonitor
1 PodDisruptionBudget
4 <other-custom-resources>
---
Total: 12 resources
```

### Migrated Resources
```
1 Deployment
1 Service
2 ConfigMap
1 ExternalSecret (NEW)
1 HTTPRoute
1 Certificate
1 ServiceMonitor
1 PodDisruptionBudget
4 <other-custom-resources>
---
Total: 13 resources
```

## Manual Review Checklist
- [x] Review all "Modified" resources for expected changes
- [x] Verify all "Added" resources are intentional
- [x] Confirm no "Deleted" resources (should be 0)
- [x] Check pod restart triggers align with expectations
- [x] Validate resource names preserved
- [x] Review critical changes flagged above

## Recommendation
[OK] **Ready to deploy**

The migration is successful. All changes are expected mozcloud additions (standard labels, annotations, and auto-generated ExternalSecret). Resource names are preserved. Resource count increased by 1 due to mozcloud generating ExternalSecret from workload secrets configuration.

## Next Steps
1. Commit changes to migration branch
2. Create pull request for review
3. Merge to main after approval
4. Monitor deployment in dev environment
5. Proceed with stage environment migration
```

---

## Usage Guidelines

### When Creating New Documentation
1. **Always start with README.md** - This is the entry point
2. **Use STATUS.md for multi-environment tracking** - Update after each environment milestone
3. **Create MIGRATION_PLAN_$ENV.md before implementing** - Get user approval on plan
4. **Update CHANGES_$ENV.md as you go** - Document changes immediately
5. **Generate DIFF_ANALYSIS_$ENV.md automatically** - Use validation scripts

### When Resuming Work
1. **Read README.md first** - Understand current state
2. **Check STATUS.md** - See where you are in multi-environment progression
3. **Review MIGRATION_PLAN_$ENV.md** - Understand the approved approach
4. **Check CHANGES_$ENV.md** - See what's already been done
5. **Use DIFF_ANALYSIS_$ENV.md** - Validate changes are expected

### Consistency Rules
- **Dates**: Always use YYYY-MM-DD format
- **Status indicators**: Use [OK], [FAIL], [WARNING] consistently
- **Code blocks**: Always specify language (```yaml, ```bash, etc.)
- **Resource names**: Always use backticks for resource names
- **Tables**: Use tables for structured data (resource mappings, status tracking)

### File Naming Conventions
- Use `$ENV` placeholder in schemas: `MIGRATION_PLAN_$ENV.md`
- Actual files use environment name: `MIGRATION_PLAN_dev.md`
- Always lowercase environment names in filenames
- Use underscores in filenames: `DIFF_ANALYSIS_dev.md` (not `DIFF-ANALYSIS-dev.md`)

---

## Schema Validation

Before considering migration documentation complete, verify:

- [ ] README.md exists and follows schema
- [ ] STATUS.md exists (for multi-environment migrations)
- [ ] MIGRATION_PLAN_$ENV.md exists for current environment
- [ ] CHANGES_$ENV.md documents all changes made
- [ ] DIFF_ANALYSIS_$ENV.md generated and reviewed
- [ ] All files use consistent formatting and structure
- [ ] Dates are in YYYY-MM-DD format
- [ ] Status indicators ([OK][FAIL][WARNING]) used appropriately
- [ ] Code blocks have language specified
- [ ] Tables are properly formatted
