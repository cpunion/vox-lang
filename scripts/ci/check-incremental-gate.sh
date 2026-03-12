#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
COMPILER_BIN="$("$ROOT/scripts/ci/resolve-selfhost-bin.sh")"

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
