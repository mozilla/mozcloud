#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

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
  agent_name="$(basename "$agent_file")"
  link "$agent_file" "$HOME/.claude/agents/$agent_name"
done
