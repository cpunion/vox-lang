#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat >&2 <<USAGE
usage: $(basename "$0") <version> [platform]
example: $(basename "$0") v0.2.0
example: $(basename "$0") v0.2.0 linux-x86
USAGE
}

if [[ $# -lt 1 || $# -gt 2 ]]; then
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

normalize_host_os() {
  local os="$1"
  case "$os" in
    linux) echo "linux" ;;
    darwin|macos) echo "darwin" ;;
    windows|mingw*|msys*|cygwin*) echo "windows" ;;
    *) echo "$os" ;;
  esac
}

normalize_host_arch() {
  local arch="$1"
  case "$1" in
    x64|amd64|x86_64) echo "amd64" ;;
    arm64|aarch64) echo "arm64" ;;
    x86|386|i386|i686) echo "x86" ;;
    *) echo "$arch" ;;
  esac
}

detect_host_os() {
  if [[ -n "${RUNNER_OS:-}" ]]; then
    echo "$(normalize_host_os "$(echo "$RUNNER_OS" | tr '[:upper:]' '[:lower:]')")"
    return 0
  fi
  echo "$(normalize_host_os "$(uname -s | tr '[:upper:]' '[:lower:]')")"
}

detect_host_arch() {
  if [[ -n "${RUNNER_ARCH:-}" ]]; then
    echo "$(normalize_host_arch "$(echo "$RUNNER_ARCH" | tr '[:upper:]' '[:lower:]')")"
    return 0
  fi
  echo "$(normalize_host_arch "$(uname -m | tr '[:upper:]' '[:lower:]')")"
}

HOST_OS="$(detect_host_os)"
HOST_ARCH="$(detect_host_arch)"

PLATFORM="${2:-${VOX_RELEASE_PLATFORM:-${HOST_OS}-${HOST_ARCH}}}"
TARGET_OS="${PLATFORM%%-*}"
TARGET_ARCH="${PLATFORM#*-}"
if [[ -z "$TARGET_OS" || -z "$TARGET_ARCH" || "$TARGET_OS" == "$PLATFORM" ]]; then
  echo "[release] invalid platform: $PLATFORM (expected <os>-<arch>)" >&2
  exit 1
fi

IS_CROSS="0"
if [[ "$TARGET_OS" != "$HOST_OS" || "$TARGET_ARCH" != "$HOST_ARCH" ]]; then
  IS_CROSS="1"
fi

EXE_SUFFIX=""
if [[ "$TARGET_OS" == "windows" ]]; then
  EXE_SUFFIX=".exe"
fi

normalize_windows_exe_path() {
  local p="$1"
  if [[ "$HOST_OS" != "windows" ]]; then
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
  if [[ "$HOST_OS" == "windows" ]]; then
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
      if [[ "$HOST_OS" == "windows" && ( "$cc_bin" == "cl" || "$cc_bin" == "cl.exe" ) ]]; then
        if [[ "$CC" == "cl" || "$CC" == "cl.exe" ]]; then
          export CC="cl /std:c11"
        fi
        echo "[release] using CC from env: $CC"
        return 0
      fi
      local resolved="$(command -v "$cc_bin")"
      if [[ "$HOST_OS" == "windows" ]]; then
        resolved="$(normalize_windows_exe_path "$resolved")"
      fi
      export CC="$resolved"
      echo "[release] using CC from env: $CC"
      return 0
    fi
    if [[ "$HOST_OS" == "windows" && "$cc_bin" =~ ^[A-Za-z]:\\ ]]; then
      echo "[release] using CC from env (absolute windows path): $CC"
      return 0
    fi
    echo "[release] CC is set but not found in PATH: $CC" >&2
  fi

  if [[ "$IS_CROSS" == "1" ]]; then
    echo "[release] cross build: leave CC unset (use target-specific defaults)"
    return 0
  fi

  local candidates=()
  if [[ "$HOST_OS" == "windows" ]]; then
    candidates=(x86_64-w64-mingw32-gcc gcc clang cc)
  else
    candidates=(cc gcc clang)
  fi

  local c
  for c in "${candidates[@]}"; do
    if command -v "$c" >/dev/null 2>&1; then
      local resolved="$(command -v "$c")"
      if [[ "$HOST_OS" == "windows" ]]; then
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
pub const BUILD_SOURCE_ROOT: String = "";

pub fn version() -> String { return VERSION; }
pub fn channel() -> String { return CHANNEL; }
pub fn build_source_root() -> String { return BUILD_SOURCE_ROOT; }
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
bootstrap_help_text="$("$BOOTSTRAP_BIN" 2>&1 || true)"
BOOTSTRAP_IS_LEGACY="0"
if [[ "$bootstrap_help_text" == *"build-pkg <out.bin>"* ]]; then
  BOOTSTRAP_IS_LEGACY="1"
fi
BOOTSTRAP_BASE="$(basename "$BOOTSTRAP_BIN")"
LEGACY_RUNTIME_C="${VOX_LEGACY_C_RUNTIME:-}"
if [[ -z "$LEGACY_RUNTIME_C" ]]; then
  if [[ "$BOOTSTRAP_BASE" == "vox_prev" || "$BOOTSTRAP_BASE" == "vox_prev.exe" ]]; then
    LEGACY_RUNTIME_C="1"
  else
    LEGACY_RUNTIME_C="0"
  fi
fi
echo "[release] bootstrap probe exit: $bootstrap_probe_rc"
echo "[release] legacy C runtime bridge: $LEGACY_RUNTIME_C"
echo "[release] host platform: ${HOST_OS}-${HOST_ARCH}"
echo "[release] target platform: $PLATFORM"

mkdir -p "$ROOT/target/release"
TOOL_BUILD_LOG="$ROOT/target/release/tool-build.log"
if [[ "$IS_CROSS" == "1" ]]; then
  TARGET_ARG="--target=${TARGET_OS}-${TARGET_ARCH}"
else
  TARGET_ARG=""
fi

set +e
(
  cd "$ROOT"
  if [[ -n "$TARGET_ARG" ]]; then
    if VOX_LEGACY_C_RUNTIME="$LEGACY_RUNTIME_C" "$BOOTSTRAP_BIN" build --driver=tool "$TARGET_ARG" target/release/vox; then
      :
    else
      VOX_LEGACY_C_RUNTIME="$LEGACY_RUNTIME_C" "$BOOTSTRAP_BIN" build-pkg --driver=tool "$TARGET_ARG" target/release/vox
    fi
  else
    if VOX_LEGACY_C_RUNTIME="$LEGACY_RUNTIME_C" "$BOOTSTRAP_BIN" build --driver=tool target/release/vox; then
      :
    else
      VOX_LEGACY_C_RUNTIME="$LEGACY_RUNTIME_C" "$BOOTSTRAP_BIN" build-pkg --driver=tool target/release/vox
    fi
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
if [[ -d "$ROOT/src/std" ]]; then
  mkdir -p "$BUNDLE_DIR/src"
  cp -R "$ROOT/src/std" "$BUNDLE_DIR/src/std"
fi
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
