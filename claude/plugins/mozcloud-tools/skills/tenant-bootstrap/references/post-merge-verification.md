# Post-merge verification

Print the block below to the user after PR 3 has merged. Substitute `<app_code>` with the collected value and pick the correct ArgoCD URL line based on the `<function>` the user chose — include only the matching one.

```
Bootstrap complete. Verify your tenant:

1. Namespace exists:
   kubectl get ns <app_code>-dev

2. ArgoCD application appears in:
```

ArgoCD URLs by function:
- `webservices`: https://webservices.argocd.global.mozgcp.net/
- `dataservices`: https://dataservices.argocd.global.mozgcp.net/
- `sandbox`: https://sandbox.argocd.global.mozgcp.net/

```
3. GCP projects visible in console:
   moz-fx-<app_code>-nonprod
   moz-fx-<app_code>-prod

4. External IPs provisioned (1 per tenant-env combination)

Next steps:
- Add your Helm chart to the k8s/ directory in <function>-infra
  See: https://mozilla-hub.atlassian.net/wiki/spaces/SRE/pages/2059600542
- Set up image publishing to GAR:
  https://mozilla-hub.atlassian.net/wiki/spaces/SRE/pages/2409234440
- Configure TLS certificates:
  https://mozilla-hub.atlassian.net/wiki/spaces/SRE/pages/2407497919
```
