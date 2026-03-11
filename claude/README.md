# mozcloud Claude integration

Skills, agents, and an MCP server for using Claude Code with mozcloud. Currently supporting mozcloud Helm chart migrations.

## Requirements

- [Claude Code](https://claude.ai/code)
- Go 1.21+

## Install

```bash
./claude/install.sh
```

This will:
- Symlink skills and agents into `~/.claude/` (user scope by default)
- Install and register the `mozcloud-mcp` MCP server if not already present

### Scope

Use `--scope` to control where skills and agents are linked and at what scope the MCP server is registered.

**User scope** (default) — available across all projects:

```bash
./claude/install.sh --scope user
```

Skills and agents are linked to `~/.claude/`. The MCP server is registered globally.

**Project scope** — available only in this repo:

```bash
./claude/install.sh --scope project
```

Skills and agents are linked to `.claude/` in the repository root. The MCP server is registered in the project's `.claude/settings.json`.

### Update

To upgrade the `mozcloud-mcp` binary to the latest published version:

```bash
./claude/install.sh --update
```

## What's included

| Type | Name | Description |
|------|------|-------------|
| Skill | `mozcloud-chart-migration` | Step-by-step Helm chart migration to mozcloud |
| Agent | `mozcloud-helm-migrator` | Autonomous agent for end-to-end chart migrations |
| MCP server | `mozcloud-mcp` | Tools for helm ops, OCI registry checks, schema validation, and render diffs |
