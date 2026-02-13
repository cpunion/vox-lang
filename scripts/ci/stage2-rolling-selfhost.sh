#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
STAGE2_DIR="$ROOT/compiler/stage2"
OUT_REL="${VOX_STAGE2_SELFHOST_OUT:-target/debug/vox_stage2_rolling}"

usage() {
  cat >&2 <<USAGE
usage: $(basename "$0") <build|test|print-bin>

modes:
  build      build stage2 tool via rolling stage2 bootstrap
  test       build stage2 tool, then run stage2 test-pkg smoke
  print-bin  build stage2 tool and print its absolute path
USAGE
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
    "$STAGE2_DIR/target/bootstrap/vox_stage2_prev"
    "$STAGE2_DIR/target/debug/vox_stage2_b_tool"
    "$STAGE2_DIR/target/debug/vox_stage2_rolling"
    "$STAGE2_DIR/target/debug/vox_stage2"
    "$STAGE2_DIR/target/release/vox_stage2"
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

build_stage2_from_bootstrap() {
  local bootstrap_bin="$1"
  (
    cd "$STAGE2_DIR"
    "$bootstrap_bin" build-pkg --driver=tool "$OUT_REL"
  )
}

MODE="${1:-}"
case "$MODE" in
  build|test|print-bin)
    ;;
  *)
    usage
    exit 1
    ;;
esac

BOOTSTRAP_BIN=""
if ! BOOTSTRAP_BIN="$(pick_bootstrap_stage2)"; then
  echo "[stage2-selfhost] no rolling bootstrap stage2 binary found" >&2
  echo "[stage2-selfhost] set VOX_BOOTSTRAP_STAGE2 or prepare compiler/stage2/target/bootstrap/vox_stage2_prev" >&2
  exit 1
fi

echo "[stage2-selfhost] bootstrap: $BOOTSTRAP_BIN"
build_stage2_from_bootstrap "$BOOTSTRAP_BIN"

SELF_BIN="$(resolve_bin "$STAGE2_DIR/$OUT_REL")"
echo "[stage2-selfhost] built: $SELF_BIN"

if [[ "$MODE" == "print-bin" ]]; then
  printf '%s\n' "$SELF_BIN"
  exit 0
fi

if [[ "$MODE" == "build" ]]; then
  exit 0
fi

RUN_GLOB="${VOX_STAGE2_TEST_RUN:-*std_sync_runtime_generic_api_smoke}"
JOBS="${VOX_STAGE2_TEST_JOBS:-8}"
TEST_OUT_REL="${VOX_STAGE2_TEST_OUT:-target/debug/vox_stage2.test}"

echo "[stage2-selfhost] test-pkg: run=$RUN_GLOB jobs=$JOBS"
(
  cd "$STAGE2_DIR"
  "$SELF_BIN" test-pkg "--jobs=$JOBS" "--run=$RUN_GLOB" "$TEST_OUT_REL"
)

echo "[stage2-selfhost] ok"
