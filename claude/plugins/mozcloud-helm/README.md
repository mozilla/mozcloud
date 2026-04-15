# mozcloud-helm

Claude Code plugin for MozCloud Helm chart migrations.

## What's included

| Type | Name | Description |
|------|------|-------------|
| Skill | `mozcloud-chart-migration` | Step-by-step Helm chart migration to the MozCloud shared chart |
| Agent | `mozcloud-helm-migrator` | Autonomous agent for end-to-end chart migrations |
| MCP server | `mozcloud` | Tools for helm ops, OCI registry checks, schema validation, and render diffs |

## Requirements

- Go 1.21+ (for building the MCP server)

## Install

```bash
# One-time: add the mozcloud marketplace
claude plugin marketplace add mozilla/mozcloud

# Install this plugin
claude plugin install mozcloud-helm
```

The MCP server binary must be built separately:

```bash
go install github.com/mozilla/mozcloud/tools/mozcloud-mcp@latest
```

## Usage

Once installed, the skill and agent are available in any Claude Code session:

- Ask Claude to migrate a Helm chart and it will use the `mozcloud-helm-migrator` agent
- Run `/mozcloud-chart-migration` for the interactive step-by-step skill
