#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
if ! COMPILER_BIN_RAW="$("$ROOT/scripts/ci/rolling-selfhost.sh" print-bin)"; then
  exit $?
fi
COMPILER_BIN="$(printf '%s\n' "$COMPILER_BIN_RAW" | tail -n 1)"
if [[ ! -x "$COMPILER_BIN" ]]; then
  echo "[incremental-gate] invalid compiler bin: $COMPILER_BIN" >&2
  exit 1
fi

TMP_BASE="${TMPDIR:-/tmp}"
WORK_DIR="$(mktemp -d "$TMP_BASE/vox-incremental-gate.XXXXXX")"
cleanup() {
  rm -rf "$WORK_DIR"
}
trap cleanup EXIT

mkdir -p "$WORK_DIR/src" "$WORK_DIR/tests"
cat > "$WORK_DIR/src/main.vox" <<'VOX'
fn main() -> i32 { return 0; }
VOX
cat > "$WORK_DIR/tests/smoke_test.vox" <<'VOX'
fn test_smoke() -> () {}
VOX

(
  cd "$WORK_DIR"

  VOX_INCREMENTAL=0 VOX_CACHE_TRACE=1 "$COMPILER_BIN" test --list >/dev/null
  if [[ -e target/debug/.vox_test_discover.key || -e target/debug/.vox_test_discover.list ]]; then
    echo "[incremental-gate] expected no discovery cache sidecars when VOX_INCREMENTAL=0" >&2
    exit 1
  fi

  VOX_INCREMENTAL=1 VOX_CACHE_TRACE=1 "$COMPILER_BIN" test --list >/dev/null
  if [[ ! -e target/debug/.vox_test_discover.key || ! -e target/debug/.vox_test_discover.list ]]; then
    echo "[incremental-gate] expected discovery cache sidecars when VOX_INCREMENTAL=1" >&2
    exit 1
  fi
)

echo "[incremental-gate] ok: VOX_INCREMENTAL gates test discovery cache reuse"
