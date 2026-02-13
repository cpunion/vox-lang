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
      local resolved="$(command -v "$cc_bin")"
      if [[ "$GOOS" == "windows" ]]; then
        resolved="$(normalize_windows_exe_path "$resolved")"
      fi
      export CC="$resolved"
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
    candidates=(x86_64-w64-mingw32-gcc gcc clang cc)
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

pick_bootstrap_stage2() {
  if [[ -n "${VOX_BOOTSTRAP_STAGE2:-}" && -f "${VOX_BOOTSTRAP_STAGE2}" ]]; then
    printf '%s\n' "$VOX_BOOTSTRAP_STAGE2"
    return 0
  fi

  local candidates=(
    "$ROOT/target/bootstrap/vox_stage2_prev"
    "$ROOT/target/release/vox_stage2"
    "$ROOT/target/debug/vox_stage2"
    "$ROOT/target/debug/vox_stage2_b_tool"
  )

  local c
  local p
  for c in "${candidates[@]}"; do
    if p="$(resolve_bin "$c" 2>/dev/null)"; then
      printf '%s\n' "$p"
      return 0
    fi
  done

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
CC_BASH="${CC:-}"
if [[ -n "$CC_BASH" ]]; then
  echo "[release] bash CC command: $CC_BASH"
fi

STAGE2_BOOTSTRAP=""
if ! STAGE2_BOOTSTRAP="$(pick_bootstrap_stage2)"; then
  echo "[release] rolling bootstrap stage2 binary is required" >&2
  echo "[release] set VOX_BOOTSTRAP_STAGE2 or prepare target/bootstrap/vox_stage2_prev" >&2
  exit 1
fi

BOOTSTRAP_MODE="rolling-stage2"
echo "[release] stage2 bootstrap binary: $STAGE2_BOOTSTRAP"
set +e
"$STAGE2_BOOTSTRAP" >/dev/null 2>&1
stage2_bootstrap_probe=$?
set -e
echo "[release] stage2 bootstrap probe exit: $stage2_bootstrap_probe"

mkdir -p "$ROOT/target/release"
STAGE2_TOOL_LOG="$ROOT/target/release/stage2-tool-build.log"
set +e
(
  cd "$ROOT"
  if [[ "$GOOS" == "windows" && -n "$CC_BASH" ]]; then
    if "$STAGE2_BOOTSTRAP" emit-pkg-c --driver=tool target/release/vox_stage2.c; then
      "$CC_BASH" -v -std=c11 -O0 -g target/release/vox_stage2.c -o target/release/vox_stage2 -lws2_32 -static -Wl,--stack,8388608
    else
      "$STAGE2_BOOTSTRAP" build-pkg --driver=tool target/release/vox_stage2
    fi
  else
    "$STAGE2_BOOTSTRAP" build-pkg --driver=tool target/release/vox_stage2
  fi
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

STAGE2_TOOL="$(resolve_bin "$ROOT/target/release/vox_stage2")"

BUNDLE_NAME="vox-lang-${VERSION}-${PLATFORM}"
BUNDLE_DIR="$DIST_DIR/$BUNDLE_NAME"
ARCHIVE_PATH="$DIST_DIR/${BUNDLE_NAME}.tar.gz"
CHECKSUM_PATH="${ARCHIVE_PATH}.sha256"

rm -rf "$BUNDLE_DIR"
mkdir -p "$BUNDLE_DIR/bin"
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
