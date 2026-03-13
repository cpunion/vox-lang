#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
HELPER="$ROOT/scripts/ci/resolve-legacy-runtime-c.sh"
ROLLING="$ROOT/scripts/ci/rolling-selfhost.sh"
RELEASE="$ROOT/scripts/release/build-release-bundle.sh"

if [[ ! -x "$HELPER" ]]; then
  echo "[legacy-runtime-switch] helper not executable: $HELPER" >&2
  exit 1
fi

expect_eq() {
  local got="$1"
  local want="$2"
  local msg="$3"
  if [[ "$got" != "$want" ]]; then
    echo "[legacy-runtime-switch] $msg: expected '$want', got '$got'" >&2
    exit 1
  fi
}

expect_eq "$("$HELPER" vox_prev)" "1" "vox_prev default"
expect_eq "$("$HELPER" vox_prev.exe)" "1" "vox_prev.exe default"
expect_eq "$("$HELPER" vox_rolling)" "0" "rolling default"
expect_eq "$(VOX_LEGACY_C_RUNTIME=on "$HELPER" vox_rolling)" "1" "env on override"
expect_eq "$(VOX_LEGACY_C_RUNTIME=true "$HELPER" vox_rolling)" "1" "env true override"
expect_eq "$(VOX_LEGACY_C_RUNTIME=off "$HELPER" vox_prev)" "0" "env off override"
expect_eq "$(VOX_LEGACY_C_RUNTIME=0 "$HELPER" vox_prev)" "0" "env 0 override"

if ! rg -q "resolve-legacy-runtime-c\\.sh" "$ROLLING"; then
  echo "[legacy-runtime-switch] rolling-selfhost missing helper usage" >&2
  exit 1
fi
if ! rg -q "resolve-legacy-runtime-c\\.sh" "$RELEASE"; then
  echo "[legacy-runtime-switch] release bundle script missing helper usage" >&2
  exit 1
fi

if rg -q "VOX_LEGACY_C_RUNTIME:-" "$ROLLING" "$RELEASE"; then
  echo "[legacy-runtime-switch] found duplicated VOX_LEGACY_C_RUNTIME resolution logic" >&2
  exit 1
fi

echo "[legacy-runtime-switch] ok: helper semantics and script reuse validated"
