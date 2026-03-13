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

## Commands

### `mzcld init`

Check that your local environment has the required tools installed and that OCI registry authentication is configured.

```bash
mzcld init
```

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
