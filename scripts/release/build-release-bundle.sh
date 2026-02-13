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

pick_bootstrap() {
  if [[ -n "${VOX_BOOTSTRAP:-}" && -f "${VOX_BOOTSTRAP}" ]]; then
    printf '%s\n' "$VOX_BOOTSTRAP"
    return 0
  fi
  if [[ -n "${VOX_BOOTSTRAP_STAGE2:-}" && -f "${VOX_BOOTSTRAP_STAGE2}" ]]; then
    printf '%s\n' "$VOX_BOOTSTRAP_STAGE2"
    return 0
  fi

  local candidates=(
    "$ROOT/target/bootstrap/vox_prev"
    "$ROOT/target/bootstrap/vox_stage2_prev"
    "$ROOT/target/release/vox"
    "$ROOT/target/debug/vox"
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

BUILDINFO_FILE="$ROOT/src/vox/buildinfo/buildinfo.vox"
BUILDINFO_BACKUP="$(mktemp "${TMPDIR:-/tmp}/vox-buildinfo-backup.XXXXXX")"
cp "$BUILDINFO_FILE" "$BUILDINFO_BACKUP"
cleanup_buildinfo() {
  if [[ -f "$BUILDINFO_BACKUP" ]]; then
    cp "$BUILDINFO_BACKUP" "$BUILDINFO_FILE"
    rm -f "$BUILDINFO_BACKUP"
  fi
}
trap cleanup_buildinfo EXIT

cat > "$BUILDINFO_FILE" <<EOF_BUILDINFO
// Build metadata embedded in the compiler executable.
// Default channel is "dev" for repository builds; release scripts override this file.

pub const VERSION: String = "$VERSION_CORE";
pub const CHANNEL: String = "release";

pub fn version() -> String { return VERSION; }
pub fn channel() -> String { return CHANNEL; }
EOF_BUILDINFO

mkdir -p "$DIST_DIR"
bootstrap_cc_env
CC_BASH="${CC:-}"
if [[ -n "$CC_BASH" ]]; then
  echo "[release] bash CC command: $CC_BASH"
fi

BOOTSTRAP_BIN=""
if ! BOOTSTRAP_BIN="$(pick_bootstrap)"; then
  echo "[release] rolling bootstrap compiler binary is required" >&2
  echo "[release] set VOX_BOOTSTRAP or prepare target/bootstrap/vox_prev" >&2
  exit 1
fi

BOOTSTRAP_MODE="rolling"
echo "[release] bootstrap compiler binary: $BOOTSTRAP_BIN"
set +e
"$BOOTSTRAP_BIN" >/dev/null 2>&1
bootstrap_probe_rc=$?
set -e
echo "[release] bootstrap probe exit: $bootstrap_probe_rc"

mkdir -p "$ROOT/target/release"
TOOL_BUILD_LOG="$ROOT/target/release/tool-build.log"
set +e
(
  cd "$ROOT"
  if [[ "$GOOS" == "windows" && -n "$CC_BASH" ]]; then
    if "$BOOTSTRAP_BIN" emit-pkg-c --driver=tool target/release/vox.c; then
      "$CC_BASH" -v -std=c11 -O0 -g target/release/vox.c -o target/release/vox -lws2_32 -static -Wl,--stack,8388608
    else
      "$BOOTSTRAP_BIN" build-pkg --driver=tool target/release/vox
    fi
  else
    "$BOOTSTRAP_BIN" build-pkg --driver=tool target/release/vox
  fi
) >"$TOOL_BUILD_LOG" 2>&1
tool_rc=$?
set -e

if [[ $tool_rc -ne 0 ]]; then
  echo "[release] tool build failed: exit $tool_rc" >&2
  echo "[release] tool build log begin" >&2
  cat "$TOOL_BUILD_LOG" >&2 || true
  echo "[release] tool build log end" >&2
  exit $tool_rc
fi

TOOL_BIN="$(resolve_bin "$ROOT/target/release/vox")"

BUNDLE_NAME="vox-lang-${VERSION}-${PLATFORM}"
BUNDLE_DIR="$DIST_DIR/$BUNDLE_NAME"
ARCHIVE_PATH="$DIST_DIR/${BUNDLE_NAME}.tar.gz"
CHECKSUM_PATH="${ARCHIVE_PATH}.sha256"

rm -rf "$BUNDLE_DIR"
mkdir -p "$BUNDLE_DIR/bin"
cp "$TOOL_BIN" "$BUNDLE_DIR/bin/vox${EXE_SUFFIX}"
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
