# Migration Directory Structure

The `.migration` directory maintains all migration-related documentation and artifacts during the migration process.

## Directory Layout

```
.migration/
├── README.md                    # Current status, next steps, decisions (ENTRY POINT)
├── STATUS.md                    # Multi-environment progress tracker
├── SUMMARY.md                   # High-level migration overview (optional)
├── MIGRATION_PLAN_$ENV.md       # Detailed plan for each environment
├── CHANGES_$ENV.md              # Resource name changes per environment
├── DIFF_ANALYSIS_$ENV.md        # Structured diff analysis per environment
├── mozcloud/                    # Reference copy of mozcloud chart
│   ├── Chart.yaml
│   ├── values.yaml
│   ├── values.schema.json
│   └── templates/
└── manifests/                   # Original rendered manifests for comparison
    ├── dev/
    │   ├── original.yaml
    │   └── migrated.yaml
    ├── stage/
    │   ├── original.yaml
    │   └── migrated.yaml
    └── prod/
        ├── original.yaml
        └── migrated.yaml
```

## File Purposes

### README.md (Entry Point)
**Purpose**: Single source of truth for current migration state

**Contents**:
- Current environment being migrated
- Migration branch name
- What's been completed
- What's pending
- Next steps
- Blockers or decisions needed
- Important notes and context

**Example**:
```markdown
# Migration Status

**Current Environment**: dev
**Branch**: claude-migration-my-app-dev
**Started**: 2024-01-15

## Completed
- ✅ Pre-flight checks passed
- ✅ Migration plan created and approved
- ✅ Values files converted to mozcloud format
- ✅ Templates gated for dev environment

## In Progress
- 🔄 Testing with render-diff
- 🔄 Reviewing diff analysis

## Next Steps
1. Address any semantic differences found
2. Get user approval on final changes
3. Update STATUS.md
4. Prepare for stage environment migration

## Important Notes
- Preserved all original deployment names
- Added one new ExternalSecret for database credentials
- No changes to stage/prod environments confirmed

## Blockers
None
```

### STATUS.md (Multi-Environment Tracker)
**Purpose**: Track progress across all environments

**Example**:
```markdown
# Migration Progress

| Environment | Status | Branch | Merged | Notes |
|-------------|--------|--------|--------|-------|
| dev | Completed | claude-migration-my-app-dev | 2024-01-16 | All tests passed |
| stage | In Progress | claude-migration-my-app-stage | - | Started 2024-01-17 |
| stage-europe-west1 | Not Started | - | - | Blocked: waiting for stage |
| prod | Not Started | - | - | Blocked: waiting for stage |

## Current Focus
- **Environment**: stage
- **Branch**: claude-migration-my-app-stage
- **Started**: 2024-01-17
- **Expected Completion**: TBD

## Lessons Learned
- Dev migration revealed nginx version mismatch (resolved)
- Template gating pattern works well
- render-diff caught subtle label differences
```

### MIGRATION_PLAN_$ENV.md (Per-Environment Plan)
**Purpose**: Detailed implementation plan for specific environment

**Example**:
```markdown
# Migration Plan: dev Environment

## Current State Analysis

### Existing Resources
- 1 Deployment: `my-app`
- 1 Service: `my-app-service`
- 1 ConfigMap: `my-app-config`
- 1 ExternalSecret: `my-app-db-secret`
- 1 HTTPRoute: `my-app-route`

### Dependencies
- Chart.yaml lists: common-0.1.0 (will be replaced)

## Proposed Changes

### Chart.yaml
- Add mozcloud dependency with condition
- Remove common chart dependency
- Version: keep current chart version

### values.yaml
- Add `mozcloud.enabled: false` as default
- Keep all existing values for non-migrated envs

### values-dev.yaml
- Set `mozcloud.enabled: true`
- Convert deployment config to `mozcloud.workloads.my-app`
- Convert service config to `mozcloud.services.my-app-service`
- Convert config to `mozcloud.configMaps.my-app-config`
- Convert secret to `mozcloud.externalSecrets.my-app-db-secret`
- Convert route to `mozcloud.httpRoutes.my-app-route`

### Templates
- Wrap deployment.yaml in `{{- if not .Values.mozcloud.enabled }}`
- Wrap service.yaml in `{{- if not .Values.mozcloud.enabled }}`
- Wrap configmap.yaml in `{{- if not .Values.mozcloud.enabled }}`
- Wrap externalsecret.yaml in `{{- if not .Values.mozcloud.enabled }}`
- Wrap httproute.yaml in `{{- if not .Values.mozcloud.enabled }}`

## Resource Name Mapping
All names preserved - no changes required:
- Deployment: `my-app` → `workloads.my-app`
- Service: `my-app-service` → `services.my-app-service`
- ConfigMap: `my-app-config` → `configMaps.my-app-config`
- ExternalSecret: `my-app-db-secret` → `externalSecrets.my-app-db-secret`
- HTTPRoute: `my-app-route` → `httpRoutes.my-app-route`

## Testing Strategy
1. render-diff for dev environment
2. render-diff for stage/prod (should show no changes)
3. Manual review of generated manifests
4. Validate resource count matches

## Risks
- Low risk: straightforward migration
- All resource names preserved
- Templates properly gated
```

### CHANGES_$ENV.md (Change Log)
**Purpose**: Detailed log of all changes made

**Example**:
```markdown
# Changes Log: dev Environment

## Chart.yaml
- Added mozcloud dependency (version 1.2.3)
- Added condition: `mozcloud.enabled`
- Removed common chart dependency

## values.yaml
- Added `mozcloud.enabled: false` (default off)

## values-dev.yaml
- Set `mozcloud.enabled: true`
- Added complete mozcloud configuration:
  - workloads.my-app (Deployment)
  - services.my-app-service
  - configMaps.my-app-config
  - externalSecrets.my-app-db-secret
  - httpRoutes.my-app-route

## Templates Modified
- templates/deployment.yaml: wrapped in mozcloud.enabled check
- templates/service.yaml: wrapped in mozcloud.enabled check
- templates/configmap.yaml: wrapped in mozcloud.enabled check
- templates/externalsecret.yaml: wrapped in mozcloud.enabled check
- templates/httproute.yaml: wrapped in mozcloud.enabled check

## Resource Name Changes
None - all original names preserved.

## New Resources
None - 1:1 mapping from original.

## Deleted Resources
None - templates preserved for other environments.

## Configuration Changes
- Image tag updated from 1.0.0 to 1.0.1 (intentional)
- Added new environment variable: DATABASE_POOL_SIZE=10
```

### DIFF_ANALYSIS_$ENV.md (Diff Analysis)
**Purpose**: Structured analysis of differences between original and migrated

**Example**:
```markdown
# Diff Analysis: dev Environment

## Resource Impact Summary
- **Added**: 0 resources
- **Modified**: 1 resource
- **Deleted**: 0 resources
- **Unchanged**: 4 resources

## Modified Resources

### Deployment "my-app"
**Changes**:
- Environment variable added: `DATABASE_POOL_SIZE=10`
- Image tag changed: `1.0.0` → `1.0.1`

**Impact**: Will trigger pod restart

**Reason**: Intentional updates from user

## Unchanged Resources
- Service "my-app-service" - No changes
- ConfigMap "my-app-config" - No changes
- ExternalSecret "my-app-db-secret" - No changes
- HTTPRoute "my-app-route" - No changes

## Changes That May Trigger Pod Restarts
1. Deployment "my-app":
   - Environment variable addition
   - Image tag change

## Critical Changes Requiring Review
None - all changes are intentional and expected.

## Semantic Differences
None detected by render-diff.

## Validation Results
✅ Resource count: 5 original → 5 migrated
✅ Resource names: All preserved
✅ Non-migrated environments: No changes detected
✅ render-diff: Semantic equivalence confirmed
```

## Best Practices

### Always Start with README.md
When resuming work or understanding current state, read `.migration/README.md` first.

### Keep Documentation Current
Update files after each major milestone:
- Complete a phase → Update README.md
- Finish environment → Update STATUS.md
- Make changes → Update CHANGES_$ENV.md
- Review diffs → Update DIFF_ANALYSIS_$ENV.md

### Use for Handoffs
When pausing work or handing off to another engineer:
1. Update README.md with current status
2. Document blockers and decisions needed
3. List clear next steps
4. Note any important context

### Clean Up After Completion
After ALL environments are migrated:
- Archive `.migration/` directory or delete it
- Key documentation can be moved to repo wiki/docs if needed
- Manifest comparisons no longer needed once all envs are live

## Storage Location
The `.migration/` directory should be in the chart root:
```
charts/my-app/
├── .migration/          # ← Here
├── Chart.yaml
├── values.yaml
├── values-dev.yaml
├── templates/
└── ...
```

Use `$CHART_DIR/.migration/` to ensure correct location.
