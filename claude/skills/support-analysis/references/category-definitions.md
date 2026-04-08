# Support Category Definitions

Categories for classifying MozCloud support interactions. Each message or ticket should be assigned to one or more categories.

## Jira Ticket Flow

SREIN tickets are created via the JSM intake form and posted to #mozcloud-support by the Jira bot. After triage, they are moved to downstream projects while retaining the `srein` label:
- **SVCSE** — general service engineering requests (~66% of volume)
- **MZCLD** — MozCloud platform-specific work (~8%)
- **INFRASEC** — security-related infrastructure requests (~3%)
- **Other** — OPST, WT, DENG, etc. (~3%)
- **SREIN** — untriaged tickets still in the intake queue (~20%)

Always query by `labels = srein` to capture the full ticket set.

## Categories

### access-permissions
Requests for access to repos, GCP projects, tools, or services.

**Slack indicators**: "access", "grant", "permission", "404 on repo", "can't see", "can't access", "added to", "viewer", "editor"
**Jira indicators**: access requests, role changes, workgroup creation
**Typical resolution**: Manual provisioning by SRE

### deployment-argocd
Issues with ArgoCD deployments, image updates, sync failures.

**Slack indicators**: "argo", "deploy", "image not picked up", "sync", "rollout", "not deploying", "stuck", "degraded"
**Jira indicators**: deployment failures, ArgoCD errors
**Typical resolution**: Manual sync, image updater config fix, or ArgoCD troubleshooting

### terraform-atlantis
Terraform/OpenTofu and Atlantis workflow issues.

**Slack indicators**: "atlantis", "terraform", "opentofu", "plan", "apply", "lock", "state", "drift"
**Jira indicators**: terraform errors (rarely filed)
**Typical resolution**: Slack-based troubleshooting, often untracked

### dns-networking
DNS records, domain configuration, SSL certificates, redirects.

**Slack indicators**: "CNAME", "domain", "SSL", "cert", "redirect", "DNS", "pointing", "Route53"
**Jira indicators**: domain/DNS requests
**Typical resolution**: Manual DNS record creation or cert provisioning

### monitoring-grafana
Grafana dashboards, alerting, metrics, tracing, observability.

**Slack indicators**: "grafana", "alert", "dashboard", "metrics", "tracing", "yardstick", "prometheus", "budget alert"
**Jira indicators**: monitoring setup requests
**Typical resolution**: Config guidance in Slack, rarely ticketed

### database
Database configuration, capacity, upgrades, connection issues.

**Slack indicators**: "DB", "CloudSQL", "MySQL", "Postgres", "connection", "disk size", "RDS", "connection limit"
**Jira indicators**: database changes, disk increases, upgrades
**Typical resolution**: SRE intervention for capacity changes

### gcp-config
GCP project creation, billing, IAM, bucket provisioning, service enablement.

**Slack indicators**: "GCP", "project", "billing", "bucket", "Vertex", "IAM", "service account"
**Jira indicators**: GCP provisioning requests
**Typical resolution**: SREIN ticket for provisioning

### helm-k8s
Helm chart configuration, Kubernetes resource questions, pod networking.

**Slack indicators**: "helm", "chart", "deployment", "HPA", "ingress", "nginx", "k8s", "pod", "container", "mozcloud-workload"
**Jira indicators**: chart config issues (rarely filed)
**Typical resolution**: Slack-based troubleshooting, tribal knowledge

### incident
Production incidents, outages, urgent issues.

**Slack indicators**: "incident", "down", "503", "outage", "urgent", "production issue", "URGENT", "broken"
**Jira indicators**: incident tickets
**Typical resolution**: Incident response workflow

### onboarding
New tenant requests, first-time platform users, service onboarding.

**Slack indicators**: "new tenant", "new service", "onboard", "first time", "getting started"
**Jira indicators**: new tenant requests, workgroup creation
**Typical resolution**: SREIN ticket for provisioning

### pr-review
Code review requests for infrastructure PRs.

**Slack indicators**: ":review:", "review please", "PR", "approve", "LGTM"
**Jira indicators**: (never ticketed — different workflow)
**Typical resolution**: Peer review in GitHub

### documentation
Requests for docs, confusion about where to find information.

**Slack indicators**: "docs", "documentation", "where do I find", "how do I", "is there a guide", "runbook"
**Jira indicators**: doc requests (rarely filed)
**Typical resolution**: Ad-hoc guidance in Slack, points to doc gaps
