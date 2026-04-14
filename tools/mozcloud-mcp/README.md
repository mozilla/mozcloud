# mozcloud-mcp

An [MCP](https://modelcontextprotocol.io) server that exposes Helm chart operations, OCI registry tooling, render-diff diffing, and mozcloud migration utilities to AI coding assistants such as Claude Code.

## Installation

The easiest way to install is via the `mozcloud-helm` plugin, which includes the MCP server:

```bash
claude plugin add-marketplace https://github.com/mozilla/mozcloud
claude plugin install mozcloud-helm
```

### Manual installation

```bash
# Build and install the binary to $(go env GOPATH)/bin
make install

# Register with Claude Code at project scope
claude mcp add --scope project mozcloud mozcloud-mcp -- --transport stdio
```

Verify registration:

```bash
claude mcp list
```

## Usage

The server runs in **stdio mode** by default (required for Claude Code). It can also run in SSE mode for browser-based clients:

```bash
mozcloud-mcp --transport sse   # listens on :8080
```

### Security: allowed write roots

By default, tools that write files are scoped to the `chart_path` argument passed by the caller. To further restrict writes to specific directories:

```bash
mozcloud-mcp --allowed-write-roots /path/to/charts,/another/path
```

## Available tools

### OCI / Chart Discovery

| Tool | Description |
|------|-------------|
| `helm_chart_latest_version` | Fetch the latest (and recent) versions of a chart from an OCI registry |
| `oci_check_auth` | Verify the local Docker credential store has an entry for a registry |
| `helm_show_values` | Retrieve the default `values.yaml` from a chart in an OCI registry |
| `helm_show_schema` | Retrieve the `values.schema.json` from a chart in an OCI registry |

### Helm Operations

| Tool | Description |
|------|-------------|
| `helm_template` | Render chart templates and return Kubernetes manifests + resource count |
| `helm_lint` | Validate a chart and return errors/warnings |
| `helm_dependency_build` | Build dependencies from `Chart.lock` |
| `helm_dependency_update` | Update dependencies and regenerate `Chart.lock` |
| `helm_pull` | Pull a chart from an OCI registry to a local directory |

### Diff / Rendering

| Tool | Description |
|------|-------------|
| `render_diff` | Render a chart and diff it against a git ref; returns `has_diff`, diff text, and a summary |
| `render_manifests` | Render a chart and return the full manifest output without diffing |

### Schema Validation

| Tool | Description |
|------|-------------|
| `schema_validate_values` | Validate YAML values against a JSON schema |

### Migration Utilities

| Tool | Description |
|------|-------------|
| `migration_preflight_check` | Run all migration prerequisites in one call (helm, render-diff, OCI auth, git cleanliness) |
| `migration_read_status` | Read `.migration/README.md` and `.migration/STATUS.md` from a chart directory |
| `chart_read_metadata` | Parse and return `Chart.yaml`, including mozcloud dependency detection |
| `values_list_environments` | Discover all `values*.yaml` files in a chart directory with extracted environment names |

## Development

```bash
make build    # build to ./bin/mozcloud-mcp
make install  # install to $(go env GOPATH)/bin
make test     # run tests
make lint     # run golangci-lint
make vet      # run go vet
make fmt      # run go fmt
```

Requires Go 1.24+.
