#!/usr/bin/env bash
# Install a published mozcloud tool from the public release bucket.
#
# Usage:
#   tools/install.sh <tool> [--version <version>] [--install-dir <dir>]
#
# Tools: mzcld, mozcloud-mcp, render-diff
# Version: "latest" (default), or a tag like "v1.2.3", or a snapshot SHA.
#
# Examples:
#   tools/install.sh mzcld
#   tools/install.sh mzcld --version v1.2.3
#   INSTALL_DIR=/usr/local/bin tools/install.sh mozcloud-mcp
set -euo pipefail

BUCKET_URL="https://storage.googleapis.com/moz-fx-platform-shared-global-mozcloud-tools"
SUPPORTED_TOOLS=(mzcld mozcloud-mcp render-diff)
INSTALL_DIR="${INSTALL_DIR:-${HOME}/.local/bin}"
VERSION="latest"
TOOL=""

usage() {
  sed -n '2,12p' "$0" | sed 's/^# \{0,1\}//'
}

err() { echo "error: $*" >&2; exit 1; }
have() { command -v "$1" >/dev/null 2>&1; }

download() {
  local url="$1" out="$2"
  if have curl; then
    curl -fsSL -o "$out" "$url"
  elif have wget; then
    wget -q -O "$out" "$url"
  else
    err "neither curl nor wget found"
  fi
}

# --- args ---
[ $# -gt 0 ] || { usage; exit 1; }
while [ $# -gt 0 ]; do
  case "$1" in
    -h|--help) usage; exit 0 ;;
    --version) VERSION="$2"; shift 2 ;;
    --install-dir) INSTALL_DIR="$2"; shift 2 ;;
    --*) err "unknown flag: $1" ;;
    *)
      [ -z "$TOOL" ] || err "unexpected argument: $1"
      TOOL="$1"; shift
      ;;
  esac
done

[ -n "$TOOL" ] || err "tool name required (one of: ${SUPPORTED_TOOLS[*]})"

ok=0
for t in "${SUPPORTED_TOOLS[@]}"; do [ "$t" = "$TOOL" ] && ok=1 && break; done
[ "$ok" -eq 1 ] || err "unknown tool: $TOOL (supported: ${SUPPORTED_TOOLS[*]})"

# --- detect platform ---
os="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$os" in
  darwin|linux) ;;
  *) err "unsupported OS: $os" ;;
esac

arch="$(uname -m)"
case "$arch" in
  x86_64|amd64) arch=amd64 ;;
  arm64|aarch64) arch=arm64 ;;
  *) err "unsupported arch: $arch" ;;
esac

[ "$os/$arch" != "darwin/amd64" ] || err "darwin/amd64 is not built; use arm64 hardware or linux"

# --- resolve URL ---
# Layout in the release bucket:
#   <tool>/latest/<tool>_<os>_<arch>.tar.gz                  (stable names)
#   <tool>/snapshots/<sha>/<tool>_<os>_<arch>.tar.gz         (stable names)
#   <tool>/<vX.Y.Z>/<tool>_<vX.Y.Z>_<os>_<arch>.tar.gz       (goreleaser names)
case "$VERSION" in
  latest)
    base="${BUCKET_URL}/${TOOL}/latest"
    archive="${TOOL}_${os}_${arch}.tar.gz"
    ;;
  v*)
    base="${BUCKET_URL}/${TOOL}/${VERSION}"
    archive="${TOOL}_${VERSION}_${os}_${arch}.tar.gz"
    ;;
  *)
    # treat as a commit SHA → snapshot
    base="${BUCKET_URL}/${TOOL}/snapshots/${VERSION}"
    archive="${TOOL}_${os}_${arch}.tar.gz"
    ;;
esac

archive_url="${base}/${archive}"
checksums_url="${base}/checksums.txt"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

echo "Downloading ${archive_url}"
download "$archive_url" "${tmp}/${archive}"

# --- verify checksum ---
if download "$checksums_url" "${tmp}/checksums.txt" 2>/dev/null; then
  expected="$(awk -v f="$archive" '$2 == f { print $1 }' "${tmp}/checksums.txt" || true)"
  if [ -n "$expected" ]; then
    if have sha256sum; then
      actual="$(sha256sum "${tmp}/${archive}" | awk '{print $1}')"
    elif have shasum; then
      actual="$(shasum -a 256 "${tmp}/${archive}" | awk '{print $1}')"
    else
      err "no sha256sum/shasum available for verification"
    fi
    [ "$actual" = "$expected" ] || err "checksum mismatch: expected $expected, got $actual"
    echo "checksum ok"
  else
    echo "warning: ${archive} not listed in checksums.txt; skipping verification" >&2
  fi
else
  echo "warning: checksums.txt unavailable; skipping verification" >&2
fi

# --- extract & install ---
tar -xzf "${tmp}/${archive}" -C "$tmp"
binary="${tmp}/${TOOL}"
[ -x "$binary" ] || err "binary ${TOOL} not found in archive"

mkdir -p "$INSTALL_DIR"
install -m 0755 "$binary" "${INSTALL_DIR}/${TOOL}"

echo "installed ${TOOL} → ${INSTALL_DIR}/${TOOL}"
case ":${PATH}:" in
  *":${INSTALL_DIR}:"*) ;;
  *) echo "note: ${INSTALL_DIR} is not in your PATH" >&2 ;;
esac
