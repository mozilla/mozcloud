#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(git -C "$SCRIPT_DIR" rev-parse --show-toplevel)"

# Default scope
SCOPE=user
UPDATE=false

MCP_MODULE="github.com/mozilla/mozcloud/tools/mozcloud-mcp"

usage() {
  cat <<EOF
Usage: $(basename "$0") [OPTIONS]

Install mozcloud Claude skills and agents, and configure the mozcloud MCP server.

Options:
  --scope <project|user>  Link skills/agents to .claude/ in the project root
                          (project) or ~/.claude/ (user, default), and register
                          the MCP server at the same scope.
  --update                Update the mozcloud-mcp binary via go install $MCP_MODULE@latest.
  -h, --help              Show this help message and exit.
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --scope)
      SCOPE="${2:-}"
      if [[ "$SCOPE" != "project" && "$SCOPE" != "user" ]]; then
        echo "Error: --scope must be 'project' or 'user'" >&2
        usage >&2
        exit 1
      fi
      shift 2
      ;;
    --update)
      UPDATE=true
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown flag: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

# ---------------------------------------------------------------------------
# Determine target directory based on scope
# ---------------------------------------------------------------------------

if [[ "$SCOPE" == "project" ]]; then
  CLAUDE_DIR="$PROJECT_ROOT/.claude"
else
  CLAUDE_DIR="$HOME/.claude"
fi

# ---------------------------------------------------------------------------
# Symlink skills and agents
# ---------------------------------------------------------------------------

mkdir -p "$CLAUDE_DIR/skills"
mkdir -p "$CLAUDE_DIR/agents"

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
  link "$skill_dir" "$CLAUDE_DIR/skills/$skill_name"
done

for agent_file in "$SCRIPT_DIR/agents"/*.md; do
  [ -e "$agent_file" ] || continue  # skip if glob finds nothing
  agent_name="$(basename "$agent_file")"
  link "$agent_file" "$CLAUDE_DIR/agents/$agent_name"
done

# ---------------------------------------------------------------------------
# MCP server — build and register if not already registered; update if requested
# ---------------------------------------------------------------------------

if [[ "$UPDATE" == true ]]; then
  echo "Updating mozcloud-mcp..."
  go install "$MCP_MODULE@latest"
  echo "mozcloud-mcp updated to $(go env GOPATH)/bin"
elif claude mcp list 2>/dev/null | grep -q "mozcloud"; then
  echo "mozcloud MCP server already registered (use --update to upgrade the binary)"
else
  echo "Installing mozcloud-mcp..."
  go install "$MCP_MODULE@latest"
  echo "mozcloud-mcp installed to $(go env GOPATH)/bin"
  claude mcp add --scope "$SCOPE" mozcloud mozcloud-mcp -- --transport stdio
  echo "registered MCP server at $SCOPE scope"
fi
