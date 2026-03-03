# Preview Environment Migration Guide

This guide explains how preview environment migrations differ from dev/stage migrations.

## Overview

Preview environments require **DIFFERENT** resource naming patterns than dev/stage environments because multiple PR deployments must coexist without conflicts.

## Resource Naming Patterns

| Aspect | Dev/Stage | Preview | Rationale |
|--------|-----------|---------|-----------|
| **Resource Names** | Preserve original | Prefix with PR# | Preview needs isolation between PRs |
| **Target Secret** | Preserve if possible | Prefix with PR# | Each PR needs own secrets |
| **ServiceAccount** | Preserve if possible | Prefix with PR# | Each PR needs own ServiceAccount |
| **ConfigMaps** | Preserve names | Prefix with PR# | Isolation between PR deployments |
| **Hostname** | Static domain | Dynamic PR hostname | ArgoCD ApplicationSet managed |
| **Strategy** | Minimize changes | Enable isolation | Different goals |

## Dev/Stage Pattern: Preserve Original Names

For dev/stage environments, **preserve resource names whenever possible**:

```yaml
# Original custom chart
Deployment: cicd-demos
ExternalSecret: external-secret-cicd-demos → cicd-demos-secrets (rename for mozcloud pattern)
ServiceAccount: cicd-demos

# After mozcloud migration
Deployment: cicd-demos  (preserved)
ExternalSecret: cicd-demos-secrets  (minimal change - target Secret name unchanged)
ServiceAccount: cicd-demos  (preserved)
```

**Goal**: Minimize resource name changes to avoid service disruption.

## Preview Pattern: Prefix All Resources with PR Number

For preview environments, **all resources are prefixed** with the PR number for isolation:

```yaml
# Original custom chart (preview)
Deployment: cicd-demos
ExternalSecret: external-secret-cicd-demos
ServiceAccount: cicd-demos
HTTPRoute: cicd-demos-cicd-demos

# After mozcloud migration (with global.preview.pr: "13")
Deployment: pr13-cicd-demos  (PR-prefixed)
ExternalSecret: pr13-cicd-demos-secrets  (PR-prefixed)
Target Secret: pr13-cicd-demos-secrets  (MUST ALSO be prefixed)
ServiceAccount: pr13-cicd-demos  (PR-prefixed)
HTTPRoute: pr13-cicd-demos  (PR-prefixed)
```

**Goal**: Isolate each PR deployment so multiple preview environments can coexist.

**Why Different**:
- **Dev/Stage**: Single deployment per environment - stability and minimal changes matter
- **Preview**: Multiple PR deployments coexist - isolation prevents conflicts

## Preview-Specific Configuration

Preview environments use `global.preview` configuration:

```yaml
global:
  preview:
    pr: "13"  # Placeholder for local testing
    host: "pr13-example.preview.mozilla.cloud"  # Placeholder - ArgoCD overrides

mozcloud:
  enabled: true

  preview:
    enabled: true
    # httpRoute defaults are typically correct:
    # - gateway.name: sandbox-high-preview-gateway
    # - gateway.namespace: preview-shared-infrastructure
    # - endpointCheck.enabled: true
```

**Important Notes**:
- `global.preview.pr` and `global.preview.host` are **placeholders for local testing**
- ArgoCD ApplicationSet **overrides these values** at deployment time with actual PR number and hostname
- Mozcloud automatically prefixes all resources with the PR number from `global.preview.pr`

## Critical Validation Points

When migrating preview environments, verify:

1. **All resources are PR-prefixed**
   ```bash
   helm template . -f values-preview.yaml | grep "name: pr.*-"
   ```

2. **Target Secret name matches pod reference**
   - ExternalSecret target: `pr13-cicd-demos-secrets`
   - Pods secretRef: `pr13-cicd-demos-secrets`
   - Both must match with PR prefix

3. **ServiceAccount name matches pod reference**
   - ServiceAccount: `pr13-cicd-demos`
   - Deployment serviceAccountName: `pr13-cicd-demos`
   - Both must match with PR prefix

## When to Approve Name Changes

**Dev/Stage Migrations**:
- Approve only if technically required (rare)
- Question any name changes - they should be minimal
- Goal: Preserve original names

**Preview Migrations**:
- Always expect PR prefixing (normal and required)
- Verify target Secret name is ALSO prefixed
- Verify ServiceAccount is ALSO prefixed
- Goal: Enable isolation

## Preview HTTPRoute Configuration

Preview environments typically use HTTPRoute with a shared gateway:

```yaml
preview:
  enabled: true
  httpRoute:
    enabled: true
    gateway:
      name: sandbox-high-preview-gateway
      namespace: preview-shared-infrastructure
  endpointCheck:
    enabled: true
    checkPath: "__heartbeat__"
```

**Default gateway settings** are typically correct - no override needed unless your infrastructure uses different gateway names.

## Common Preview Issues

### Secret Reference Mismatch
**Symptom**: Pods can't find secret - reference `cicd-demos-secrets` but secret is `pr13-cicd-demos-secrets`

**Cause**: Mozcloud version issue or misconfiguration

**Solution**: Verify mozcloud is correctly prefixing the target Secret name and pod references

### Resource Conflicts Between PRs
**Symptom**: PR deployments interfere with each other

**Cause**: Resources not properly prefixed

**Solution**: Verify all resources have PR prefix by checking `global.preview.pr` is set and mozcloud is applying it
