# Tenant bootstrap report — <app_code>

**Tenant:** `<app_code>` (function: `<function>`, risk level: `<risk_level>`, workgroup: `<workgroup>`)
**Tracking:** <jira_ticket>

## PRs landed

- PR 1 — infra project entry: <pr1_url>
- PR 2 — infra project/env resources: <pr2_url>
- PR 3 — tenant definition (GPA): <pr3_url>

## Verification

If you don't already have credentials for the shared GKE clusters, follow the setup at https://mozilla-hub.atlassian.net/wiki/spaces/SRE/pages/27920460/Using+shared+GKE+clusters

The cluster name is `<function>-<risk_level>`; the hosting projects are `moz-fx-<function>-<risk_level>-nonprod` and `moz-fx-<function>-<risk_level>-prod`. Each project has one cluster per region — you only need to connect to one region per risk level to check namespace-scoped resources.

### Namespaces

Non-prod (point kubectl at the nonprod cluster):
- `kubectl get ns <app_code>-dev <app_code>-stage`

Prod (switch to the prod cluster):
- `kubectl get ns <app_code>-prod`

### Workload Identity

ServiceAccounts in each namespace should carry the `iam.gke.io/gcp-service-account` annotation:
- `kubectl -n <app_code>-dev get sa -o yaml`
- `kubectl -n <app_code>-stage get sa -o yaml`
- `kubectl -n <app_code>-prod get sa -o yaml`

### NetworkPolicies

NetworkPolicies should be in place for your configured `application_ports`:
- `kubectl -n <app_code>-dev get networkpolicies`
- `kubectl -n <app_code>-stage get networkpolicies`
- `kubectl -n <app_code>-prod get networkpolicies`

### GCP resources

GCP projects:
- https://console.cloud.google.com/home/dashboard?project=moz-fx-<app_code>-nonprod
- https://console.cloud.google.com/home/dashboard?project=moz-fx-<app_code>-prod

External IPs (1 per tenant-env combination):
- https://console.cloud.google.com/networking/addresses/list?project=moz-fx-<app_code>-nonprod
- https://console.cloud.google.com/networking/addresses/list?project=moz-fx-<app_code>-prod

## Next steps

### 1. Publish container images to GAR

Push your application image to `us-docker.pkg.dev/moz-fx-<app_code>-prod/<app_code>-prod/<image_name>`.

Full setup guide: https://mozilla-hub.atlassian.net/wiki/spaces/SRE/pages/997163545/How+to+Publish+Container+Images+to+GAR

### 2. Tag-scheme expectations

The default tenant template wires ArgoCD Image Updater with these per-env defaults:

- **dev**: tracks the `latest` tag by digest (`update_strategy: digest`)
- **stage / prod**: tracks semver tags matching `^\d+\.\d+\.\d+$`

If your tags use a different scheme (e.g. `v1.2.3`, commit SHAs, env-prefixed tags), edit the `image_regex` / `update_strategy` fields in `global-platform-admin/tenants/<app_code>.yaml` before expecting deploys to flow.

### 3. Sync ArgoCD Applications

Your Applications will appear in the `<function>` Argo instance: https://<function>.argocd.global.mozgcp.net/

They will not show Synced/Healthy until you publish your first container image (step 1) and trigger the initial sync. The first sync of each Application usually needs to be triggered manually.

Suggested auto-sync policy once things are stable:

- **dev**: enable auto-sync
- **stage**: enable auto-sync (Image Updater bumps the semver, Argo rolls it out automatically)
- **prod**: leave auto-sync *disabled* (Image Updater still commits the bump, but an operator reviews and clicks sync)

ArgoCD Service User Guide (sync policies, operations, troubleshooting): https://mozilla-hub.atlassian.net/wiki/spaces/SRE/pages/1695252738/ArgoCD+Service+User+Guide

### 4. Additional bootstrap tasks

- Add your Helm chart to the `k8s/` directory in `<function>-infra`: https://mozilla-hub.atlassian.net/wiki/spaces/SRE/pages/2059600542
- Configure TLS certificates: https://mozilla-hub.atlassian.net/wiki/spaces/SRE/pages/2407497919
