# cloudops-infra Overlay Values System

This reference applies **only when migrating charts within the `cloudops-infra` repository**.
All tenants in cloudops-infra use a rules-based overlay system for environment-specific values.
Per-environment files (`values-dev.yaml`, `values-stage.yaml`, etc.) **do not exist** in the chart directory —
they are generated as part of the migration. Environment-specific values live in a shared `values/` directory
alongside the chart.

## Structure

Every cloudops-infra tenant chart follows this layout:

```
projects/<tenant>/k8s/
├── charts/
│   └── <chart>/          ← CHART_DIR (you are here)
│       ├── Chart.yaml
│       ├── values.yaml   ← base values only
│       └── templates/
└── values/               ← overlay rules live here
    ├── app.yaml
    └── telegraf-override.yaml
```

## How the Overlay System Works

Each file in `values/` contains a list of `rules`. Each rule has:
- `filters`: a map of key-value pairs that must ALL match the deployment parameters
- `values`: chart values to merge on top of `values.yaml` when filters match
- `templating` (optional): if `true`, values may contain Go template expressions

Example `values/app.yaml`:
```yaml
rules:
- filters:
    app: <app-name>
    realm: nonprod
    env: dev
    type: app
  values:
    appLabel: testing
- filters:
    app: <app-name>
    realm: prod
    env: prod
    type: app
  values:
    appLabel: production
```

The deployment tool (`deploylib` via the `tasks` script) is called with parameters like:
```bash
./projects/<tenant>/tasks deploy --realm nonprod --env dev --type app
```

It applies every rule whose filters are a **subset match** of the provided parameters. All specified filter
keys must match — rules with non-matching keys are skipped.

## Common Filter Dimensions

| Filter key | Typical values | Meaning |
|---|---|---|
| `app` | `<app-name>`, etc. | The application name |
| `realm` | `nonprod`, `prod` | GCP project realm |
| `env` | `dev`, `stage`, `prod` | Deployment environment |
| `type` | `app`, `worker` | Workload type |
| `lib` | `influx` | Library/sidecar config (e.g., telegraf) |

Rules with `lib: influx` are telegraf sidecar configuration and do **not** affect the main app deployment.

## Release Name Convention

The original chart uses `{{ .Chart.Name }}-{{ .Release.Name }}` as the resource fullname.
The `deploylib` tool passes `env` as the Helm release name, so:

- `env: dev` → release name `dev` → fullname `<chart>-dev`
- `env: prod` → release name `prod` → fullname `<chart>-prod`

**Always pass `release_name=<env>` when calling `helm_template` on the original chart** to get correctly
named resources.

## Generating the True Original Manifest

Because the overlay is not a standard values file, `render-diff` **cannot be used** for comparison.
Instead, manually render the original and compare.

**Step 1**: Collect all matching overlay values for the target environment.

Read `../../values/app.yaml` and find every rule where filters match `{app: <chart>, realm: <realm>, env: <env>, type: app}`.
Merge the `values` blocks together (later rules override earlier ones).

**Step 2**: Create a temporary overlay file in `.migration/`:
```yaml
# .migration/values-<env>-overlay-orig.yaml
# Overlay values applied for <env> (realm:<realm>, env:<env>, type:app)
# from ../../values/app.yaml
appLabel: "testing"
```

**Step 3**: Render the original chart with the correct release name and overlay:
```
helm_template(
  chart_path=$CHART_DIR,
  release_name="<env>",                          # e.g., "dev"
  values_files=["values.yaml", ".migration/values-<env>-overlay-orig.yaml"]
)
```
Save the output as `.migration/manifests/<env>/original-true.yaml`.

**Step 4**: Render the migrated chart:
```
render_manifests(
  chart_path=$CHART_DIR,
  values_files=["values.yaml", "values-<env>.yaml"],
  update_dependencies=true
)
```
Save the output as `.migration/manifests/<env>/migrated.yaml`.

**Step 5**: Compare `original-true.yaml` vs `migrated.yaml` manually to populate `DIFF_ANALYSIS_<env>.md`.

## Generating values-<env>.yaml for MozCloud

When writing the mozcloud `values-<env>.yaml`, apply overlay values on top of the base `values.yaml`:

1. Start with the base chart `values.yaml`
2. For each matching rule in `../../values/app.yaml`, overlay its `values` onto the base
3. Use the resulting merged values when mapping to the mozcloud schema

Example: if base has `appLabel: ""` but the dev overlay has `appLabel: testing`, the
mozcloud ConfigMap data should contain `APP_LABEL: "testing"`.

Add a comment referencing the source rule:
```yaml
APP_LABEL: "testing"  # from overlay: values/app.yaml filters{app:<app-name>,realm:nonprod,env:dev,type:app}
```

## Per-Environment Realm Mapping

| env | realm | GCP project pattern |
|---|---|---|
| `dev` | `nonprod` | `moz-fx-<app>-nonprod` |
| `stage` | `nonprod` | `moz-fx-<app>-nonprod` |
| `prod` | `prod` | `moz-fx-<app>-prod` |

## Validation Without render-diff

Since `render-diff` cannot be used, validation consists of:

1. **Manual manifest comparison**: diff `original-true.yaml` vs `migrated.yaml`
2. **Document all differences** in `DIFF_ANALYSIS_<env>.md` with clear categorization:
   - Expected changes (labels, added mozcloud resources)
   - Functional changes needing review (selector changes, removed lifecycle hooks)
   - Data changes (verify ConfigMap data is identical to true original)
3. **Selector immutability warning**: if Deployment `selector.matchLabels` changes, flag that
   ArgoCD will need to delete+recreate the Deployment on first sync
4. **Schema validation**: still run `schema_validate_values` on the generated values file

## Non-Migrated Environment Verification

For environments not yet migrated, verify the original templates still render correctly:

```
helm_template(
  chart_path=$CHART_DIR,
  release_name="<other-env>",
  values_files=["values.yaml", ".migration/values-<other-env>-overlay-orig.yaml"]
)
```

Since `mozcloud.enabled` defaults to `false`, non-migrated environments will continue to render
the original templates unchanged.

---

## Required Chart.yaml Changes

All cloudops-infra charts use `apiVersion: v1` (legacy Helm convention). Adding the mozcloud
dependency requires upgrading to `apiVersion: v2`. This fails with a clear error if missed:

```
dependencies are not valid in the Chart file with apiVersion 'v1'
```

Change `apiVersion: v1` → `apiVersion: v2` as the **first edit** to every chart.

Also add `mozcloud.enabled: false` to the base `values.yaml`. Without this, `helm template` on
non-migrated environments fails because the mozcloud subchart tries to evaluate `global.mozcloud`
values even when the condition should gate it out.

---

## Required global.mozcloud Fields

The mozcloud schema requires **five fields** in `global.mozcloud`. Missing any one causes a schema
validation error:

```yaml
global:
  mozcloud:
    app_code: <chart-name>       # e.g. "myapp"
    chart: <helm-chart-name>     # e.g. "myapp" (often same as app_code)
    env_code: <env>              # e.g. "dev"
    project_id: <gcp-project>   # e.g. "moz-fx-<app>-nonprod-1234"
    realm: <realm>              # "nonprod" or "prod"
```

`chart` is the one most often missed — it does not appear in the cloudops-infra overlay system
and must be derived from the chart directory name.

---

## Overlay System Nuances

### Multiple Overlay Files

Many cloudops-infra projects have more than one overlay file in `k8s/values/`. Always read **all**
files in the directory before starting. Common patterns found across projects:

| Pattern | Example | Meaning |
|---|---|---|
| `app.yaml` | `<tenant-a>`, `<tenant-b>` | Main app chart rules |
| `<chart>.yaml` | `api.yaml` in `<tenant>` | Rules specific to one chart in a multi-chart project |
| `<chart>.<realm>.yaml` | `api.nonprod.yaml`, `api.prod.yaml` | Chart + realm split |
| `<realm>.yaml` | `nonprod.yaml`, `prod.yaml` | Realm-level overrides (apply to ALL charts in project) |
| `<name>.yaml` | `<tenant>.yaml` | Project-wide settings, no `chart` filter |
| `admin.yaml` | `<tenant>` | Rules for a sibling chart (type: admin), not the app chart |
| `prod.yaml` | `<tenant>` | Production-scale resource overrides — complement, not replace, app.yaml |
| `telegraf-override.yaml` | all | Telegraf sidecar config — uses `lib: influx` filter, **never matches** app chart deployments |

### Sibling Chart Overlay Files

Projects with multiple charts (e.g., `<app>`, `<app>-jobs`, `<app>-test`) often store all overlay
rules in the same `values/` directory. Rules for sibling charts use a `chart:` filter key that
doesn't match the current chart name. **Identify and skip them** — they do not apply to the chart
you're migrating.

Example: `<tenant>/k8s/values/<sibling>.yaml` contains `filters: {app: <tenant>, chart: "<sibling-chart>"}`.
This is for the sibling chart. The overlay never fires when deploying the main chart.

The presence of sibling overlay files also tells you which other charts in the project will need
their own separate migrations.

### Global (Chart-Agnostic) Overlay Files

Files without a `chart:` filter apply to ALL charts in the project. For example, `<tenant>.yaml`
(filter: `app: <tenant>`) and `nonprod.yaml` (filter: `env: dev, realm: nonprod`) both apply to
all charts in the project. These "global" files set shared values like OIDC
domains, mail server config, and environment strings.

Always check every overlay file — not just `app.yaml`. A chart-agnostic file may set values
that are critical to the app and won't appear in `helm template` output if missed.

### The prod.yaml Isolation Pattern

Some tenants use a separate `prod.yaml` file exclusively for production-scale
resource overrides:
- All rules in `prod.yaml` require `{realm: prod, env: prod}` — they will **never match** dev or stage
- When migrating dev: you can skip `prod.yaml` entirely
- When migrating prod: merge `prod.yaml` rules on top of `app.yaml` rules (prod.yaml values override)

### The `templating: true` Rule Pattern

Rules with `templating: true` contain Go template expressions that `deploylib` evaluates before
passing values to Helm:

```yaml
- filters: {app: <app-name>, realm: nonprod, type: admin}
  templating: true
  values:
    cloudArmorPolicy: "<app-name>-{{ env }}-admin"
    ingressCertName: "admin-{{ env }}-<app-name>-{{ realm }}-cloudops.mozgcp.net"
```

When generating `values-dev.yaml`, manually interpolate these expressions:
- `{{ env }}` → the env name (e.g., `dev`)
- `{{ realm }}` → the realm (e.g., `nonprod`)
- `{{ project_id }}` → the GCP project ID

Add a comment noting the original template expression so future migrators understand the source.

### Rule Ordering and Specificity

Rules within a file are applied in the order they appear (later rules override earlier ones when
the same key is set). This can create ambiguity when a type-specific rule appears first and a
generic rule appears later in the same file — the generic rule may override the specific one.

When you see this pattern, check the actual functional intent (what should the value be?) and
verify against the deploylib behavior if possible before committing to a value. Document the
ambiguity in `MIGRATION_PLAN_<env>.md`.

---

## Workload Key Naming in cloudops-infra

cloudops-infra charts use `{{ .Chart.Name }}-{{ .Release.Name }}` as the fullname (where
`Release.Name` = the env, e.g., `dev`). So all resources are named `<chart>-<env>`.

The mozcloud workload key becomes the Deployment/Service/HPA name. **Always use the full
`<chart>-<env>` name as the workload key** — not a shortened version — to preserve the original
resource names and avoid the selector immutability problem with a rename.

```yaml
mozcloud:
  workloads:
    <chart>-dev:    # ← full original name, not just "<chart>" or "app"
      component: app
      ...
```

---

## Selector Immutability — Always Applies

Every cloudops-infra chart migration will result in changed Deployment selector labels. The
original charts use Jenkins-era labels (`app.kubernetes.io/managed-by: jenkins`,
`app.kubernetes.io/instance: <env>`, often with hardcoded env values like `instance: stage`
across all environments). mozcloud uses its own label convention (`env_code: <env>`, etc.).

Because `spec.selector.matchLabels` is immutable in Kubernetes, **every cloudops-infra migration
requires Deployment delete+recreate on first sync**. This is unavoidable and expected. Always:

1. Call this out in `MIGRATION_PLAN_<env>.md` and `CHANGES_<env>.md`
2. Flag it in `DIFF_ANALYSIS_<env>.md` as a functional change requiring engineering attention
3. Recommend coordinating the first sync for a low-traffic window
4. Note that old Deployments become orphans (ArgoCD prune is disabled) and need manual pruning

---

## nginx Configuration in cloudops-infra Charts

### Port Must Be 8080

mozcloud uses `nginx-unprivileged`, which cannot bind to ports below 1024. Any custom nginx
configmap copied from the original chart must have `listen 80` changed to `listen 8080`. Without
this change, the nginx container will fail to start.

Also update the pid path: `pid /var/run/nginx.pid` → `pid /tmp/nginx.pid`.

### Add the Health Check Endpoint

mozcloud's readiness probe defaults to `/__lbheartbeat__`. If the custom nginx config doesn't
include this location, pods will fail readiness checks. Add it if missing:

```nginx
location = /__lbheartbeat__ { return 200; }
location = /__nginxheartbeat__ { return 200; }
```

### nginx-Only Charts (No App Container)

Some charts (e.g., CDN proxies and static content servers) have only an nginx container with no
application. In mozcloud, set `containers: {}` (empty) and configure nginx normally:

```yaml
workloads:
  <chart>-dev:
    nginx:
      configMap: <chart>-dev-nginx  # custom nginx config
    hosts:
      <chart>-dev:
        ...
    containers: {}    # nginx-only — no app container
```

**Critical**: mozcloud only enables the nginx sidecar when `hosts` is non-empty. For nginx-only
charts, hosts must be configured even for dev environments where ingress isn't strictly needed —
without it, the Deployment will have no containers and fail to schedule.

---

## Known mozcloud Limitations in cloudops-infra Context

These are functional capabilities in the original charts that are **not supported** by mozcloud
as of version 0.11.0. Flag each as needing attention in `CHANGES_<env>.md`:

| Feature | Original Pattern | mozcloud Status | Recommended Action |
|---|---|---|---|
| GCP CDN | `BackendConfig.spec.cdn.enabled: true` | Not supported | Keep custom BackendConfig template or file feature request |
| Cloud Armor | `BackendConfig.spec.securityPolicy.name` | Not supported | Keep custom BackendConfig template or apply at network level |
| Connection draining | `BackendConfig.spec.connectionDraining.drainingTimeoutSec` | Not supported | Accept loss or keep custom BackendConfig |
| Dual IPv4+IPv6 Ingress | Two Ingress resources per host | Not supported | IPv6 Ingress becomes orphan; file feature request |
| `tls.create: false` | Expected to suppress ManagedCertificate | Bug: cert always created | Use `tls.type: pre-shared` with a placeholder cert name |
| PodDisruptionBudget | `PodDisruptionBudget` template | Not generated | Keep as ungated custom template (no conflict) |
| Secret volume mounts | `key.json` from K8s Secret at `/var/secrets/google` | Not supported | Migrate to Workload Identity (separate work item) |
| Resource limits | Explicit `limits` values | Always set to 2× requests | Communicate to teams with tight resource budgets |

---

## Secret Migration in cloudops-infra

### SOPS-Injected Raw Secrets → ExternalSecrets

Original cloudops-infra charts render a Kubernetes Secret directly from Helm values, with real
values injected at deploy time from SOPS-encrypted files via `deploylib`. After mozcloud migration,
secrets come from GCP Secret Manager via ExternalSecret.

**Required action before deploying the migration**: Populate `<env>-gke-app-secrets` in GCP
Secret Manager with all secret keys. The old Secret becomes an orphan requiring manual pruning.

If the original chart constructs composite values (e.g., a full `DBURI` from `dbUser`,
`dbPassword`, `dbEndpoint`, `dbName`), the **assembled value** must be stored in GSM — not the
component parts. The assembly logic in the Helm template won't run with ExternalSecrets.

### Auto-Generated ExternalSecret

mozcloud always creates a default ExternalSecret named `{chart}-secrets` from the GSM secret
`<env>-gke-app-secrets`. This happens even for charts that have no application secrets (pure
nginx serving charts, static content servers).

For secrets-free charts: the GSM secret still needs to exist (even if empty/with a placeholder
key) to avoid ArgoCD sync failures.

### GCP Service Account key.json Pattern

Many older cloudops-infra charts inject a GCP service account JSON key via a Kubernetes Secret
mounted at `/var/secrets/google/key.json`. This pattern is incompatible with mozcloud:

```yaml
# Original pattern — NOT supported by mozcloud:
volumes:
  - name: google-cloud-key
    secret:
      secretName: <chart>-<env>-service-account
containers:
  - volumeMounts:
    - name: google-cloud-key
      mountPath: /var/secrets/google
```

mozcloud creates a Workload Identity `ServiceAccount` instead. Migrating an app off this pattern
requires an application code change to use Application Default Credentials (ADC). This is a
blocking dependency that needs coordination with the app team — plan it as a separate work item.

In the meantime, keep the `secret-serviceaccount.yaml` custom template ungated so the Secret
continues to exist, and document the blocker in `CHANGES_<env>.md`.

### Config Files With Embedded Secrets

Some charts store a configuration file (e.g., `app.ini`) as a Kubernetes Secret because it
contains a secret value embedded in the config. This pattern cannot map directly to mozcloud:
- mozcloud ConfigMaps are for non-secret data
- mozcloud ExternalSecrets source from GSM and inject as env vars, not files

The proper migration path is to split the config file into a ConfigMap (non-secret portions) and
an env var (the secret value), then update the application to read from the env var. Plan this as
a separate work item and note the dependency in `CHANGES_<env>.md`.

---

## Multiple Charts in One Tenant

When a project has multiple charts in `k8s/charts/` (e.g., a tenant may have `app`, `admin`, `agent`,
`migrate` charts):

- Migrate one chart at a time — each chart is independent
- Each chart type has its own overlay routing via the `type:` filter key (e.g., `type: app` vs `type: admin`)
- Overlay files like `admin.yaml` or `agent.yaml` are for sibling charts; skip them when migrating the main app chart
- The `migrate` chart (database migration job) may not need continuous mozcloud migration — verify with the team

### Polymorphic Charts (Multiple Workload Types in One Chart)

Some charts render completely different resource sets based on a `deploymentType` value (e.g., a
single chart may handle multiple workload types — each as a separate ArgoCD application with its
own Helm release).

When you encounter this pattern:
1. Identify all types from the overlay files and templates
2. Scope the migration to **one type per migration branch** — "migrating dev" means one release (e.g., `publicapi`)
3. Consider whether to use one values file per type (`values-dev-publicapi.yaml`) or split charts
4. Document the type being migrated prominently in `MIGRATION_PLAN_<env>.md`

Types that render fundamentally different resource kinds (e.g., StatefulSet vs Deployment) are
strong candidates for chart splitting rather than a single polymorphic chart.

