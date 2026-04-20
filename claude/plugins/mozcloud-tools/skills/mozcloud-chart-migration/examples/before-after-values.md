# Before/After Values Examples

This document shows real examples from the **cicd-demos** chart migration, demonstrating actual transformations when migrating from custom charts to the mozcloud shared chart. These are production patterns used in Mozilla's infrastructure.

---

## Table of Contents

1. [Complete Multi-Container Application (cicd-demos)](#example-1-complete-multi-container-application-cicd-demos)
2. [GCP Ingress with Static IP and ManagedCertificate](#example-2-gcp-ingress-configuration)
3. [Multi-Environment Values](#example-3-multi-environment-values)
4. [Custom Nginx Configuration](#example-4-custom-nginx-configuration)
5. [Argo Rollout to Deployment Migration](#example-5-argo-rollout-to-deployment)

---

## Example 1: Complete Multi-Container Application (cicd-demos)

This example shows a complete before/after transformation of the cicd-demos chart, including multi-container pods, autoscaling, health checks, and GCP-specific resources.

### Before: Custom Chart

**Chart.yaml**:
```yaml
apiVersion: v2
name: cicd-demos
description: A Helm chart for Kubernetes
type: application
version: 0.1.0
appVersion: "1.16.0"
```

**values.yaml**:
```yaml
app_code: cicd-demos
component_code: cicd-demos-web
projectId: ""
ghaRunId: ""

externalsecrets:
  version: latest
  refresh_interval: 5m

labels:
  app: cicd-demos

image:
  repository: us-docker.pkg.dev/moz-fx-cicd-demos-nonprod/cicd-demos-nonprod/go-demo
  pullPolicy: Always
  tag: "latest"

nginxImage:
  repository: us-west1-docker.pkg.dev/moz-fx-platform-artifacts/platform-shared-images/nginx-unprivileged
  tag: "1.22"
  pullPolicy: IfNotPresent

ingress:
  staticIpName: cicd-demos-ip-v4
  managedCertificates: managed-certificate
  hosts:
    - host: cicd-demos.local
      backend:
        name: cicd-demos
        port: 8080

frontendConfig:
  sslPolicy: "mozilla-intermediate"

resources:
  limits:
    cpu: 100m
    memory: 128Mi
  requests:
    cpu: 100m
    memory: 128Mi

autoscaling:
  minReplicas: 2
  maxReplicas: 4
  cpu:
    averageUtilization: 80
  memory:
    averageValue: 1Gi
```

**values-dev.yaml**:
```yaml
environment: dev
realm: nonprod
projectId: "moz-fx-cicd-demos-nonprod"

image:
  tag: 'ae5f62c26fc72074b1022aa5b85220df66f1384f'
  name: 'us-docker.pkg.dev/moz-fx-cicd-demos-nonprod/cicd-demos-nonprod/go-demo'

ingress:
  staticIpName: cicd-demos-dev-ip-v4
  hosts:
  - host: dev.cicd-demos.nonprod.sandbox.mozgcp.net
    backend:
      name: cicd-demos
      port: 8080

resources:
  limits:
    cpu: 100m
    memory: 128Mi
  requests:
    cpu: 100m
    memory: 128Mi
```

---

### After: Mozcloud Chart

**Chart.yaml**:
```yaml
apiVersion: v2
name: cicd-demos
description: A Helm chart for Kubernetes
type: application
version: 0.1.0
appVersion: "1.16.0"

dependencies:
  - name: mozcloud
    repository: oci://us-west1-docker.pkg.dev/moz-fx-platform-artifacts/mozcloud-charts
    version: 0.7.2
    condition: mozcloud.enabled
```

**values.yaml**:
```yaml
# Mozcloud configuration (disabled by default, enabled per-environment)
mozcloud:
  enabled: false

# Legacy configuration (kept for non-migrated environments)
app_code: cicd-demos
component_code: cicd-demos-web
projectId: ""
ghaRunId: ""
externalsecrets:
  version: latest
  refresh_interval: 5m
labels:
  app: cicd-demos
image:
  repository: us-docker.pkg.dev/moz-fx-cicd-demos-nonprod/cicd-demos-nonprod/go-demo
  pullPolicy: Always
  tag: "latest"
nginxImage:
  repository: us-west1-docker.pkg.dev/moz-fx-platform-artifacts/platform-shared-images/nginx-unprivileged
  tag: "1.22"
  pullPolicy: IfNotPresent
ingress:
  staticIpName: cicd-demos-ip-v4
  managedCertificates: managed-certificate
  hosts:
    - host: cicd-demos.local
      backend:
        name: cicd-demos
        port: 8080
frontendConfig:
  sslPolicy: "mozilla-intermediate"
resources:
  limits:
    cpu: 100m
    memory: 128Mi
  requests:
    cpu: 100m
    memory: 128Mi
autoscaling:
  minReplicas: 2
  maxReplicas: 4
  cpu:
    averageUtilization: 80
  memory:
    averageValue: 1Gi
```

**values-dev.yaml**:
```yaml
environment: dev
realm: nonprod
projectId: "moz-fx-cicd-demos-nonprod"

# Legacy configuration (for non-migrated environments)
image:
  tag: 'ae5f62c26fc72074b1022aa5b85220df66f1384f'
  name: 'us-docker.pkg.dev/moz-fx-cicd-demos-nonprod/cicd-demos-nonprod/go-demo'
ingress:
  staticIpName: cicd-demos-dev-ip-v4
  hosts:
  - host: dev.cicd-demos.nonprod.sandbox.mozgcp.net
    backend:
      name: cicd-demos
      port: 8080
resources:
  limits:
    cpu: 100m
    memory: 128Mi
  requests:
    cpu: 100m
    memory: 128Mi

# ============================================================================
# MOZCLOUD CONFIGURATION (Dev Environment Migration)
# ============================================================================

global:
  mozcloud:
    app_code: cicd-demos
    chart: cicd-demos
    env_code: dev
    project_id: moz-fx-cicd-demos-nonprod
    realm: nonprod
    image:
      repository: us-docker.pkg.dev/moz-fx-cicd-demos-nonprod/cicd-demos-nonprod/go-demo
      tag: ae5f62c26fc72074b1022aa5b85220df66f1384f
  preview:
    pr: ""
    host: ""

mozcloud:
  enabled: true

  configMaps:
    cicd-demos:
      data:
        PORT: "8000"

  workloads:
    cicd-demos:
      component: cicd-demos-web

      autoscaling:
        enabled: true
        metrics:
          - type: cpu
            threshold: 80
        replicas:
          min: 2
          max: 4

      security:
        runAsRoot: false

      # Multi-container configuration
      containers:
        # Nginx sidecar container
        nginx:
          image:
            repository: us-west1-docker.pkg.dev/moz-fx-platform-artifacts/platform-shared-images/nginx-unprivileged
            tag: "1.22"

          ports:
            - name: http
              containerPort: 8080
              protocol: TCP

          resources:
            cpu: 100m
            memory: 256Mi

          configMaps:
            - cicd-demos-nginx

          healthCheck:
            readiness:
              enabled: true
              path: /__lbheartbeat__
              probes:
                initialDelaySeconds: 10
                periodSeconds: 6
                timeoutSeconds: 5
                successThreshold: 1
                failureThreshold: 5
            liveness:
              enabled: true
              path: /__nginxheartbeat__
              probes:
                initialDelaySeconds: 5
                periodSeconds: 6
                timeoutSeconds: 2

          lifecycle:
            preStop:
              exec:
                command:
                  - /bin/bash
                  - -c
                  - /bin/sleep 25 && /usr/sbin/nginx -s quit

        # Main application container
        cicd-demos:
          # Image comes from global.mozcloud.image

          ports:
            - name: app
              containerPort: 8000
              protocol: TCP

          resources:
            cpu: 100m
            memory: 128Mi

          configMaps:
            - cicd-demos

          # Default external secret (cicd-demos-secrets) is automatically mounted

          healthCheck:
            readiness:
              enabled: true
              path: /__lbheartbeat__
              probes:
                initialDelaySeconds: 10
                periodSeconds: 6
                timeoutSeconds: 5
            liveness:
              enabled: true
              path: /__lbheartbeat__
              probes:
                initialDelaySeconds: 10
                periodSeconds: 6
                timeoutSeconds: 5
                failureThreshold: 5

          lifecycle:
            preStop:
              exec:
                command:
                  - /bin/sleep
                  - "25"

      # Ingress configuration (GCP-specific)
      hosts:
        cicd-demos:
          api: ingress  # Creates Ingress resource natively
          domains:
            - dev.cicd-demos.nonprod.sandbox.mozgcp.net
          addresses:
            - cicd-demos-dev-ip-v4  # Static IP name
          tls:
            type: ManagedCertificate  # GCP ManagedCertificate
```

**Key Points**:
- **Resource name preserved**: Workload key `cicd-demos` matches original deployment name
- **Multi-container**: nginx + cicd-demos containers in single pod
- **Autoscaling**: HPA with CPU threshold
- **Health checks**: Separate readiness/liveness probes for each container
- **Lifecycle hooks**: PreStop handlers for graceful shutdown
- **GCP Ingress**: Native Ingress support with static IP and ManagedCertificate
- **Global config**: Common values in `global.mozcloud` for reuse
- **Legacy config kept**: For non-migrated environments

---

## Example 2: GCP Ingress Configuration

Real example showing GCP-specific Ingress with static IPs, ManagedCertificates, and FrontendConfig.

### Before: Custom Template

**templates/ingress.yaml**:
```yaml
{{- if not (index .Values "mozcloud" "enabled" | default false) }}
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: cicd-demos
  annotations:
    kubernetes.io/ingress.global-static-ip-name: cicd-demos-dev-ip-v4
    networking.gke.io/managed-certificates: managed-certificate
    kubernetes.io/ingress.class: "gce"
    networking.gke.io/v1beta1.FrontendConfig: cicd-demos
spec:
  defaultBackend:
    service:
      name: cicd-demos
      port:
        number: 8080
{{- end }}
```

**templates/managedcert.yaml**:
```yaml
{{- if not (index .Values "mozcloud" "enabled" | default false) }}
apiVersion: networking.gke.io/v1
kind: ManagedCertificate
metadata:
  name: managed-certificate
spec:
  domains:
    - dev.cicd-demos.nonprod.sandbox.mozgcp.net
{{- end }}
```

**templates/frontendconfig.yaml**:
```yaml
{{- if not (index .Values "mozcloud" "enabled" | default false) }}
apiVersion: networking.gke.io/v1beta1
kind: FrontendConfig
metadata:
  name: cicd-demos
spec:
  sslPolicy: mozilla-intermediate
{{- end }}
```

### After: Mozcloud Configuration

**values-dev.yaml** (mozcloud section):
```yaml
mozcloud:
  enabled: true

  workloads:
    cicd-demos:
      # ... other workload config ...

      hosts:
        cicd-demos:
          api: ingress  # Creates Ingress natively
          domains:
            - dev.cicd-demos.nonprod.sandbox.mozgcp.net
          addresses:
            - cicd-demos-dev-ip-v4  # Static IP name
          tls:
            type: ManagedCertificate  # GCP ManagedCertificate
          annotations:
            kubernetes.io/ingress.class: "gce"

# FrontendConfig (GCP-specific resource, defined at top level if needed)
frontendConfigs:
  cicd-demos:
    sslPolicy: "mozilla-intermediate"
```

**Result**: Mozcloud creates:
- Ingress resource with GCP annotations
- ManagedCertificate for TLS
- FrontendConfig for SSL policy

**Key Points**:
- `api: ingress` tells mozcloud to create an Ingress resource
- `domains` becomes ManagedCertificate domains
- `addresses` specifies static IP name(s)
- `tls.type: ManagedCertificate` uses GCP's managed certificates
- FrontendConfig/BackendConfig supported as separate resources

---

## Example 3: Multi-Environment Values

Shows how mozcloud configuration differs across environments (dev vs stage).

### values-dev.yaml
```yaml
environment: dev
realm: nonprod
projectId: "moz-fx-cicd-demos-nonprod"

global:
  mozcloud:
    app_code: cicd-demos
    chart: cicd-demos
    env_code: dev
    project_id: moz-fx-cicd-demos-nonprod
    realm: nonprod
    image:
      repository: us-docker.pkg.dev/moz-fx-cicd-demos-nonprod/cicd-demos-nonprod/go-demo
      tag: ae5f62c26fc72074b1022aa5b85220df66f1384f

mozcloud:
  enabled: true

  workloads:
    cicd-demos:
      autoscaling:
        replicas:
          min: 2
          max: 4

      hosts:
        cicd-demos:
          api: ingress
          domains:
            - dev.cicd-demos.nonprod.sandbox.mozgcp.net
          addresses:
            - cicd-demos-dev-ip-v4
          tls:
            type: ManagedCertificate
```

### values-stage.yaml
```yaml
environment: stage
realm: nonprod
projectId: "moz-fx-cicd-demos-nonprod"

global:
  mozcloud:
    app_code: cicd-demos
    chart: cicd-demos
    env_code: stage
    project_id: moz-fx-cicd-demos-nonprod
    realm: nonprod
    image:
      repository: us-docker.pkg.dev/moz-fx-cicd-demos-nonprod/cicd-demos-nonprod/go-demo
      tag: stable-release-tag  # Different tag for stage

mozcloud:
  enabled: true

  workloads:
    cicd-demos:
      autoscaling:
        replicas:
          min: 3  # More replicas in stage
          max: 6

      hosts:
        cicd-demos:
          api: ingress
          domains:
            - stage.cicd-demos.nonprod.sandbox.mozgcp.net  # Different domain
          addresses:
            - cicd-demos-stage-ip-v4  # Different static IP
          tls:
            type: ManagedCertificate
```

**Key Points**:
- Environment-specific overrides in separate files
- Different image tags per environment
- Different replica counts (higher in stage/prod)
- Different domains and static IPs
- `mozcloud.enabled: true` in each migrated environment

---

## Example 4: Custom Nginx Configuration

The cicd-demos chart uses a custom nginx sidecar for reverse proxy.

### Before: Custom ConfigMap Template

**templates/nginx-configmap.yaml**:
```yaml
{{- if not (index .Values "mozcloud" "enabled" | default false) }}
apiVersion: v1
kind: ConfigMap
metadata:
  name: cicd-demos-nginx
data:
  default.conf: |
    server {
      listen 8080;
      server_name _;

      location /__nginxheartbeat__ {
        return 200 "nginx is alive\n";
        add_header Content-Type text/plain;
      }

      location / {
        proxy_pass http://127.0.0.1:8000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
      }
    }
{{- end }}
```

### After: Mozcloud Configuration

**Static file approach** (recommended for large configs):

Create `configs/nginx/default.conf`:
```nginx
server {
  listen 8080;
  server_name _;

  location /__nginxheartbeat__ {
    return 200 "nginx is alive\n";
    add_header Content-Type text/plain;
  }

  location / {
    proxy_pass http://127.0.0.1:8000;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
  }
}
```

**values.yaml**:
```yaml
mozcloud:
  configMaps:
    cicd-demos-nginx:
      files:
        - configs/nginx/default.conf

  workloads:
    cicd-demos:
      containers:
        nginx:
          configMaps:
            - cicd-demos-nginx  # References the ConfigMap
          volumeMounts:
            - name: nginx-conf
              mountPath: /etc/nginx/conf.d
```

**Alternative: Inline data approach**:
```yaml
mozcloud:
  configMaps:
    cicd-demos-nginx:
      data:
        default.conf: |
          server {
            listen 8080;
            server_name _;

            location /__nginxheartbeat__ {
              return 200 "nginx is alive\n";
              add_header Content-Type text/plain;
            }

            location / {
              proxy_pass http://127.0.0.1:8000;
              proxy_set_header Host $host;
              proxy_set_header X-Real-IP $remote_addr;
              proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            }
          }
```

**Key Points**:
- **Static files**: Better for large configs, syntax highlighting in editors
- **Inline data**: Simpler for small configs, everything in values.yaml
- ConfigMap name preserved: `cicd-demos-nginx`
- Referenced in container's `configMaps` list

---

## Example 5: Argo Rollout to Deployment

The cicd-demos chart originally used Argo Rollout for progressive delivery. Mozcloud uses standard Deployments.

### Before: Argo Rollout Template

**templates/rollout.yaml**:
```yaml
{{- if not (index .Values "mozcloud" "enabled" | default false) }}
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: cicd-demos
spec:
  progressDeadlineSeconds: 300
  minReadySeconds: 10
  strategy:
    canary:
      steps:
      - setWeight: 50
      - pause: {duration: 2m}
      - setWeight: 100
      - pause: {duration: 5m}
      - analysis:
          templates:
          - templateName: heartbeat-check
            clusterScope: true
          args:
          - name: serviceUrl
            value: https://dev.cicd-demos.nonprod.sandbox.mozgcp.net/
  replicas: 2
  selector:
    matchLabels:
      app: cicd-demos
  template:
    spec:
      containers:
      - name: cicd-demos
        image: us-docker.pkg.dev/project/repo/go-demo:tag
        # ... container spec ...
{{- end }}
```

### After: Mozcloud Deployment

**values-dev.yaml**:
```yaml
mozcloud:
  enabled: true

  workloads:
    cicd-demos:
      type: deployment  # Standard Kubernetes Deployment (not Rollout)

      # Autoscaling replaces static replica count
      autoscaling:
        enabled: true
        replicas:
          min: 2
          max: 4
        metrics:
          - type: cpu
            threshold: 80

      containers:
        cicd-demos:
          # Container spec...
```

**Key Points**:
- Argo Rollout → standard Kubernetes Deployment
- Progressive delivery strategy removed (handled by ArgoCD)
- Static replicas → HPA with min/max
- Resource name preserved: `cicd-demos`
- Analysis templates removed (monitoring handled separately)

---

## General Transformation Patterns

Based on the cicd-demos migration:

### Environment Variables
```yaml
# Before (not shown in cicd-demos, but common pattern)
env:
  - name: KEY
    value: "value"

# After (mozcloud prefers map format)
env:
  KEY: "value"
```

### Image Configuration
```yaml
# Before
image:
  repository: us-docker.pkg.dev/project/repo/app
  tag: "v1.0.0"
  pullPolicy: Always

# After (in global.mozcloud for reuse)
global:
  mozcloud:
    image:
      repository: us-docker.pkg.dev/project/repo/app
      tag: "v1.0.0"

workloads:
  app-name:
    # Image inherited from global.mozcloud.image
```

### Multi-Container Pods
```yaml
# Before: Multiple container specs in pod template
spec:
  template:
    spec:
      containers:
      - name: nginx
        image: nginx:1.22
      - name: app
        image: app:latest

# After: Containers defined in workload
workloads:
  app-name:
    containers:
      nginx:
        image:
          repository: nginx
          tag: "1.22"
      app:
        # Main app (image from global or specified here)
```

### Health Checks
```yaml
# Before: Kubernetes probe syntax
livenessProbe:
  httpGet:
    path: /__lbheartbeat__
    port: 8000
  initialDelaySeconds: 10
  periodSeconds: 6

# After: Mozcloud health check syntax
healthCheck:
  liveness:
    enabled: true
    path: /__lbheartbeat__
    port: 8000  # Optional, defaults to first container port
    probes:
      initialDelaySeconds: 10
      periodSeconds: 6
```

### Lifecycle Hooks
```yaml
# Before: Kubernetes lifecycle syntax
lifecycle:
  preStop:
    exec:
      command: ["/bin/sleep", "25"]

# After: Same structure in mozcloud
lifecycle:
  preStop:
    exec:
      command:
        - /bin/sleep
        - "25"
```

---

## Tips for Values Transformation

Based on lessons from cicd-demos migration:

1. **Preserve Resource Names**: Use full original deployment name as workload key (`cicd-demos`, not `demos`)
2. **Keep Legacy Config**: Don't delete old values until all environments migrated - it's needed for non-migrated envs
3. **Use global.mozcloud**: Share common config (app_code, project_id, image) across environments
4. **Test Both Configs**: Render with `mozcloud.enabled: false` and `true` to verify both work
5. **Multi-Container Naming**: Container names in `containers` map should match original names
6. **Environment-Specific Overrides**: Only put differences in environment values files (dev/stage/prod)
7. **GCP Resources**: Use `api: ingress` with `tls.type: ManagedCertificate` for GCP
8. **Template Gating**: Wrap all custom templates with `{{- if not .Values.mozcloud.enabled }}`
9. **ConfigMaps**: Use static files for large configs (nginx), inline data for small configs
10. **Documentation**: Update `.migration/` directory with decisions and changes

---

## Migration Checklist

Based on cicd-demos migration:

- [ ] Chart.yaml: Add mozcloud dependency with `condition: mozcloud.enabled`
- [ ] values.yaml: Add mozcloud configuration, keep legacy config
- [ ] values-<env>.yaml: Enable mozcloud, override environment-specific settings
- [ ] global.mozcloud: Set common values (app_code, project_id, image)
- [ ] workloads: Configure with original deployment name as key
- [ ] Multi-container: Define all containers in `containers` map
- [ ] Ingress: Use `hosts` configuration with `api: ingress`
- [ ] ConfigMaps: Move to `configMaps` (inline data or files)
- [ ] Health checks: Convert to mozcloud `healthCheck` format
- [ ] Autoscaling: Convert static replicas to HPA config
- [ ] Templates: Gate all with `{{- if not .Values.mozcloud.enabled }}`
- [ ] Validation: Test with `render-diff -f values-<env>.yaml -su`
- [ ] Documentation: Update `.migration/README.md` with status

---

## Resources

- **cicd-demos chart**: `~/git/sandbox-infra/cicd-demos/k8s/cicd-demos/`
- **Migration docs**: `.migration/README.md`, `STATUS.md`, `CHANGES_*.md`
- **Mozcloud chart**: Download from `oci://us-west1-docker.pkg.dev/moz-fx-platform-artifacts/mozcloud-charts/mozcloud`
- **render-diff tool**: https://github.com/mozilla/mozcloud/tree/main/tools/render-diff
