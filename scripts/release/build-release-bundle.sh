#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat >&2 <<USAGE
usage: $(basename "$0") <version>
example: $(basename "$0") v0.1.0
USAGE
}

if [[ $# -ne 1 ]]; then
  usage
  exit 1
fi

VERSION="$1"
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
DIST_DIR="${DIST_DIR:-$ROOT/dist}"

GOOS="$(go env GOOS)"
GOARCH="$(go env GOARCH)"
PLATFORM="${GOOS}-${GOARCH}"
EXE_SUFFIX=""
if [[ "$GOOS" == "windows" ]]; then
  EXE_SUFFIX=".exe"
fi

resolve_bin() {
  local base="$1"
  if [[ -f "$base" ]]; then
    printf '%s\n' "$base"
    return 0
  fi
  if [[ -f "${base}.exe" ]]; then
    printf '%s\n' "${base}.exe"
    return 0
  fi
  return 1
}

sha256_file() {
  local src="$1"
  local out="$2"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$src" > "$out"
    return
  fi
  shasum -a 256 "$src" > "$out"
}

mkdir -p "$DIST_DIR"

STAGE0_OUT="$ROOT/compiler/stage0/target/release/vox-stage0${EXE_SUFFIX}"
mkdir -p "$(dirname "$STAGE0_OUT")"

echo "[release] build stage0 (${PLATFORM})"
(
  cd "$ROOT/compiler/stage0"
  go build -o "$STAGE0_OUT" ./cmd/vox
)

echo "[release] build stage1 user binary via stage0"
"$STAGE0_OUT" build "$ROOT/compiler/stage1"
STAGE1_USER="$(resolve_bin "$ROOT/compiler/stage1/target/debug/vox_stage1")"

echo "[release] build stage1 tool binary"
mkdir -p "$ROOT/compiler/stage1/target/release"
(
  cd "$ROOT/compiler/stage1"
  "$STAGE1_USER" build-pkg --driver=tool target/release/vox_stage1
)
STAGE1_TOOL="$(resolve_bin "$ROOT/compiler/stage1/target/release/vox_stage1")"

BOOTSTRAP_MODE="stage1-fallback"
STAGE2_BOOTSTRAP=""
if [[ -n "${VOX_BOOTSTRAP_STAGE2:-}" ]]; then
  if [[ -f "$VOX_BOOTSTRAP_STAGE2" ]]; then
    STAGE2_BOOTSTRAP="$VOX_BOOTSTRAP_STAGE2"
  fi
fi
if [[ -z "$STAGE2_BOOTSTRAP" ]]; then
  if p="$(resolve_bin "$ROOT/compiler/stage2/target/bootstrap/vox_stage2_prev" 2>/dev/null)"; then
    STAGE2_BOOTSTRAP="$p"
  fi
fi
if [[ -n "$STAGE2_BOOTSTRAP" ]]; then
  BOOTSTRAP_MODE="rolling-stage2"
  echo "[release] build stage2 tool via previous stage2 binary: $STAGE2_BOOTSTRAP"
else
  STAGE2_BOOTSTRAP="$STAGE1_TOOL"
  echo "[release] build stage2 tool via stage1 fallback"
fi

mkdir -p "$ROOT/compiler/stage2/target/release"
(
  cd "$ROOT/compiler/stage2"
  "$STAGE2_BOOTSTRAP" build-pkg --driver=tool target/release/vox_stage2
)
STAGE2_TOOL="$(resolve_bin "$ROOT/compiler/stage2/target/release/vox_stage2")"

BUNDLE_NAME="vox-lang-${VERSION}-${PLATFORM}"
BUNDLE_DIR="$DIST_DIR/$BUNDLE_NAME"
ARCHIVE_PATH="$DIST_DIR/${BUNDLE_NAME}.tar.gz"
CHECKSUM_PATH="${ARCHIVE_PATH}.sha256"

rm -rf "$BUNDLE_DIR"
mkdir -p "$BUNDLE_DIR/bin"
cp "$STAGE0_OUT" "$BUNDLE_DIR/bin/vox-stage0${EXE_SUFFIX}"
cp "$STAGE1_TOOL" "$BUNDLE_DIR/bin/vox-stage1${EXE_SUFFIX}"
cp "$STAGE2_TOOL" "$BUNDLE_DIR/bin/vox-stage2${EXE_SUFFIX}"
printf '%s\n' "$VERSION" > "$BUNDLE_DIR/VERSION"
printf '%s\n' "$BOOTSTRAP_MODE" > "$BUNDLE_DIR/BOOTSTRAP_MODE"

rm -f "$ARCHIVE_PATH" "$CHECKSUM_PATH"
(
  cd "$DIST_DIR"
  tar -czf "$(basename "$ARCHIVE_PATH")" "$BUNDLE_NAME"
)
sha256_file "$ARCHIVE_PATH" "$CHECKSUM_PATH"

echo "[release] wrote: $ARCHIVE_PATH"
echo "[release] wrote: $CHECKSUM_PATH"
