# mozcloud

Infrastructure tooling, CRD schemas, and Claude Code integrations for MozCloud platform operations.

## Contents

- [CRD Schemas](#crd-schemas)
- [Tools](#tools)
- [Claude Integration](#claude-integration)

---

## CRD Schemas

The [`crdSchemas/`](./crdSchemas/) directory contains JSON schemas for Kubernetes Custom Resource Definitions (CRDs) used across MozCloud clusters. These schemas enable `kubeconform` to validate Helm-rendered manifests against the actual CRDs deployed in our environments.

Schemas are organized by API group (e.g., `argoproj.io/`, `external-secrets.io/`, `cert-manager.io/`) and follow the naming convention `{ResourceKind}_{version}.json`.

### Where these schemas are used

**[deploy-actions: validate-k8s-manifests](https://github.com/mozilla-it/deploy-actions/blob/main/.github/workflows/validate-k8s-manifests.yml)**

This reusable GitHub Actions workflow renders Helm charts and validates the resulting manifests with `kubeconform`. It references this repository as a custom schema location:

```
https://raw.githubusercontent.com/mozilla/mozcloud/main/crdSchemas/{{.Group}}/{{.ResourceKind}}_{{.ResourceAPIVersion}}.json
```

**[mozilla/helm-charts: kubeconform make target](https://github.com/mozilla/helm-charts)**

The `helm-charts` repository has a `kubeconform` Makefile target that similarly pulls schemas from this repository to validate charts locally and in CI.

### Updating schemas

Schemas are extracted from live clusters using [Datree's CRD Extractor](https://github.com/datreeio/CRDs-catalog). Run this against each MozCloud cluster to keep schemas current:

```bash
# Requires: kubectl context pointing at a MozCloud cluster, python3, git
make update_crds
```

This clones the CRDs-catalog repository, runs the extractor against your current `kubectl` context, and copies the results into `crdSchemas/`. Run against multiple clusters to collect all CRDs.

---

## Tools

### [render-diff](./tools/render-diff/)

A fast, local preview tool for Kubernetes manifest changes. Compares locally rendered Helm chart or Kustomize overlay output against a target git ref and prints a colored diff.

```bash
go install github.com/mozilla/mozcloud/tools/render-diff@latest

# Helm chart diff against main
render-diff --path ./charts/my-app --values values-dev.yaml

# Semantic diff using dyff
render-diff --path ./charts/my-app --values values-dev.yaml --semantic
```

### [mozcloud-mcp](./tools/mozcloud-mcp/)

An [MCP](https://modelcontextprotocol.io) server that exposes Helm chart operations, OCI registry tooling, render-diff, and mozcloud migration utilities to Claude Code and other AI coding assistants.

**Tools provided:**
- OCI/Chart discovery (`helm_chart_latest_version`, `helm_show_values`, `helm_show_schema`, `oci_check_auth`)
- Helm operations (`helm_template`, `helm_lint`, `helm_dependency_build`, `helm_dependency_update`, `helm_pull`)
- Diff/rendering (`render_diff`, `render_manifests`)
- Schema validation (`schema_validate_values`)
- Migration utilities (`migration_preflight_check`, `migration_read_status`, `chart_read_metadata`, `values_list_environments`)

See the [mozcloud-mcp README](./tools/mozcloud-mcp/README.md) for installation and usage.

---

## Claude Integration

Skills, agents, and an MCP server for using [Claude Code](https://claude.ai/code) with MozCloud workflows. See the [claude/ README](./claude/README.md) for full installation instructions.

```bash
# Install at project scope (recommended)
./claude/install.sh --scope project

# Or install at user scope (available across all projects)
./claude/install.sh --scope user
```

This symlinks skills and agents into `.claude/` or `~/.claude/`, and installs + registers the `mozcloud-mcp` server.

### Skills and Agents

| Type | Name | Description |
|------|------|-------------|
| Skill | [`mozcloud-chart-migration`](./claude/skills/mozcloud-chart-migration/) | Step-by-step guided Helm chart migration to mozcloud |
| Skill | [`srein-triage`](./claude/skills/srein-triage/) | Aid for daily intake and triage of MozCloud customer support requests in the Jira SREIN project |
| Agent | `mozcloud-helm-migrator` | Autonomous agent for end-to-end chart migrations |
| MCP Server | [`mozcloud-mcp`](./tools/mozcloud-mcp/) | Helm ops, OCI registry checks, schema validation, and render diffs |

### mozcloud-chart-migration skill

Guides migration of custom tenant Helm charts to the shared `mozcloud` chart (`oci://us-west1-docker.pkg.dev/moz-fx-platform-artifacts/mozcloud-charts/mozcloud`).

```bash
# In your infra repo, from a chart directory:
/mozcloud-chart-migration dev
```

Key features:
- Multi-environment migration (dev → stage → prod) with isolated branches
- Semantic diff validation via `render-diff`
- Resource name preservation (requires explicit approval for any changes)
- Migration tracking in `.migration/` directory
- Cross-tenant pattern learning from previous migrations

See the [skill README](./claude/skills/mozcloud-chart-migration/README.md) and [SKILL.md](./claude/skills/mozcloud-chart-migration/SKILL.md) for full documentation.

### srein-triage skill

Aids daily intake and triage of MozCloud customer support requests in the Jira SREIN project. Fetches BACKLOG and NEEDS CLARIFICATION issues, reads the current triage process from Confluence, and produces an HTML report with structured triage suggestions and relevant documentation. Then share your screen and go through the report during the meeting.

```bash
# Triage all Backlog and Needs Clarification issues -- do this 10
# minutes before the meeting
/srein-triage

# Triage a single issue
/srein-triage SREIN-1127
```

Requires the [Atlassian MCP server](https://github.com/atlassian/atlassian-mcp-server) plugin for JIRA/Confluence access. The skill is read-only -- it never modifies tickets or Confluence pages.

See the [skill README](./claude/skills/srein-triage/README.md) and [SKILL.md](./claude/skills/srein-triage/SKILL.md) for full documentation.

