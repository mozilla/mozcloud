#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Flags
MCP_PROJECT=false
MCP_USER=false

usage() {
  cat <<EOF
Usage: $(basename "$0") [OPTIONS]

Install mozcloud Claude skills and agents, and optionally configure the
mozcloud MCP server for Claude Code.

Options:
  --mcp-project   Register the mozcloud MCP server at project scope
                  (runs: claude mcp add --scope project ...).
  --mcp-user      Register the mozcloud MCP server at user scope
                  (runs: claude mcp add --scope user ...).
  -h, --help      Show this help message and exit.

Default behaviour (no flags): symlink skills and agents to ~/.claude/.
EOF
}

for arg in "$@"; do
  case "$arg" in
    --mcp-project) MCP_PROJECT=true ;;
    --mcp-user)    MCP_USER=true ;;
    -h|--help)     usage; exit 0 ;;
    *) echo "Unknown flag: $arg" >&2; usage >&2; exit 1 ;;
  esac
done

# ---------------------------------------------------------------------------
# Symlink skills and agents (always runs)
# ---------------------------------------------------------------------------

mkdir -p "$HOME/.claude/skills"
mkdir -p "$HOME/.claude/agents"

link() {
  local src="$1"
  local dst="$2"
  if [ -L "$dst" ]; then
    echo "skipped (already linked): $dst"
  elif [ -e "$dst" ]; then
    echo "skipped (file exists): $dst"
  else
    ln -s "$src" "$dst"
    echo "linked: $dst -> $src"
  fi
}

for skill_dir in "$SCRIPT_DIR/skills"/*/; do
  skill_name="$(basename "$skill_dir")"
  link "$skill_dir" "$HOME/.claude/skills/$skill_name"
done

for agent_file in "$SCRIPT_DIR/agents"/*.md; do
  [ -e "$agent_file" ] || continue  # skip if glob finds nothing
  agent_name="$(basename "$agent_file")"
  link "$agent_file" "$HOME/.claude/agents/$agent_name"
done

# ---------------------------------------------------------------------------
# MCP server registration via claude mcp add
# ---------------------------------------------------------------------------

build_mcp() {
  echo "Building mozcloud-mcp..."
  (cd "$SCRIPT_DIR/../tools/mozcloud-mcp" && go install .)
  echo "mozcloud-mcp installed to $(go env GOPATH)/bin"
}

add_mcp() {
  local scope="$1"
  claude mcp add --scope "$scope" mozcloud mozcloud-mcp -- --transport stdio
  echo "registered MCP server at $scope scope"
}

if [ "$MCP_PROJECT" = true ]; then
  build_mcp
  add_mcp project
fi

if [ "$MCP_USER" = true ]; then
  build_mcp
  add_mcp user
fi
