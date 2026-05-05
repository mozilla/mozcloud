# mzcld

Unified CLI for the MozCloud platform.

## Install

```bash
go install github.com/mozilla/mozcloud/tools/mzcld@latest
```

Or build locally:

```bash
make build   # outputs bin/mzcld
make install # installs to $GOPATH/bin
```

## Authentication

mzcld uses a global `--auth-mode` flag to control how it authenticates with GCP APIs. All commands use the same auth path for consistent behavior.

### `gcloud` (default)

Delegates token generation to the gcloud CLI via `gcloud auth print-access-token`. This handles RAPT reauthentication and security key challenges transparently, which is required under Mozilla's Workspace policy.

```bash
# Authenticate first
gcloud auth login

# All commands use your gcloud session
mzcld gsm list -p my-project
mzcld jit elevate
```

### `adc`

Uses Application Default Credentials. Intended for CI pipelines and service accounts where gcloud CLI is not available or interactive auth is not possible.

```bash
# Authenticate with ADC
gcloud auth application-default login

# Or set a service account key
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/key.json

# Pass the flag
mzcld --auth-mode adc jit elevate "deploy pipeline"
```

## Commands

### `mzcld init`

Check that your local environment has the required tools installed and that OCI registry authentication is configured.

```bash
mzcld init
```

### `mzcld iap`

Generate an IAP token for an IAP-protected endpoint, or start a local proxy that injects the token automatically.

The command discovers the OAuth Client ID by inspecting the redirect from the target host, then generates an identity token via service account impersonation.

**Print a token to stdout** (suitable for shell capture):

```bash
TOKEN=$(mzcld iap --host argocd.example.mozilla.com)
```

**Start a local proxy** that forwards requests with the token injected and refreshed automatically:

```bash
mzcld iap --host yardstick.mozilla.org --proxy --port 3000
# ✓ IAP proxy listening on http://localhost:3000
#   Forwarding → https://yardstick.mozilla.org
#   Press Ctrl+C to stop
```

Then point your browser or tool at `http://localhost:3000` instead of the IAP-protected URL.

| Flag | Default | Description |
|------|---------|-------------|
| `--host` | (required) | IAP-protected hostname |
| `--proxy` | false | Start a local proxy instead of printing a token |
| `--port` | 8080 | Local port for the proxy |
| `--service-account` | auto-detected | Service account to impersonate |

Default service accounts are known for `argocd` and `yardstick` (Grafana) hostnames. For any other host, pass `--service-account`.

### `mzcld claude install`

Interactively install Claude Code skills, agents, and the `mozcloud-mcp` server from this repository. Run from anywhere inside the mozcloud repo.

```bash
mzcld claude install
```

Non-interactive:

```bash
mzcld claude install --all --scope user
```

Update the MCP binary to the latest published version:

```bash
mzcld claude install --update
```

### `mzcld claude uninstall`

Remove installed skills, agents, and the MCP server.

```bash
mzcld claude uninstall
```

## Development

```bash
make vet    # go vet
make fmt    # go fmt
make test   # go test
make lint   # golangci-lint
```
