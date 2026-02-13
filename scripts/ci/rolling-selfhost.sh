#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
WORK_DIR="$ROOT"
OUT_REL="${VOX_SELFHOST_OUT:-target/debug/vox_rolling}"

usage() {
  cat >&2 <<USAGE
usage: $(basename "$0") <build|test|print-bin>

modes:
  build      build compiler via rolling bootstrap
  test       build compiler, then run test-pkg smoke
  print-bin  build compiler and print its absolute path
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

is_runnable_bin() {
  local p="$1"
  [[ -f "$p" ]] || return 1
  [[ -x "$p" ]] || return 1
  [[ "$p" == *.c ]] && return 1
  [[ "$p" == *.test ]] && return 1
  [[ "$p" == *.cache.key ]] && return 1
  [[ "$p" == *.log ]] && return 1
  return 0
}

pick_bootstrap() {
  if [[ -n "${VOX_BOOTSTRAP:-}" && -f "${VOX_BOOTSTRAP}" ]]; then
    printf '%s\n' "$VOX_BOOTSTRAP"
    return 0
  fi

  local candidates=(
    "$WORK_DIR/target/bootstrap/vox_prev"
    "$WORK_DIR/target/debug/vox_tool"
    "$WORK_DIR/target/debug/vox_rolling"
    "$WORK_DIR/target/debug/vox"
    "$WORK_DIR/target/release/vox"
  )

  local c
  local p
  for c in "${candidates[@]}"; do
    if p="$(resolve_bin "$c" 2>/dev/null)"; then
      printf '%s\n' "$p"
      return 0
    fi
  done

  for p in "$WORK_DIR"/target/debug/vox* "$WORK_DIR"/target/release/vox*; do
    if is_runnable_bin "$p"; then
      printf '%s\n' "$p"
      return 0
    fi
  done

  return 1
}

build_from_bootstrap() {
  local bootstrap_bin="$1"
  (
    cd "$WORK_DIR"
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
if ! BOOTSTRAP_BIN="$(pick_bootstrap)"; then
  echo "[selfhost] no bootstrap compiler binary found" >&2
  echo "[selfhost] set VOX_BOOTSTRAP or prepare target/bootstrap/vox_prev" >&2
  exit 1
fi

echo "[selfhost] bootstrap: $BOOTSTRAP_BIN"
build_from_bootstrap "$BOOTSTRAP_BIN"

SELF_BIN="$(resolve_bin "$WORK_DIR/$OUT_REL")"
echo "[selfhost] built: $SELF_BIN"

if [[ "$MODE" == "print-bin" ]]; then
  printf '%s\n' "$SELF_BIN"
  exit 0
fi

if [[ "$MODE" == "build" ]]; then
  exit 0
fi

RUN_GLOB="${VOX_TEST_RUN:-*std_sync_runtime_generic_api_smoke}"
JOBS="${VOX_TEST_JOBS:-8}"
TEST_OUT_REL="${VOX_TEST_OUT:-target/debug/vox.test}"

echo "[selfhost] test-pkg: run=$RUN_GLOB jobs=$JOBS"
(
  cd "$WORK_DIR"
  "$SELF_BIN" test-pkg "--jobs=$JOBS" "--run=$RUN_GLOB" "$TEST_OUT_REL"
)

echo "[selfhost] ok"
