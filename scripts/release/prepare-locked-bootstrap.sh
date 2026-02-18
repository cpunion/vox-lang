#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
LOCK_FILE="$ROOT/scripts/release/bootstrap.lock"

usage() {
  cat >&2 <<USAGE
usage: $(basename "$0") <repo> <platform>
example: $(basename "$0") cpunion/vox-lang darwin-amd64
USAGE
}

if [[ $# -ne 2 ]]; then
  usage
  exit 1
fi

REPO="$1"
PLATFORM="$2"

if [[ ! -f "$LOCK_FILE" ]]; then
  echo "[bootstrap] lock file not found: $LOCK_FILE" >&2
  exit 1
fi

# shellcheck disable=SC1090
source "$LOCK_FILE"

TAG="${BOOTSTRAP_TAG:-}"
if [[ -z "$TAG" ]]; then
  echo "[bootstrap] BOOTSTRAP_TAG is empty in $LOCK_FILE" >&2
  exit 1
fi

if ! command -v curl >/dev/null 2>&1; then
  echo "[bootstrap] curl is not installed" >&2
  exit 1
fi

mkdir -p "$ROOT/target/bootstrap"
rm -f "$ROOT/target/bootstrap/vox_prev" "$ROOT/target/bootstrap/vox_prev.exe"

TMP_DIR="$ROOT/.tmp_prev_locked"
rm -rf "$TMP_DIR"
mkdir -p "$TMP_DIR"

ASSET="vox-lang-${TAG}-${PLATFORM}.tar.gz"
echo "[bootstrap] lock tag: $TAG"
echo "[bootstrap] expected asset: $ASSET"
ARCHIVE="$TMP_DIR/$ASSET"
URL="https://github.com/${REPO}/releases/download/${TAG}/${ASSET}"
echo "[bootstrap] download: $URL"
if ! curl -fL --retry 3 --retry-delay 1 --retry-all-errors -o "$ARCHIVE" "$URL"; then
  echo "[bootstrap] failed to download locked bootstrap archive" >&2
  exit 1
fi

if [[ ! -s "$ARCHIVE" ]]; then
  echo "[bootstrap] downloaded archive is empty: $ARCHIVE" >&2
  exit 1
fi

tar -xzf "$ARCHIVE" -C "$TMP_DIR"
PREV_BIN="$(find "$TMP_DIR" -type f \( -path '*/bin/vox' -o -path '*/bin/vox.exe' -o -name 'vox' -o -name 'vox.exe' -o -path '*/bin/vox-stage2' -o -path '*/bin/vox-stage2.exe' -o -name 'vox-stage2' -o -name 'vox-stage2.exe' \) | head -n 1 || true)"
if [[ -z "$PREV_BIN" ]]; then
  echo "[bootstrap] compiler binary not found in locked archive" >&2
  exit 1
fi

cp "$PREV_BIN" "$ROOT/target/bootstrap/vox_prev"
if [[ "$PREV_BIN" == *.exe ]]; then
  cp "$PREV_BIN" "$ROOT/target/bootstrap/vox_prev.exe"
fi

echo "[bootstrap] using locked bootstrap compiler: $PREV_BIN"
