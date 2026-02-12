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

normalize_windows_exe_path() {
  local p="$1"
  if [[ "$GOOS" != "windows" ]]; then
    printf '%s\n' "$p"
    return 0
  fi
  if command -v cygpath >/dev/null 2>&1; then
    local wp=""
    wp="$(cygpath -w "$p" 2>/dev/null || true)"
    if [[ -n "$wp" ]]; then
      printf '%s\n' "$wp"
      return 0
    fi
  fi
  printf '%s\n' "$p"
}

bootstrap_cc_env() {
  if [[ "$GOOS" == "windows" ]]; then
    local mingw_a="/c/ProgramData/mingw64/mingw64/bin"
    local mingw_b="/c/ProgramData/chocolatey/lib/mingw/tools/install/mingw64/bin"
    if [[ -d "$mingw_a" ]]; then
      PATH="$mingw_a:$PATH"
    fi
    if [[ -d "$mingw_b" ]]; then
      PATH="$mingw_b:$PATH"
    fi
    export PATH
  fi

  if [[ -n "${CC:-}" ]]; then
    local cc_bin="${CC%% *}"
    if command -v "$cc_bin" >/dev/null 2>&1; then
      local cc_resolved="$(command -v "$cc_bin")"
      if [[ "$GOOS" == "windows" ]]; then
        cc_resolved="$(normalize_windows_exe_path "$cc_resolved")"
      fi
      export CC="$cc_resolved"
      echo "[release] using CC from env: $CC"
      return 0
    fi
    if [[ "$GOOS" == "windows" && "$cc_bin" =~ ^[A-Za-z]:\\ ]]; then
      echo "[release] using CC from env (absolute windows path): $CC"
      return 0
    fi
    echo "[release] CC is set but not found in PATH: $CC" >&2
  fi

  local candidates=()
  if [[ "$GOOS" == "windows" ]]; then
    candidates=(gcc clang cc)
  else
    candidates=(cc gcc clang)
  fi

  local c
  for c in "${candidates[@]}"; do
    if command -v "$c" >/dev/null 2>&1; then
      local resolved="$(command -v "$c")"
      if [[ "$GOOS" == "windows" ]]; then
        resolved="$(normalize_windows_exe_path "$resolved")"
      fi
      export CC="$resolved"
      echo "[release] auto-detected CC: $CC"
      return 0
    fi
  done

  echo "[release] no C compiler found (checked: ${candidates[*]})" >&2
  return 1
}

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
bootstrap_cc_env

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
echo "[release] stage1 user binary: $STAGE1_USER"

set +e
"$STAGE1_USER" >/dev/null 2>&1
probe_code=$?
set -e
echo "[release] stage1 user probe exit: $probe_code"

echo "[release] build stage1 tool binary"
mkdir -p "$ROOT/compiler/stage1/target/release"
STAGE1_TOOL=""
STAGE1_TOOL_LOG="$ROOT/compiler/stage1/target/release/stage1-tool-build.log"
set +e
(
  cd "$ROOT/compiler/stage1"
  "$STAGE1_USER" build-pkg --driver=tool target/release/vox_stage1
) >"$STAGE1_TOOL_LOG" 2>&1
stage1_tool_rc=$?
set -e

if [[ $stage1_tool_rc -ne 0 ]]; then
  echo "[release] stage1 tool self-build failed: exit $stage1_tool_rc" >&2
  echo "[release] stage1 tool build log begin" >&2
  cat "$STAGE1_TOOL_LOG" >&2 || true
  echo "[release] stage1 tool build log end" >&2
  if [[ "$GOOS" == "windows" ]]; then
    echo "[release] windows fallback: stage0 build-tool for stage1" >&2
    "$STAGE0_OUT" build-tool "$ROOT/compiler/stage1"
    STAGE1_TOOL_DEBUG="$(resolve_bin "$ROOT/compiler/stage1/target/debug/vox_stage1")"
    cp "$STAGE1_TOOL_DEBUG" "$ROOT/compiler/stage1/target/release/vox_stage1${EXE_SUFFIX}"
    STAGE1_TOOL="$STAGE1_TOOL_DEBUG"
    stage1_tool_rc=0
  fi
fi

if [[ $stage1_tool_rc -ne 0 ]]; then
  echo "[release] stage1 tool build failed" >&2
  exit $stage1_tool_rc
fi

if [[ -z "$STAGE1_TOOL" ]]; then
  STAGE1_TOOL="$(resolve_bin "$ROOT/compiler/stage1/target/release/vox_stage1")"
fi

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
STAGE2_TOOL_LOG="$ROOT/compiler/stage2/target/release/stage2-tool-build.log"
set +e
(
  cd "$ROOT/compiler/stage2"
  "$STAGE2_BOOTSTRAP" build-pkg --driver=tool target/release/vox_stage2
) >"$STAGE2_TOOL_LOG" 2>&1
stage2_tool_rc=$?
set -e

if [[ $stage2_tool_rc -ne 0 ]]; then
  echo "[release] stage2 tool build failed: exit $stage2_tool_rc" >&2
  echo "[release] stage2 tool build log begin" >&2
  cat "$STAGE2_TOOL_LOG" >&2 || true
  echo "[release] stage2 tool build log end" >&2
  exit $stage2_tool_rc
fi

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
