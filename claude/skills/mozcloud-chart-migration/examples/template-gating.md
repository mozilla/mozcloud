# Template Gating Patterns

This document explains how to properly gate custom Helm templates during migration so that:
- Migrated environments use the mozcloud chart
- Non-migrated environments continue using custom templates
- No disruption to non-migrated environments

---

## Why Template Gating?

During migration, we want to migrate one environment at a time (e.g., dev → stage → prod). Template gating allows us to:

1. **Enable mozcloud selectively**: Only migrated environments use the mozcloud chart
2. **Preserve existing behavior**: Non-migrated environments keep working with custom templates
3. **Gradual migration**: Reduce risk by isolating changes per environment
4. **Easy rollback**: If issues arise, simply disable mozcloud for that environment

---

## Basic Template Gating Pattern

The fundamental pattern is to wrap custom templates with a condition that checks if mozcloud is enabled:

```yaml
{{- if not .Values.mozcloud.enabled }}
# ... existing template content ...
{{- end }}
```

This ensures the template is only rendered when `mozcloud.enabled: false` (non-migrated environments).

---

## Example 1: Simple Deployment

### Before Migration

**templates/deployment.yaml**:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "app.fullname" . }}
  labels:
    {{- include "app.labels" . | nindent 4 }}
spec:
  replicas: {{ .Values.replicaCount }}
  selector:
    matchLabels:
      {{- include "app.selectorLabels" . | nindent 6 }}
  template:
    metadata:
      labels:
        {{- include "app.selectorLabels" . | nindent 8 }}
    spec:
      containers:
      - name: {{ .Chart.Name }}
        image: "{{ .Values.image.repository }}:{{ .Values.image.tag }}"
        ports:
        - name: http
          containerPort: 8080
        resources:
          {{- toYaml .Values.resources | nindent 12 }}
```

### After Migration (Gated)

**templates/deployment.yaml**:
```yaml
{{- if not .Values.mozcloud.enabled }}
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "app.fullname" . }}
  labels:
    {{- include "app.labels" . | nindent 4 }}
spec:
  replicas: {{ .Values.replicaCount }}
  selector:
    matchLabels:
      {{- include "app.selectorLabels" . | nindent 6 }}
  template:
    metadata:
      labels:
        {{- include "app.selectorLabels" . | nindent 8 }}
    spec:
      containers:
      - name: {{ .Chart.Name }}
        image: "{{ .Values.image.repository }}:{{ .Values.image.tag }}"
        ports:
        - name: http
          containerPort: 8080
        resources:
          {{- toYaml .Values.resources | nindent 12 }}
{{- end }}
```

**What Changed**:
- Added `{{- if not .Values.mozcloud.enabled }}` at the top
- Added `{{- end }}` at the bottom
- No changes to template logic itself

**How It Works**:
- When `mozcloud.enabled: false` (non-migrated env): Template renders
- When `mozcloud.enabled: true` (migrated env): Template is skipped, mozcloud creates Deployment

---

## Example 2: Service

### Before Migration

**templates/service.yaml**:
```yaml
apiVersion: v1
kind: Service
metadata:
  name: {{ include "app.fullname" . }}
  labels:
    {{- include "app.labels" . | nindent 4 }}
spec:
  type: {{ .Values.service.type }}
  ports:
  - port: {{ .Values.service.port }}
    targetPort: http
    protocol: TCP
    name: http
  selector:
    {{- include "app.selectorLabels" . | nindent 4 }}
```

### After Migration (Gated)

**templates/service.yaml**:
```yaml
{{- if not .Values.mozcloud.enabled }}
apiVersion: v1
kind: Service
metadata:
  name: {{ include "app.fullname" . }}
  labels:
    {{- include "app.labels" . | nindent 4 }}
spec:
  type: {{ .Values.service.type }}
  ports:
  - port: {{ .Values.service.port }}
    targetPort: http
    protocol: TCP
    name: http
  selector:
    {{- include "app.selectorLabels" . | nindent 4 }}
{{- end }}
```

---

## Example 3: ConfigMap (Fully Replaced by Mozcloud)

### Before Migration

**templates/configmap.yaml**:
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ include "app.fullname" . }}-config
  labels:
    {{- include "app.labels" . | nindent 4 }}
data:
  config.json: |
    {
      "port": {{ .Values.service.port }},
      "env": "{{ .Values.environment }}"
    }
```

### After Migration (Gated)

**templates/configmap.yaml**:
```yaml
{{- if not .Values.mozcloud.enabled }}
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ include "app.fullname" . }}-config
  labels:
    {{- include "app.labels" . | nindent 4 }}
data:
  config.json: |
    {
      "port": {{ .Values.service.port }},
      "env": "{{ .Values.environment }}"
    }
{{- end }}
```

**Mozcloud Replacement** (in values.yaml):
```yaml
configs:
  app-config:
    data:
      config.json: |
        {
          "port": 8080,
          "env": "production"
        }
```

---

## Example 4: Ingress (Replaced by HTTPRoute in Mozcloud)

### Before Migration

**templates/ingress.yaml**:
```yaml
{{- if .Values.ingress.enabled }}
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: {{ include "app.fullname" . }}
  labels:
    {{- include "app.labels" . | nindent 4 }}
  {{- with .Values.ingress.annotations }}
  annotations:
    {{- toYaml . | nindent 4 }}
  {{- end }}
spec:
  ingressClassName: {{ .Values.ingress.className }}
  {{- if .Values.ingress.tls }}
  tls:
  {{- range .Values.ingress.tls }}
  - hosts:
    {{- range .hosts }}
    - {{ . | quote }}
    {{- end }}
    secretName: {{ .secretName }}
  {{- end }}
  {{- end }}
  rules:
  {{- range .Values.ingress.hosts }}
  - host: {{ .host | quote }}
    http:
      paths:
      {{- range .paths }}
      - path: {{ .path }}
        pathType: {{ .pathType }}
        backend:
          service:
            name: {{ include "app.fullname" $ }}
            port:
              number: {{ $.Values.service.port }}
      {{- end }}
  {{- end }}
{{- end }}
```

### After Migration (Gated)

**templates/ingress.yaml**:
```yaml
{{- if and .Values.ingress.enabled (not .Values.mozcloud.enabled) }}
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: {{ include "app.fullname" . }}
  labels:
    {{- include "app.labels" . | nindent 4 }}
  {{- with .Values.ingress.annotations }}
  annotations:
    {{- toYaml . | nindent 4 }}
  {{- end }}
spec:
  ingressClassName: {{ .Values.ingress.className }}
  {{- if .Values.ingress.tls }}
  tls:
  {{- range .Values.ingress.tls }}
  - hosts:
    {{- range .hosts }}
    - {{ . | quote }}
    {{- end }}
    secretName: {{ .secretName }}
  {{- end }}
  {{- end }}
  rules:
  {{- range .Values.ingress.hosts }}
  - host: {{ .host | quote }}
    http:
      paths:
      {{- range .paths }}
      - path: {{ .path }}
        pathType: {{ .pathType }}
        backend:
          service:
            name: {{ include "app.fullname" $ }}
            port:
              number: {{ $.Values.service.port }}
      {{- end }}
  {{- end }}
{{- end }}
```

**What Changed**:
- Changed condition to: `{{- if and .Values.ingress.enabled (not .Values.mozcloud.enabled) }}`
- Preserves original `ingress.enabled` check
- Adds mozcloud check to prevent rendering when mozcloud is enabled

**Mozcloud Replacement** (in values.yaml):
```yaml
httpRoutes:
  app:
    hostnames:
      - app.example.com
    parentRefs:
      - name: internal-gateway
        namespace: gateway-system
    rules:
      - backendRefs:
          - name: app
            port: 8080
```

---

## Example 5: ExternalSecret

### Before Migration

**templates/externalsecret.yaml**:
```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: {{ include "app.fullname" . }}-secrets
  labels:
    {{- include "app.labels" . | nindent 4 }}
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: gcpsm-secret-store
    kind: SecretStore
  target:
    name: {{ include "app.fullname" . }}-secrets
    creationPolicy: Owner
  data:
  {{- range .Values.externalSecrets.data }}
  - secretKey: {{ .name }}
    remoteRef:
      key: {{ .key }}
      {{- if .property }}
      property: {{ .property }}
      {{- end }}
  {{- end }}
```

### After Migration (Gated)

**templates/externalsecret.yaml**:
```yaml
{{- if not .Values.mozcloud.enabled }}
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: {{ include "app.fullname" . }}-secrets
  labels:
    {{- include "app.labels" . | nindent 4 }}
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: gcpsm-secret-store
    kind: SecretStore
  target:
    name: {{ include "app.fullname" . }}-secrets
    creationPolicy: Owner
  data:
  {{- range .Values.externalSecrets.data }}
  - secretKey: {{ .name }}
    remoteRef:
      key: {{ .key }}
      {{- if .property }}
      property: {{ .property }}
      {{- end }}
  {{- end }}
{{- end }}
```

**Mozcloud Replacement** (in values.yaml):
```yaml
externalSecrets:
  app-secrets:
    backendType: gcpSecretsManager
    projectId: my-project
    data:
      - key: my-secret-key
        name: SECRET_VAR
        property: value
```

---

## Example 6: Template That Cannot Be Replaced (Keep Without Gating)

Some resources cannot be replaced by mozcloud and must remain as custom templates:

**templates/custom-resource.yaml**:
```yaml
apiVersion: custom.io/v1
kind: CustomResource
metadata:
  name: {{ include "app.fullname" . }}-custom
spec:
  # Custom resource spec that mozcloud doesn't support
  customField: {{ .Values.custom.field }}
```

**What to Do**:
- **DO NOT gate this template** - it's needed for all environments
- Keep the template as-is
- Document in migration plan why this template remains

---

## Multi-Template Files

Sometimes a single template file contains multiple resources. Gate the entire file:

### Before Migration

**templates/resources.yaml**:
```yaml
---
apiVersion: v1
kind: Service
metadata:
  name: {{ include "app.fullname" . }}
spec:
  type: ClusterIP
  ports:
  - port: 8080
    targetPort: 8080
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ include "app.fullname" . }}-config
data:
  key: value
```

### After Migration (Gated)

**templates/resources.yaml**:
```yaml
{{- if not .Values.mozcloud.enabled }}
---
apiVersion: v1
kind: Service
metadata:
  name: {{ include "app.fullname" . }}
spec:
  type: ClusterIP
  ports:
  - port: 8080
    targetPort: 8080
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ include "app.fullname" . }}-config
data:
  key: value
{{- end }}
```

---

## Partial Template Gating (Advanced)

Sometimes you need to gate only part of a template:

### Example: HPA (Horizontal Pod Autoscaler)

**Scenario**: Mozcloud creates Deployment, but custom chart also creates HPA

**templates/hpa.yaml**:
```yaml
{{- if .Values.autoscaling.enabled }}
{{- if not .Values.mozcloud.enabled }}
# Custom HPA for non-mozcloud environments
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: {{ include "app.fullname" . }}
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: {{ include "app.fullname" . }}
  minReplicas: {{ .Values.autoscaling.minReplicas }}
  maxReplicas: {{ .Values.autoscaling.maxReplicas }}
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: {{ .Values.autoscaling.targetCPUUtilizationPercentage }}
{{- else }}
# Mozcloud HPA configuration (if mozcloud doesn't auto-create HPA)
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: {{ include "app.fullname" . }}
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: {{ .Values.workloads | keys | first }}  # Use mozcloud workload name
  minReplicas: {{ .Values.autoscaling.minReplicas }}
  maxReplicas: {{ .Values.autoscaling.maxReplicas }}
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: {{ .Values.autoscaling.targetCPUUtilizationPercentage }}
{{- end }}
{{- end }}
```

---

## Testing Template Gating

After gating templates, test both configurations:

### 1. Test Non-Migrated Environment (mozcloud.enabled: false)

```bash
# Should render using custom templates
helm template . -f values.yaml -f values-prod.yaml | grep -c "^kind:"
```

### 2. Test Migrated Environment (mozcloud.enabled: true)

```bash
# Should render using mozcloud chart
helm template . -f values.yaml -f values-dev.yaml | grep -c "^kind:"
```

### 3. Use render-diff to Verify

```bash
# Non-migrated environment should show no changes
render-diff -f values-prod.yaml

# Migrated environment may show changes (expected)
render-diff -f values-dev.yaml -su
```

---

## Common Mistakes to Avoid

### ❌ Mistake 1: Forgetting the "not"

```yaml
# WRONG - this enables template when mozcloud is enabled
{{- if .Values.mozcloud.enabled }}
# ... template ...
{{- end }}

# CORRECT - this disables template when mozcloud is enabled
{{- if not .Values.mozcloud.enabled }}
# ... template ...
{{- end }}
```

### ❌ Mistake 2: Gating Helper Functions

```yaml
# WRONG - don't gate helper functions
{{- if not .Values.mozcloud.enabled }}
{{- define "app.labels" -}}
app: {{ .Chart.Name }}
{{- end }}
{{- end }}

# CORRECT - keep helpers ungated (they're used by both configs)
{{- define "app.labels" -}}
app: {{ .Chart.Name }}
{{- end }}
```

### ❌ Mistake 3: Inconsistent Gating

```yaml
# WRONG - service gated, but deployment not gated
# This creates orphaned services!

# templates/deployment.yaml (NOT GATED)
apiVersion: apps/v1
kind: Deployment
...

# templates/service.yaml (GATED)
{{- if not .Values.mozcloud.enabled }}
apiVersion: v1
kind: Service
...
{{- end }}

# CORRECT - gate related resources together
```

### ❌ Mistake 4: Gating Resources Mozcloud Doesn't Create

```yaml
# WRONG - gating a custom resource that mozcloud can't create
{{- if not .Values.mozcloud.enabled }}
apiVersion: custom.io/v1
kind: CustomResource
...
{{- end }}

# CORRECT - keep custom resources ungated (needed for all envs)
apiVersion: custom.io/v1
kind: CustomResource
...
```

---

## Validation Checklist

After gating templates, verify:

- [ ] All Deployments gated (mozcloud creates workloads)
- [ ] All Services gated (mozcloud creates services)
- [ ] All ConfigMaps gated (or moved to `configs`)
- [ ] All ExternalSecrets gated (or moved to `externalSecrets`)
- [ ] All Ingress resources gated (mozcloud creates HTTPRoutes)
- [ ] Custom resources NOT gated (mozcloud can't create them)
- [ ] Helper templates (_helpers.tpl) NOT gated
- [ ] Related resources gated consistently
- [ ] Non-migrated environments render same resources
- [ ] Migrated environments use mozcloud chart
- [ ] `render-diff` passes for all environments

---

## Cleanup Phase

After ALL environments are migrated:

1. **Remove gated templates entirely**
   ```bash
   rm templates/deployment.yaml
   rm templates/service.yaml
   rm templates/ingress.yaml
   ```

2. **Clean up legacy values**
   ```yaml
   # Remove these from values.yaml
   replicaCount: 2  # Not needed anymore
   image:
     repository: ...
     tag: ...
   ```

3. **Simplify Chart.yaml**
   ```yaml
   dependencies:
     - name: mozcloud
       version: "1.2.3"
       repository: oci://...
       # Remove condition (always enabled now)
   ```

4. **Remove mozcloud.enabled from values**
   ```yaml
   # Remove from all values-*.yaml files
   mozcloud:
     enabled: true  # Not needed anymore
   ```

See [cleanup-phase.md](../references/cleanup-phase.md) for complete cleanup guide.

---

## Summary

**Key Principles**:
1. **Gate templates with `{{- if not .Values.mozcloud.enabled }}`**
2. **Test both configurations** (with and without mozcloud)
3. **Keep custom resources ungated** (mozcloud can't create them)
4. **Don't gate helper functions** (used by both configs)
5. **Gate related resources together** (avoid orphaned resources)
6. **Validate with render-diff** for each environment
7. **Clean up after full migration** (remove gated templates)

**Benefits**:
- [OK] Gradual, low-risk migration
- [OK] Easy rollback per environment
- [OK] No disruption to non-migrated environments
- [OK] Clear separation between old and new configs
- [OK] Simplified cleanup after migration complete
