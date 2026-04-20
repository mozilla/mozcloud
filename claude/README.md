# mozcloud Claude integration

Skills, agents, and an MCP server for using Claude Code with mozcloud.

## Install via plugin registry (recommended)

Add the mozcloud marketplace and install the plugins you need:

```bash
# One-time: register the marketplace
claude plugin marketplace add mozilla/mozcloud
```

## Install the individual plugins
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
