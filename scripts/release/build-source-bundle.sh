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
BUNDLE_NAME="vox-lang-src-${VERSION}"
ARCHIVE_PATH="$DIST_DIR/${BUNDLE_NAME}.tar.gz"
CHECKSUM_PATH="${ARCHIVE_PATH}.sha256"

if ! command -v git >/dev/null 2>&1; then
  echo "[source] git is required to build source bundle" >&2
  exit 1
fi

mkdir -p "$DIST_DIR"

TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/vox-src-bundle.XXXXXX")"
trap 'rm -rf "$TMP_DIR"' EXIT

git -C "$ROOT" archive --format=tar --prefix="${BUNDLE_NAME}/" HEAD | tar -xf - -C "$TMP_DIR"

BUILDINFO_PATH="$TMP_DIR/$BUNDLE_NAME/src/vox/buildinfo/buildinfo.vox"
if [[ ! -f "$BUILDINFO_PATH" ]]; then
  echo "[source] buildinfo file missing in archive: $BUILDINFO_PATH" >&2
  exit 1
fi

cat > "$BUILDINFO_PATH" <<EOF_BUILDINFO
// Build metadata embedded in the compiler executable.
// Default channel is "dev" for repository builds; release scripts override this file.

pub const VERSION: String = "$VERSION_CORE";
pub const CHANNEL: String = "release";

pub fn version() -> String { return VERSION; }
pub fn channel() -> String { return CHANNEL; }
EOF_BUILDINFO

rm -f "$ARCHIVE_PATH" "$CHECKSUM_PATH"
(
  cd "$TMP_DIR"
  tar -czf "$ARCHIVE_PATH" "$BUNDLE_NAME"
)

if command -v sha256sum >/dev/null 2>&1; then
  sha256sum "$ARCHIVE_PATH" > "$CHECKSUM_PATH"
else
  shasum -a 256 "$ARCHIVE_PATH" > "$CHECKSUM_PATH"
fi

echo "[source] wrote: $ARCHIVE_PATH"
echo "[source] wrote: $CHECKSUM_PATH"
