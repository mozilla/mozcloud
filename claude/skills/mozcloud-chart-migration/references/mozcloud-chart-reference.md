# Mozcloud Chart Reference

## Chart Location

- **OCI Repository**: `oci://us-west1-docker.pkg.dev/moz-fx-platform-artifacts/mozcloud-charts/mozcloud`
- **Chart Type**: Application chart (not library)
- **Version Strategy**: Always use the latest version

## Downloading the Chart

During migration setup, download the latest mozcloud chart for reference:

```bash
# Check latest version
helm show chart oci://us-west1-docker.pkg.dev/moz-fx-platform-artifacts/mozcloud-charts/mozcloud

# Download to migration directory
helm pull oci://us-west1-docker.pkg.dev/moz-fx-platform-artifacts/mozcloud-charts/mozcloud \
  --version <latest> \
  --untar \
  --untardir $CHART_DIR/.migration/

# Return to chart root
cd $CHART_DIR
```

## Key Files to Reference

Once downloaded to `.migration/mozcloud/`, reference these files:

1. **`values.yaml`** - Default values and structure
2. **`values.schema.json`** - Schema validation and documentation
3. **`templates/`** - Understanding what resources mozcloud generates

## Chart Dependency Configuration

Add mozcloud as a dependency in the target chart's `Chart.yaml`:

```yaml
dependencies:
  - name: mozcloud
    version: "<latest-version>"
    repository: "oci://us-west1-docker.pkg.dev/moz-fx-platform-artifacts/mozcloud-charts"
    condition: mozcloud.enabled
```

## Values File Structure

The mozcloud chart expects values in this general structure:

```yaml
mozcloud:
  enabled: true  # Toggle for environment-specific migrations

  # Workloads (Deployments/Rollouts/CronJobs)
  workloads:
    <workload-name>:  # Use FULL original deployment name
      type: deployment  # or rollout, cronjob
      image:
        repository: ...
        tag: ...
      # ... other workload config

  # Services
  services:
    <service-name>:
      # ... service config

  # Ingress/HTTPRoutes
  httpRoutes:
    <route-name>:
      # ... routing config

  # ConfigMaps/Secrets
  configMaps:
    <configmap-name>:
      # ... config

  externalSecrets:
    <externalsecret-name>:
      # ... secret config
```

## Important Notes

### Version Compatibility
- Check for nginx image version: if custom chart uses `us-west1-docker.pkg.dev/moz-fx-platform-artifacts/platform-shared-images/nginx-unprivileged:1.22`, use the latest from mozcloud instead
- Mozcloud chart may have newer versions of common dependencies

### Naming Conventions
- **Workload names become Deployment names**: Use full original names as workload keys
- **Service names**: Typically match workload names unless explicitly different
- **ConfigMap/Secret names**: Preserve exact original names to avoid reference breaks

### Schema Validation
- The `values.schema.json` file provides validation rules
- Review it to understand required vs optional fields
- Check for enum values (e.g., workload types, service types)

## Common Migration Patterns

### Converting a Deployment
```yaml
# Original custom chart
# templates/deployment.yaml with values.yaml:
#   image: my-app:v1.0.0
#   replicas: 3

# Mozcloud equivalent
mozcloud:
  workloads:
    my-app-deployment:  # Preserve full original name
      type: deployment
      image:
        repository: my-app
        tag: v1.0.0
      replicas: 3
```

### Converting a Service
```yaml
# Original custom chart
# templates/service.yaml

# Mozcloud equivalent
mozcloud:
  services:
    my-app-service:  # Preserve original service name
      ports:
        - name: http
          port: 8080
          targetPort: 8080
```

### Converting HTTPRoute/Ingress
```yaml
# Original custom chart
# templates/ingress.yaml or httproute.yaml

# Mozcloud equivalent
mozcloud:
  httpRoutes:
    my-app-route:
      hostnames:
        - my-app.example.com
      rules:
        - backendRefs:
            - name: my-app-service
              port: 8080
```

## Troubleshooting Chart Issues

### Chart Not Found
- Verify OCI registry authentication: `gcloud auth configure-docker us-west1-docker.pkg.dev`
- Check network connectivity to GCP Artifact Registry

### Schema Validation Errors
- Run `helm lint` to see schema violations
- Compare your values structure with `.migration/mozcloud/values.yaml`
- Check `values.schema.json` for required fields

### Unexpected Resource Names
- Review mozcloud templates to understand naming logic
- Check for `nameOverride` or `fullnameOverride` in values
- Ensure workload keys match full original deployment names
