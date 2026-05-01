#!/usr/bin/env bash
# UserPromptSubmit hook for mozcloud-tools.
# On the first prompt of a session, if mozcloud-mcp is missing, inject
# additionalContext that tells Claude to ask the user how to install it.
# Always exits 0 so the prompt continues either way.
#
# Why UserPromptSubmit and not SessionStart: SessionStart context is
# treated as passive ("may or may not be relevant") and Claude won't act
# on it before a user message exists. Firing on the user's first prompt
# puts the directive in the same turn the model is about to respond to.
#
# Suppression:
#   - Permanent opt-out: file
#       ${XDG_CONFIG_HOME:-$HOME/.config}/mozcloud-tools/skip-mcp-prompt
#     (delete to re-enable).
#   - Once-per-session: marker file in $TMPDIR keyed on $CLAUDE_SESSION_ID
#     so we only inject the directive on the first prompt, not every one.
set -eu

if command -v mozcloud-mcp >/dev/null 2>&1; then
  exit 0
fi

skip_file="${XDG_CONFIG_HOME:-$HOME/.config}/mozcloud-tools/skip-mcp-prompt"
skip_dir="$(dirname "$skip_file")"

if [ -f "$skip_file" ]; then
  exit 0
fi

session_id="${CLAUDE_SESSION_ID:-pid-$$}"
marker="${TMPDIR:-/tmp}/mozcloud-mcp-asked-${session_id}"
if [ -f "$marker" ]; then
  exit 0
fi
mkdir -p "$(dirname "$marker")"
: > "$marker"

cat <<EOF
{
  "continue": true,
  "hookSpecificOutput": {
    "hookEventName": "UserPromptSubmit",
    "additionalContext": "PROACTIVE ACTION REQUIRED: Before responding to the user's message, you MUST call the AskUserQuestion tool to ask how to handle a missing dependency. mozcloud-mcp is not on PATH; the mozcloud MCP server will fail to start until it is installed. Present these options:\n\n  - 'Install latest' — run: curl -fsSL https://storage.googleapis.com/moz-fx-platform-shared-global-mozcloud-tools/install.sh | bash -s mozcloud-mcp\n  - 'Install a specific version' — same command with '-- mozcloud-mcp --version vX.Y.Z' (ask the user for the version after they pick this option)\n  - 'Skip for now' — do nothing; warn that MCP tools will be unavailable this session\n  - 'Never show again' — suppress this prompt permanently by running: mkdir -p '${skip_dir}' && touch '${skip_file}'\n\nAfter the user picks: if an install option, run the install command in a Bash tool call and confirm 'mozcloud-mcp --version' works (binary lands in ~/.local/bin by default; if that's not on PATH, suggest INSTALL_DIR=/usr/local/bin). If 'Never show again', tell them they can undo it by deleting ${skip_file}. Then proceed to address the original user message."
  }
}
EOF
