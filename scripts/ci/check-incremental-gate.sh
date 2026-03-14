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

mk_project() {
  local dir="$1"
  mkdir -p "$dir/src" "$dir/tests"
  cat > "$dir/vox.toml" <<'TOML'
[package]
name = "incr_gate"
version = "0.1.0"
edition = "2026"

[dependencies]
TOML
  cat > "$dir/src/main.vox" <<'VOX'
fn main() -> i32 { return 0; }
VOX
  cat > "$dir/tests/smoke_test.vox" <<'VOX'
fn test_smoke() -> () {}
VOX
}

require_no_file() {
  local path="$1"
  local why="$2"
  if [[ -e "$path" ]]; then
    echo "[incremental-gate] unexpected file exists: $path ($why)" >&2
    exit 1
  fi
}

require_any_file_in_dir() {
  local dir="$1"
  local why="$2"
  if [[ ! -d "$dir" ]]; then
    echo "[incremental-gate] expected directory missing: $dir ($why)" >&2
    exit 1
  fi
  if ! find "$dir" -type f | grep -q .; then
    echo "[incremental-gate] expected files under directory: $dir ($why)" >&2
    exit 1
  fi
}

INCR0="$WORK_DIR/incr0"
INCR1="$WORK_DIR/incr1"
INCR1_FAST="$WORK_DIR/incr1-fast"
mk_project "$INCR0"
mk_project "$INCR1"
mk_project "$INCR1_FAST"

(
  cd "$INCR0"

  VOX_INCREMENTAL=0 VOX_PROFILE=1 VOX_QUERY_SHADOW=1 VOX_QUERY_SHADOW_TRACE=1 VOX_CACHE_TRACE=1 "$COMPILER_BIN" build --driver=tool > build.log 2>&1
  VOX_INCREMENTAL=0 VOX_PROFILE=1 VOX_QUERY_SHADOW=1 VOX_QUERY_SHADOW_TRACE=1 VOX_CACHE_TRACE=1 "$COMPILER_BIN" test --list > list.log 2>&1

  require_no_file "target/debug/.vox_test_discover.key" "VOX_INCREMENTAL=0 should disable test discovery sidecar writes"
  require_no_file "target/debug/.vox_test_discover.list" "VOX_INCREMENTAL=0 should disable test discovery sidecar writes"
  if [[ -d target/cache ]]; then
    if find target/cache -type f | grep -q .; then
      echo "[incremental-gate] expected no target/cache artifacts when VOX_INCREMENTAL=0" >&2
      exit 1
    fi
  fi
  if rg -n "\[query-shadow\]" build.log list.log >/dev/null 2>&1; then
    echo "[incremental-gate] query-shadow logs should be absent when VOX_INCREMENTAL=0" >&2
    exit 1
  fi
  if ! rg -n "\[profile\] cache build: cache=off incremental=off" build.log >/dev/null 2>&1; then
    echo "[incremental-gate] expected profile cache build summary for VOX_INCREMENTAL=0" >&2
    exit 1
  fi
  if ! rg -n "\[profile\] cache test-list: cache=off incremental=off" list.log >/dev/null 2>&1; then
    echo "[incremental-gate] expected profile cache test-list summary for VOX_INCREMENTAL=0" >&2
    exit 1
  fi
)

(
  cd "$INCR1"

  VOX_INCREMENTAL=1 VOX_PROFILE=1 VOX_QUERY_SHADOW=1 VOX_QUERY_SHADOW_TRACE=1 VOX_CACHE_TRACE=1 "$COMPILER_BIN" build --driver=tool > build1.log 2>&1
  VOX_INCREMENTAL=1 VOX_PROFILE=1 VOX_QUERY_SHADOW=1 VOX_QUERY_SHADOW_TRACE=1 VOX_CACHE_TRACE=1 "$COMPILER_BIN" build --driver=tool > build2.log 2>&1
  VOX_INCREMENTAL=1 VOX_PROFILE=1 VOX_QUERY_SHADOW=1 VOX_QUERY_SHADOW_TRACE=1 VOX_CACHE_TRACE=1 "$COMPILER_BIN" test --list > list.log 2>&1

  if [[ ! -e target/debug/.vox_test_discover.key || ! -e target/debug/.vox_test_discover.list ]]; then
    echo "[incremental-gate] expected test discovery sidecars when VOX_INCREMENTAL=1" >&2
    exit 1
  fi
  require_any_file_in_dir "target/cache/pkg-sem-v1" "VOX_INCREMENTAL=1 should enable semantic cache artifacts"
  require_any_file_in_dir "target/cache/pkg-obj-v1" "VOX_INCREMENTAL=1 should enable object cache artifacts"
  require_any_file_in_dir "target/cache/link-v1" "VOX_INCREMENTAL=1 should enable link cache artifacts"

  if ! rg -n "\[cache\] mode=build cache=on incremental=on" build1.log build2.log >/dev/null 2>&1; then
    echo "[incremental-gate] expected cache=on incremental=on trace line for VOX_INCREMENTAL=1 build" >&2
    exit 1
  fi
  if ! rg -n "\[profile\] cache build: cache=on incremental=on" build1.log build2.log >/dev/null 2>&1; then
    echo "[incremental-gate] expected profile cache build summary for VOX_INCREMENTAL=1 build" >&2
    exit 1
  fi
  if ! rg -n "\[profile\] cache build: cache=on incremental=on sem=miss c=miss obj=miss link=miss" build1.log >/dev/null 2>&1; then
    echo "[incremental-gate] expected cold query-shadow build to report sem=miss with cold cache state" >&2
    exit 1
  fi
  if ! rg -n "\[profile\] cache build: cache=on incremental=on sem=hit c=hit obj=hit link=hit" build2.log >/dev/null 2>&1; then
    echo "[incremental-gate] expected warm query-shadow build to report sem=hit with cache hits" >&2
    exit 1
  fi
  if ! rg -n "\[profile\] cache test-list: cache=on incremental=on" list.log >/dev/null 2>&1; then
    echo "[incremental-gate] expected profile cache test-list summary for VOX_INCREMENTAL=1 list" >&2
    exit 1
  fi
  if ! rg -n "\[query-shadow\] phase=build" build1.log build2.log >/dev/null 2>&1; then
    echo "[incremental-gate] expected query-shadow build trace when VOX_QUERY_SHADOW=1 and VOX_INCREMENTAL=1" >&2
    exit 1
  fi
  if [[ ! -d target/cache/pkg-sem-v1/parse-load-shadow-v1 ]]; then
    echo "[incremental-gate] expected parse-load-shadow artifacts when VOX_QUERY_SHADOW=1 and VOX_INCREMENTAL=1" >&2
    exit 1
  fi
)

(
  cd "$INCR1_FAST"

  VOX_INCREMENTAL=1 VOX_PROFILE=1 VOX_CACHE_TRACE=1 "$COMPILER_BIN" build --driver=tool > build1.log 2>&1
  VOX_INCREMENTAL=1 VOX_PROFILE=1 VOX_CACHE_TRACE=1 "$COMPILER_BIN" build --driver=tool > build2.log 2>&1

  if ! rg -n "\[profile\] cache build: cache=on incremental=on sem=miss c=miss obj=miss link=miss" build1.log >/dev/null 2>&1; then
    echo "[incremental-gate] expected cold non-query-shadow build to report sem=miss with cold cache state" >&2
    exit 1
  fi
  if ! rg -n "\[profile\] cache build: cache=on incremental=on sem=skip c=hit obj=hit link=hit" build2.log >/dev/null 2>&1; then
    echo "[incremental-gate] expected warm non-query-shadow build to report sem=skip with cache hits" >&2
    exit 1
  fi
)

echo "[incremental-gate] ok: VOX_INCREMENTAL gates build/test/list cache + query-shadow behavior"
