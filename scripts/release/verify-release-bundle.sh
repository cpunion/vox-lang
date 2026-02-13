#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat >&2 <<USAGE
usage: $(basename "$0") <version> [platform]
example: $(basename "$0") v0.1.0 darwin-amd64
USAGE
}

if [[ $# -lt 1 || $# -gt 2 ]]; then
  usage
  exit 1
fi

VERSION="$1"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
DIST_DIR="${DIST_DIR:-$ROOT/dist}"

if [[ $# -eq 2 ]]; then
  PLATFORM="$2"
else
  GOOS="$(go env GOOS)"
  GOARCH="$(go env GOARCH)"
  PLATFORM="${GOOS}-${GOARCH}"
fi

ARCHIVE="$DIST_DIR/vox-lang-${VERSION}-${PLATFORM}.tar.gz"
CHECKSUM="${ARCHIVE}.sha256"

if [[ ! -f "$ARCHIVE" ]]; then
  echo "[verify] archive not found: $ARCHIVE" >&2
  exit 1
fi
if [[ ! -f "$CHECKSUM" ]]; then
  echo "[verify] checksum file not found: $CHECKSUM" >&2
  exit 1
fi

EXPECTED="$(awk '{print $1}' "$CHECKSUM" | head -n 1)"
if [[ -z "$EXPECTED" ]]; then
  echo "[verify] empty checksum file: $CHECKSUM" >&2
  exit 1
fi

if command -v sha256sum >/dev/null 2>&1; then
  ACTUAL="$(sha256sum "$ARCHIVE" | awk '{print $1}')"
else
  ACTUAL="$(shasum -a 256 "$ARCHIVE" | awk '{print $1}')"
fi

if [[ "$EXPECTED" != "$ACTUAL" ]]; then
  echo "[verify] checksum mismatch" >&2
  echo "[verify] expected: $EXPECTED" >&2
  echo "[verify] actual:   $ACTUAL" >&2
  exit 1
fi

TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/vox-release-verify.XXXXXX")"
trap 'rm -rf "$TMP_DIR"' EXIT

tar -xzf "$ARCHIVE" -C "$TMP_DIR"

BUNDLE_NAME="vox-lang-${VERSION}-${PLATFORM}"
BUNDLE_DIR="$TMP_DIR/$BUNDLE_NAME"
if [[ ! -d "$BUNDLE_DIR" ]]; then
  echo "[verify] bundle directory missing after extract: $BUNDLE_DIR" >&2
  exit 1
fi

EXE_SUFFIX=""
if [[ "$PLATFORM" == windows-* ]]; then
  EXE_SUFFIX=".exe"
fi

require_file() {
  local p="$1"
  if [[ ! -f "$p" ]]; then
    echo "[verify] required file missing: $p" >&2
    exit 1
  fi
}

require_file "$BUNDLE_DIR/bin/vox-stage2${EXE_SUFFIX}"
require_file "$BUNDLE_DIR/VERSION"
require_file "$BUNDLE_DIR/BOOTSTRAP_MODE"

BUNDLE_VERSION="$(tr -d '\r\n' < "$BUNDLE_DIR/VERSION")"
if [[ "$BUNDLE_VERSION" != "$VERSION" ]]; then
  echo "[verify] VERSION mismatch: expected=$VERSION actual=$BUNDLE_VERSION" >&2
  exit 1
fi

BOOTSTRAP_MODE="$(tr -d '\r\n' < "$BUNDLE_DIR/BOOTSTRAP_MODE")"
if [[ "$BOOTSTRAP_MODE" != "rolling-stage2" ]]; then
  echo "[verify] BOOTSTRAP_MODE must be rolling-stage2, actual=$BOOTSTRAP_MODE" >&2
  exit 1
fi

echo "[verify] bundle OK: $ARCHIVE"
echo "[verify] checksum OK: $CHECKSUM"
echo "[verify] bootstrap mode: $BOOTSTRAP_MODE"
