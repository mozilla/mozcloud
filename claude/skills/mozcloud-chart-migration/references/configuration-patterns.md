# Mozcloud Configuration Patterns

This guide documents common configuration patterns for mozcloud chart migrations.

## Mozcloud Default Nginx

Mozcloud provides a **default nginx configuration** for sidecar containers. Use mozcloud's default unless you have specific customization requirements.

### When to Use Mozcloud Default Nginx

**Recommended for most cases**:
- Standard reverse proxy to application container
- No special nginx configuration requirements
- Want to stay current with platform standards

**Benefits**:
- Automatically maintained by platform team
- Security updates applied centrally
- Standard configuration across all tenants
- Reduces chart complexity

### Using Mozcloud Default (Recommended)

Simply define the nginx container **without custom ConfigMaps**:

```yaml
mozcloud:
  workloads:
    my-app:
      containers:
        nginx:
          # No image specification - mozcloud provides default nginx version
          # No configMaps - mozcloud provides default nginx.conf

          ports:
            - name: http
              containerPort: 8080
              protocol: TCP

          resources:
            cpu: 100m
            memory: 256Mi

          healthCheck:
            readiness:
              enabled: true
              path: /__lbheartbeat__
            liveness:
              enabled: true
              path: /__nginxheartbeat__
```

Mozcloud will automatically:
- Provide the latest stable nginx image
- Generate a standard nginx.conf
- Mount it as a ConfigMap volume
- Handle all nginx configuration

### When Custom Nginx Config is Needed

**Only use custom nginx configuration if you require**:
- Custom log formats beyond mozcloud defaults
- Special proxy settings or headers
- Custom location blocks or rewrites
- Specific nginx modules or directives

### Custom Nginx Configuration (If Required)

When custom nginx configuration is needed, choose the appropriate approach based on complexity:

#### Option 1: Simple Static Config → Values File

For **simple, static nginx configurations** without Helm templating:

```yaml
mozcloud:
  configMaps:
    my-app-nginx:
      data:
        nginx.conf: |
          # Static nginx configuration here
          server {
            listen 8080;
            location / {
              proxy_pass http://localhost:8000;
            }
          }

  workloads:
    my-app:
      containers:
        nginx:
          volumes:
            - name: my-app-nginx
              type: configMap
              key: nginx.conf
              path: /etc/nginx/nginx.conf
              readOnly: true
```

**Use this when**:
- Config is short and readable
- No Helm templating needed (no `{{ .Values.* }}`)
- Config is fully static

#### Option 2: Complex or Templated Config → Keep in Templates

For **complex configurations or configs using Helm templating**, keep the ConfigMap in the templates directory:

```yaml
# templates/nginx-configmap.yaml
{{- if not (index .Values "mozcloud" "enabled" | default false) }}
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ include "application.fullname" . }}-nginx
  labels:
    {{- include "application.labels" . | nindent 4 }}
data:
  nginx.conf: |
    # Complex nginx config using Helm values
    upstream backend {
      server 127.0.0.1:{{ .Values.appPort }};
    }

    server {
      listen {{ .Values.nginxPort }};
      client_max_body_size {{ .Values.maxBodySize }};

      location / {
        proxy_pass http://backend;
        {{- if .Values.customHeaders }}
        {{- range .Values.customHeaders }}
        proxy_set_header {{ .name }} {{ .value }};
        {{- end }}
        {{- end }}
      }
    }
{{- end }}
```

**Use this when**:
- Config uses Helm templating (`{{ .Values.* }}`, `{{ include }}`, etc.)
- Config is very long (>50 lines)
- Config has complex logic or conditionals
- Config needs to be generated dynamically

**Important**: Gate the template with `{{ if not .Values.mozcloud.enabled }}` so it only renders when using custom templates, not when mozcloud is enabled.

### Decision Guide

When migrating, ask:

1. **Does the nginx ConfigMap use Helm templating?**
   - Yes (has `{{ }}` syntax) → **Keep in templates/** (Option 2)
   - No (fully static) → Continue to step 2

2. **Is the config complex or very long?**
   - Yes (>50 lines, complex logic) → **Keep in templates/** (Option 2)
   - No (simple, short) → **Move to values** (Option 1)

**Example decision message**:
```
Found custom nginx ConfigMap in templates/nginx-configmap.yaml:
- Uses Helm templating: {{ .Values.appPort }}, {{ .Values.maxBodySize }}
- Length: 120 lines
- Has conditional logic

Recommendation: Keep nginx ConfigMap in templates directory (gated with mozcloud.enabled check)
- Preserves Helm templating functionality
- Keeps values file readable
- Easier to maintain complex config

Proceeding with templates approach.
```

### Important Notes

- Define ConfigMap in global `configMaps` section for static configs (Option 1)
- Keep in `templates/` directory for templated configs (Option 2)
- Use `volumes` array (NOT `configMaps` array) to mount as file
- ConfigMap is mounted as a **file volume**, not environment variables

## ConfigMap vs Volumes

Mozcloud has two ways to use ConfigMaps in containers:

### Pattern 1: Environment Variables (use `configMaps` array)

Use when ConfigMap contains key-value pairs that should be environment variables:

```yaml
configMaps:
  my-app-config:
    data:
      PORT: "8080"
      ENV: "production"

workloads:
  my-app:
    containers:
      app:
        configMaps:
          - my-app-config  # Mounted as environment variables
```

### Pattern 2: File Volumes (use `volumes` array)

Use when ConfigMap contains files that should be mounted in the filesystem:

```yaml
configMaps:
  nginx-config:
    data:
      nginx.conf: |
        # file content here

workloads:
  my-app:
    containers:
      nginx:
        volumes:
          - name: nginx-config
            type: configMap
            key: nginx.conf
            path: /etc/nginx/nginx.conf
            readOnly: true
```

**Never use both** `configMaps` and `volumes` arrays for the same ConfigMap - choose one pattern based on whether you need environment variables or file mounts.
