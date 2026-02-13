#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat >&2 <<USAGE
usage: $(basename "$0") <version>
example: $(basename "$0") v0.2.0-rc1
USAGE
}

if [[ $# -ne 1 ]]; then
  usage
  exit 1
fi

VERSION="$1"
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
    "$ROOT/target/release/vox"
    "$ROOT/target/debug/vox"
    "$ROOT/target/bootstrap/vox_prev"
    "$ROOT/target/release/vox_stage2"
    "$ROOT/target/debug/vox_stage2"
    "$ROOT/target/bootstrap/vox_stage2_prev"
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

BOOTSTRAP_BIN=""
if ! BOOTSTRAP_BIN="$(pick_bootstrap)"; then
  echo "[dry-run] no rolling bootstrap compiler binary found" >&2
  echo "[dry-run] set VOX_BOOTSTRAP or prepare target/bootstrap/vox_prev" >&2
  exit 1
fi

echo "[dry-run] using rolling bootstrap binary: $BOOTSTRAP_BIN"

export VOX_BOOTSTRAP="$BOOTSTRAP_BIN"

"$ROOT/scripts/release/build-release-bundle.sh" "$VERSION"
"$ROOT/scripts/release/smoke-toolchains.sh"
"$ROOT/scripts/release/verify-release-bundle.sh" "$VERSION"

echo "[dry-run] release rolling bootstrap gate passed for version=$VERSION"
