# mozcloud-tools

Claude Code plugin bundling MozCloud platform-engineering tools.

## What's included

| Type | Name | Description |
|------|------|-------------|
| Skill | `mozcloud-chart-migration` | Step-by-step Helm chart migration to the MozCloud shared chart |
| Skill | `tenant-bootstrap` | Step-by-step MozCloud tenant bootstrap across three sequential PRs |
| Agent | `mozcloud-helm-migrator` | Autonomous agent for end-to-end chart migrations |
| MCP server | `mozcloud` | Tools for helm ops, OCI registry checks, schema validation, and render diffs |

## Install

```bash
# One-time: add the mozcloud marketplace
claude plugin marketplace add mozilla/mozcloud

# Install this plugin
claude plugin install mozcloud-tools
```

The `mozcloud-mcp` binary must be on `PATH` for the MCP server to start. Install
the latest published build with:

```bash
curl -fsSL https://storage.googleapis.com/moz-fx-platform-shared-global-mozcloud-tools/install.sh | bash -s mozcloud-mcp
```

Pin a specific version:

```bash
curl -fsSL https://storage.googleapis.com/moz-fx-platform-shared-global-mozcloud-tools/install.sh | bash -s -- mozcloud-mcp --version v1.2.3
```

By default the binary lands in `~/.local/bin`. Override with `INSTALL_DIR=/usr/local/bin` or `--install-dir <dir>`.

## Usage

Once installed, the skills and agent are available in any Claude Code session:

- Ask Claude to migrate a Helm chart and it will use the `mozcloud-helm-migrator` agent
- Run `/mozcloud-chart-migration` for the interactive step-by-step Helm migration
- Run `/tenant-bootstrap` to bootstrap a new MozCloud tenant
