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
global:
  mozcloud:
    app_code: myapp          # Used for SA name, labels
    chart: myapp             # Required — must match the chart name
    env_code: dev
    project_id: my-gcp-project
    realm: nonprod           # nonprod or prod
    image:
      repository: us-docker.pkg.dev/my-project/my-repo/my-image
      tag: "abc1234"         # Use tag from .argocd-source-*.yaml, not "latest"

mozcloud:
  enabled: true              # Toggle for environment-specific migrations

  serviceAccounts:
    default:
      enabled: true          # Creates a SA named after app_code

  configMaps:
    <configmap-name>:
      data:
        KEY: value

  externalSecrets:
    <externalsecret-name>:
      gsmSecretName: my-gsm-secret-name

  workloads:
    <workload-name>:         # Use FULL original deployment name — becomes the Deployment name
      component: app         # Required — e.g. app, worker, script-*
      nginx:
        enabled: false       # Disable nginx sidecar if app handles its own HTTP
      autoscaling:
        enabled: true
        replicas:
          min: 2
          max: 10
        metrics:
          - type: cpu
            threshold: 60
      hosts:                 # Only for workloads that need ingress/gateway
        <host-key>:          # Becomes the Ingress/HTTPRoute name
          type: external
          api: ingress       # or gateway
          domains:
            - myapp.example.com
          addresses:
            - myapp-dev-ip-v4
          tls:
            type: ManagedCertificate
            create: true
          options:
            timeoutSec: 600
            logSampleRate: 10
      containers:
        <container-name>:
          port: 8000
          healthCheck:
            readiness:
              enabled: true
              path: /__lbheartbeat__
            liveness:
              enabled: false
          envVars:
            MY_VAR: "value"
          configMaps:
            - <configmap-name>   # Mounted as envFrom
          resources:
            cpu: "1"
            memory: "1Gi"
      serviceAccount: ""         # Override SA name; defaults to app_code
      terminationGracePeriodSeconds: 60
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
# Original: templates/deployment.yaml — Deployment named "myapp-worker"
# Mozcloud equivalent — workload key becomes the Deployment name
mozcloud:
  workloads:
    myapp-worker:        # Full original name, not shortened
      component: worker
      nginx:
        enabled: false
      autoscaling:
        enabled: false
      containers:
        myapp-worker:
          port: 8000
          resources:
            cpu: "500m"
            memory: "512Mi"
```

### Converting a GKE Ingress
```yaml
# Original: templates/ingress.yaml — Ingress named "myapp"
# Mozcloud equivalent — host key becomes the Ingress/BackendConfig/FrontendConfig/Service name
mozcloud:
  workloads:
    myapp:
      component: app
      hosts:
        myapp:             # Use same name as original Ingress to preserve it
          type: external
          api: ingress
          domains:
            - myapp.dev.example.com
          addresses:
            - myapp-dev-ip-v4
          tls:
            type: ManagedCertificate
            create: true
          options:
            timeoutSec: 600
            logSampleRate: 10
      containers:
        myapp:
          port: 8000
```

**Expected changes when migrating from custom GKE Ingress templates** (not bugs — document in CHANGES):
- Service type: `NodePort` → `ClusterIP` (correct for NEG-based ingress)
- Service port: original app port → `8080` (mozcloud nginx proxy port)
- Service annotation: `beta.cloud.google.com/backend-config` → `cloud.google.com/backend-config` (deprecated → current)
- HPA: `autoscaling/v1` → `autoscaling/v2` (same name, API version upgrade)
- FrontendConfig gains `redirectToHttps: true` (mozcloud default — enables HTTP→HTTPS redirect)
- ManagedCertificate name is auto-derived from domain (`mcrt-<domain-with-dashes>`) by default — use `tls.certs` to preserve the original name (see "Preserving ManagedCertificate Name" below)

### Preserving ServiceAccount Name

By default mozcloud creates a SA named after `global.mozcloud.app_code`. To preserve a different original SA name:

```yaml
mozcloud:
  serviceAccounts:
    default:
      enabled: false        # Disable the default app_code-named SA
    <original-sa-name>:     # Use the original SA name as the key
      gcpServiceAccount:
        name: <gcp-sa-name> # GCP SA name (part before @ in the email)

  workloads:
    myapp:
      serviceAccount: <original-sa-name>  # Reference the custom SA in each workload
```

### Preserving ManagedCertificate Name

By default mozcloud auto-generates ManagedCertificate names from the domain (`mcrt-<domain-with-dashes>`). This will differ from the original chart's cert name, causing a new certificate to be provisioned (which can take minutes to hours).

Always specify the existing cert name in `tls.certs` to perform an in-place replacement:

```yaml
hosts:
  myapp:
    api: ingress
    tls:
      type: ManagedCertificate
      create: true
      certs:
        - my-original-cert-name    # Preserves the existing cert — no reprovisioning
```

Mozcloud creates a ManagedCertificate with that exact name, matching what's already on the cluster. No provisioning delay.

**Post-migration note for generic cert names:** If the original cert has a generic name (e.g. `managed-certificate`), add a TODO to the migration report: after the migration is stable, provision a new cert with a meaningful name by adding it to the `tls.certs` array alongside the old name, then remove the generic entry and prune the old ManagedCertificate via ArgoCD once the new cert is active.

### BackendConfig Options (GKE Ingress)

`timeoutSec` and log sampling are configured via `options` on the host:

```yaml
hosts:
  myapp:
    options:
      timeoutSec: 600    # BackendConfig timeoutSec
      logSampleRate: 10  # Percentage (0-100); stored as sampleRate: 0.1 in BackendConfig
```

Note: `connectionDraining` is not supported in mozcloud. Keep a custom BackendConfig template if it is required.

## ExternalSecrets: Default Auto-Injection vs Explicit Secrets

mozcloud has two separate mechanisms for injecting secrets into containers:

### 1. Default ExternalSecret auto-injection (Deployment template)

When `externalSecrets.default.enabled: true` (the default), mozcloud:
- Automatically creates an ExternalSecret named `{chart}-secrets` pulling from `{env_code}-gke-app-secrets` in GSM
- Automatically adds `secretRef: name: {chart}-secrets` to **every** container's `envFrom` in the workload

You do **not** need to list the default secret in the container's `secrets:` array.

### 2. Explicit container secrets

The `secrets: [name]` array in a container config is for secrets **beyond** the default, or for job containers (which have no auto-injection). The job template does not auto-inject the default secret.

### Workload containers vs job containers

Workload containers (Deployments/Rollouts) get the default secret auto-injected — listing it in `secrets:` is harmless (the chart deduplicates) but unnecessary. Only list **additional** secrets beyond the default.

Job containers do **not** get auto-injection. You must explicitly list all secrets (including the default) in the job container's `secrets:` array:

```yaml
# Workload container — default secret auto-injected, no need to list it
containers:
  myapp:
    configMaps: [myapp-config]
    # No secrets: array needed; default ExternalSecret handles it

# Job container — must list all secrets explicitly
tasks:
  jobs:
    my-job:
      containers:
        my-job:
          configMaps: [myapp-config]
          secrets: [myapp-secrets]   # Required here — no auto-injection in jobs
```

## targetPort Override for Non-HTTP Container Names

mozcloud derives the container port name from the container name using RFC6335 normalization (via `mozcloud.portName()`). A container named `myapp` gets a port named `myapp`. The Service defaults to `targetPort: http`, which won't match.

When the container name is not `http`, add `targetPort` to the host config:

```yaml
hosts:
  myapp:
    type: external
    api: ingress
    targetPort: myapp    # Must match the container port name (= container name)
```

## Known Mozcloud Limitations

The following configurations from custom templates are **not currently supported** in mozcloud. Document each as an accepted difference in `CHANGES_$ENV.md` and consider filing feature requests:

| Feature | mozcloud support | Notes |
|---------|-----------------|-------|
| `strategy.rollingUpdate.maxUnavailable` | Not configurable | Defaults to Kubernetes 25%. Original charts often set `0` for zero-downtime deploys. |
| `strategy.rollingUpdate.maxSurge` | Not configurable | Defaults to Kubernetes 25%. |
| `minReadySeconds` | Not configurable | Defaults to `0`. |
| `progressDeadlineSeconds` | Not configurable | Defaults to `600s` (10 min). Original charts often set `300s`. |
| `revisionHistoryLimit` | Not configurable | Defaults to `10` (same as Kubernetes default — no functional loss). |
| `preStop` lifecycle hooks (app containers) | Not supported | nginx sidecar gets a preStop hook; app containers do not. |
| `BackendConfig.connectionDraining` | Not supported | Cannot set `drainingTimeoutSec`. |
| `BackendConfig.healthCheck` | Not supported | NEG health checks are used instead. |
| `PodDisruptionBudget` | Not generated | Keep as a custom template (ungated — does not duplicate any mozcloud resource). |
| `nginx.enabled: false` (with GKE Ingress) | Not supported | mozcloud's Service always targets port `8080` named `http`, which requires the nginx sidecar. Without nginx, the pod has no port named `http` and traffic cannot reach the app. **Always keep `nginx.enabled: true` (the default) for any workload with a GKE Ingress host.** |
| Job `imagePullPolicy` | Changed to `Always` when image has digest | When `global.mozcloud.image.tag` contains a digest (`sha256:...`), mozcloud forces `imagePullPolicy: Always`. |

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
