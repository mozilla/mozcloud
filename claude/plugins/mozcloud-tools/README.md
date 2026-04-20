# mozcloud-tools

Claude Code plugin bundling MozCloud platform-engineering tools.

## What's included

| Type | Name | Description |
|------|------|-------------|
| Skill | `mozcloud-chart-migration` | Step-by-step Helm chart migration to the MozCloud shared chart |
| Skill | `tenant-bootstrap` | Step-by-step MozCloud tenant bootstrap across three sequential PRs |
| Agent | `mozcloud-helm-migrator` | Autonomous agent for end-to-end chart migrations |
| MCP server | `mozcloud` | Tools for helm ops, OCI registry checks, schema validation, and render diffs |

## Requirements

- Go 1.21+ (for building the `mozcloud` MCP server; only needed if you use the Helm skill or agent)

## Install

```bash
# One-time: add the mozcloud marketplace
claude plugin marketplace add mozilla/mozcloud

# Install this plugin
claude plugin install mozcloud-tools
```

The MCP server binary must be built separately:

```bash
go install github.com/mozilla/mozcloud/tools/mozcloud-mcp@latest
```

## Usage

Once installed, the skills and agent are available in any Claude Code session:

- Ask Claude to migrate a Helm chart and it will use the `mozcloud-helm-migrator` agent
- Run `/mozcloud-chart-migration` for the interactive step-by-step Helm migration
- Run `/tenant-bootstrap` to bootstrap a new MozCloud tenant
