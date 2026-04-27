# mozcloud Claude integration

Skills, agents, and an MCP server for using Claude Code with mozcloud.

## Install

We recommend you install the mozcloud Claude integration using plugins from the plugin registry.

> [!NOTE]
> If you previously installed the `mozcloud-chart-migration` skill and MCP server using the old script-based method, follow the steps in [Migrating from legacy installation](#migrating-from-legacy-installation) before proceeding.

### Install via plugin registry (recommended)

Add the mozcloud marketplace and install the plugins you need:

```bash
# One-time: register the marketplace
claude plugin marketplace add mozilla/mozcloud
```

### Install the individual plugins
```bash
# Platform-engineering toolkit
claude plugin install mozcloud-tools

# Customer support analysis (no dependencies)
claude plugin install mozcloud-support
```

The `mozcloud-tools` plugin's Helm skill and agent use an MCP server that must be built separately (requires Go 1.21+):

```bash
go install github.com/mozilla/mozcloud/tools/mozcloud-mcp@latest
```

## Migrating from legacy installation

If you previously installed the `mozcloud-chart-migration` skill and MCP server via the old script-based method, follow these steps to migrate to the plugin system:

1. Remove the legacy skill directory:
   ```bash
   rm -rf ~/.claude/skills/mozcloud-chart-migration
   ```

2. Verify the legacy MCP server is installed (look for a `mozcloud` entry):
   ```bash
   claude plugin list
   ```

3. Remove the legacy MCP server:
   ```bash
   claude plugin remove mozcloud
   ```

4. Follow the [Install via plugin registry (recommended)](#install-via-plugin-registry-recommended) steps.

5. Follow the [Install the individual plugins](#install-the-individual-plugins) steps.

## Available plugins

| Plugin | Contents | Requirements |
|--------|----------|--------------|
| `mozcloud-tools` | Skills, agent, and MCP server for MozCloud platform work | Go 1.21+ (only for Helm skill/agent) |
| `mozcloud-support` | `support-analysis` skill, `srein-triage` skill | None |

## Details

| Type | Name | Description |
|------|------|-------------|
| Skill | `mozcloud-chart-migration` | Step-by-step Helm chart migration to mozcloud |
| Skill | `srein-triage` | Aid for daily intake and triage of MozCloud customer support requests in the Jira SREIN project |
| Skill | `tenant-bootstrap` | Step-by-step MozCloud tenant bootstrap across three sequential PRs |
| Agent | `mozcloud-helm-migrator` | Autonomous agent for end-to-end chart migrations |
| MCP server | `mozcloud-mcp` | Tools for helm ops, OCI registry checks, schema validation, and render diffs |
