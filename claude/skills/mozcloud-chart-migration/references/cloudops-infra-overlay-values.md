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
    app: pulsebot
    realm: nonprod
    env: dev
    type: app
  values:
    pulseApplabel: testing
- filters:
    app: pulsebot
    realm: prod
    env: prod
    type: app
  values:
    pulseApplabel: production
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
| `app` | `pulsebot`, `phabricator`, etc. | The application name |
| `realm` | `nonprod`, `prod` | GCP project realm |
| `env` | `dev`, `stage`, `prod` | Deployment environment |
| `type` | `app`, `worker` | Workload type |
| `lib` | `influx` | Library/sidecar config (e.g., telegraf) |

Rules with `lib: influx` are telegraf sidecar configuration and do **not** affect the main app deployment.

## Release Name Convention

The original chart uses `{{ .Chart.Name }}-{{ .Release.Name }}` as the resource fullname.
The `deploylib` tool passes `env` as the Helm release name, so:

- `env: dev` → release name `dev` → fullname `<chart>-dev` (e.g., `pulsebot-dev`)
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
pulseApplabel: "testing"
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

Example: if base has `pulseApplabel: ""` but the dev overlay has `pulseApplabel: testing`, the
mozcloud ConfigMap data should contain `PULSE_APPLABEL: "testing"`.

Add a comment referencing the source rule:
```yaml
PULSE_APPLABEL: "testing"  # from overlay: values/app.yaml filters{app:pulsebot,realm:nonprod,env:dev,type:app}
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
