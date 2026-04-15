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
# Helm chart migration toolkit (skill + agent + MCP server, requires Go 1.21+)
claude plugin install mozcloud-helm

# Customer support analysis (no dependencies)
claude plugin install mozcloud-support
```

The `mozcloud-helm` plugin requires building the MCP server binary:

```bash
go install github.com/mozilla/mozcloud/tools/mozcloud-mcp@latest
```

## Available plugins

| Plugin | Contents | Requirements |
|--------|----------|--------------|
| `mozcloud-helm` | `mozcloud-chart-migration` skill, `mozcloud-helm-migrator` agent, `mozcloud` MCP server | Go 1.21+ |
| `mozcloud-support` | `support-analysis` skill, `srein-triage` skill | None |

## Details

| Type | Name | Description |
|------|------|-------------|
| Skill | `mozcloud-chart-migration` | Step-by-step Helm chart migration to mozcloud |
| Skill | `srein-triage` | Aid for daily intake and triage of MozCloud customer support requests in the Jira SREIN project |
| Agent | `mozcloud-helm-migrator` | Autonomous agent for end-to-end chart migrations |
| MCP server | `mozcloud-mcp` | Tools for helm ops, OCI registry checks, schema validation, and render diffs |
