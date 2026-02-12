#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

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

STAGE0_BIN="$(resolve_bin "$ROOT/compiler/stage0/target/release/vox-stage0")"
STAGE1_BIN="$(resolve_bin "$ROOT/compiler/stage1/target/release/vox_stage1")"
STAGE2_BIN="$(resolve_bin "$ROOT/compiler/stage2/target/release/vox_stage2")"

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

echo "[smoke] stage0 test (interp)"
"$STAGE0_BIN" test --engine=interp --run=test_basic "$TMP_DIR"

echo "[smoke] stage1 test-pkg"
(
  cd "$TMP_DIR"
  "$STAGE1_BIN" test-pkg target/smoke_stage1
)

echo "[smoke] stage2 test-pkg"
(
  cd "$TMP_DIR"
  "$STAGE2_BIN" test-pkg --run='*test_basic*' target/smoke_stage2
)

echo "[smoke] ok"
