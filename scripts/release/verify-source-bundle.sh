#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat >&2 <<USAGE
usage: $(basename "$0") <version>
example: $(basename "$0") v0.2.0
USAGE
}

if [[ $# -ne 1 ]]; then
  usage
  exit 1
fi

VERSION="$1"
VERSION_CORE="$VERSION"
if [[ "$VERSION_CORE" == v* ]]; then
  VERSION_CORE="${VERSION_CORE#v}"
fi

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
DIST_DIR="${DIST_DIR:-$ROOT/dist}"
ARCHIVE="$DIST_DIR/vox-lang-src-${VERSION}.tar.gz"
CHECKSUM="${ARCHIVE}.sha256"

if [[ ! -f "$ARCHIVE" ]]; then
  echo "[verify-source] archive not found: $ARCHIVE" >&2
  exit 1
fi
if [[ ! -f "$CHECKSUM" ]]; then
  echo "[verify-source] checksum file not found: $CHECKSUM" >&2
  exit 1
fi

EXPECTED="$(awk '{print $1}' "$CHECKSUM" | head -n 1)"
if [[ -z "$EXPECTED" ]]; then
  echo "[verify-source] empty checksum file: $CHECKSUM" >&2
  exit 1
fi

if command -v sha256sum >/dev/null 2>&1; then
  ACTUAL="$(sha256sum "$ARCHIVE" | awk '{print $1}')"
else
  ACTUAL="$(shasum -a 256 "$ARCHIVE" | awk '{print $1}')"
fi
if [[ "$EXPECTED" != "$ACTUAL" ]]; then
  echo "[verify-source] checksum mismatch" >&2
  echo "[verify-source] expected: $EXPECTED" >&2
  echo "[verify-source] actual:   $ACTUAL" >&2
  exit 1
fi

TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/vox-source-verify.XXXXXX")"
trap 'rm -rf "$TMP_DIR"' EXIT

tar -xzf "$ARCHIVE" -C "$TMP_DIR"

ROOT_DIR="$TMP_DIR/vox-lang-src-${VERSION}"
BUILDINFO="$ROOT_DIR/src/vox/buildinfo/buildinfo.vox"
MANIFEST="$ROOT_DIR/vox.toml"

if [[ ! -f "$BUILDINFO" ]]; then
  echo "[verify-source] buildinfo missing: $BUILDINFO" >&2
  exit 1
fi
if [[ ! -f "$MANIFEST" ]]; then
  echo "[verify-source] manifest missing: $MANIFEST" >&2
  exit 1
fi

if ! grep -Fq "pub const VERSION: String = \"$VERSION_CORE\";" "$BUILDINFO"; then
  echo "[verify-source] buildinfo VERSION mismatch: expected $VERSION_CORE" >&2
  exit 1
fi
if ! grep -Fq 'pub const CHANNEL: String = "release";' "$BUILDINFO"; then
  echo "[verify-source] buildinfo CHANNEL must be release" >&2
  exit 1
fi
if ! grep -Fq "version = \"$VERSION_CORE\"" "$MANIFEST"; then
  echo "[verify-source] vox.toml version mismatch: expected $VERSION_CORE" >&2
  exit 1
fi

echo "[verify-source] source bundle OK: $ARCHIVE"
echo "[verify-source] checksum OK: $CHECKSUM"
echo "[verify-source] embedded buildinfo: version=$VERSION_CORE channel=release"
