#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
GOOS="$(go env GOOS)"

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
      echo "[smoke] using CC from env: $CC"
      return 0
    fi
    if [[ "$GOOS" == "windows" && "$cc_bin" =~ ^[A-Za-z]:\\ ]]; then
      echo "[smoke] using CC from env (absolute windows path): $CC"
      return 0
    fi
    echo "[smoke] CC is set but not found in PATH: $CC" >&2
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
      echo "[smoke] auto-detected CC: $CC"
      return 0
    fi
  done

  echo "[smoke] no C compiler found (checked: ${candidates[*]})" >&2
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

bootstrap_cc_env

STAGE2_BIN="$(resolve_bin "$ROOT/target/release/vox_stage2")"

TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/vox-release-smoke.XXXXXX")"
cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

mkdir -p "$TMP_DIR/src" "$TMP_DIR/tests" "$TMP_DIR/target"
cat > "$TMP_DIR/vox.toml" <<'TOML'
[package]
name = "smoke"
version = "0.1.0"
edition = "2026"
TOML
cat > "$TMP_DIR/src/main.vox" <<'VOX'
fn main() -> i32 { return 0; }
VOX
cat > "$TMP_DIR/tests/smoke_test.vox" <<'VOX'
import "std/testing" as t
fn test_basic() -> () { t.assert(true); }
VOX

echo "[smoke] stage2 build-pkg"
(
  cd "$TMP_DIR"
  "$STAGE2_BIN" build-pkg target/smoke_stage2
)

echo "[smoke] stage2 test-pkg"
(
  cd "$TMP_DIR"
  "$STAGE2_BIN" test-pkg --run='*test_basic*' target/smoke_stage2_tests
)

echo "[smoke] ok"
