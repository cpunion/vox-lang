#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat >&2 <<USAGE
usage: $(basename "$0") <version>
example: $(basename "$0") v0.1.0-rc1
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

pick_bootstrap_stage2() {
  if [[ -n "${VOX_BOOTSTRAP_STAGE2:-}" && -f "${VOX_BOOTSTRAP_STAGE2}" ]]; then
    printf '%s\n' "$VOX_BOOTSTRAP_STAGE2"
    return 0
  fi

  local candidates=(
    "$ROOT/compiler/stage2/target/release/vox_stage2"
    "$ROOT/compiler/stage2/target/debug/vox_stage2"
    "$ROOT/compiler/stage2/target/bootstrap/vox_stage2_prev"
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
if ! BOOTSTRAP_BIN="$(pick_bootstrap_stage2)"; then
  echo "[dry-run] no existing stage2 bootstrap binary found; build one via stage1"
  (
    cd "$ROOT/compiler/stage0"
    go run ./cmd/vox build-tool ../stage1
  )

  STAGE1_TOOL="$(resolve_bin "$ROOT/compiler/stage1/target/debug/vox_stage1")"
  mkdir -p "$ROOT/compiler/stage2/target/bootstrap"
  (
    cd "$ROOT/compiler/stage2"
    "$STAGE1_TOOL" build-pkg --driver=tool target/bootstrap/vox_stage2_prev
  )
  BOOTSTRAP_BIN="$(resolve_bin "$ROOT/compiler/stage2/target/bootstrap/vox_stage2_prev")"
fi

echo "[dry-run] using rolling bootstrap binary: $BOOTSTRAP_BIN"

export VOX_BOOTSTRAP_STAGE2="$BOOTSTRAP_BIN"
export VOX_REQUIRE_ROLLING_BOOTSTRAP=1

"$ROOT/scripts/release/build-release-bundle.sh" "$VERSION"
"$ROOT/scripts/release/smoke-toolchains.sh"
"$ROOT/scripts/release/verify-release-bundle.sh" "$VERSION"

echo "[dry-run] release rolling bootstrap gate passed for version=$VERSION"
