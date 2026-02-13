#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
LOCK_FILE="$ROOT/scripts/release/bootstrap-stage2.lock"

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

TAG="${STAGE2_BOOTSTRAP_TAG:-}"
if [[ -z "$TAG" ]]; then
  echo "[bootstrap] STAGE2_BOOTSTRAP_TAG is empty in $LOCK_FILE" >&2
  exit 1
fi

if ! command -v gh >/dev/null 2>&1; then
  echo "[bootstrap] gh is not installed" >&2
  exit 1
fi

mkdir -p "$ROOT/target/bootstrap"
rm -f "$ROOT/target/bootstrap/vox_stage2_prev" "$ROOT/target/bootstrap/vox_stage2_prev.exe"

TMP_DIR="$ROOT/.tmp_prev_locked"
rm -rf "$TMP_DIR"
mkdir -p "$TMP_DIR"

ASSET="vox-lang-${TAG}-${PLATFORM}.tar.gz"
echo "[bootstrap] lock tag: $TAG"
echo "[bootstrap] expected asset: $ASSET"

gh release download "$TAG" --repo "$REPO" --pattern "$ASSET" --dir "$TMP_DIR"

ARCHIVE="$TMP_DIR/$ASSET"
if [[ ! -f "$ARCHIVE" ]]; then
  echo "[bootstrap] missing archive after download: $ARCHIVE" >&2
  exit 1
fi

tar -xzf "$ARCHIVE" -C "$TMP_DIR"
PREV_BIN="$(find "$TMP_DIR" -type f \( -path '*/bin/vox-stage2' -o -path '*/bin/vox-stage2.exe' -o -name 'vox-stage2' -o -name 'vox-stage2.exe' \) | head -n 1 || true)"
if [[ -z "$PREV_BIN" ]]; then
  echo "[bootstrap] stage2 binary not found in locked archive" >&2
  exit 1
fi

cp "$PREV_BIN" "$ROOT/target/bootstrap/vox_stage2_prev"
if [[ "$PREV_BIN" == *.exe ]]; then
  cp "$PREV_BIN" "$ROOT/target/bootstrap/vox_stage2_prev.exe"
fi

echo "[bootstrap] using locked stage2 bootstrap: $PREV_BIN"
