#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
COMPILER_BIN="${VOX_PROFILE_COMPILER_BIN:-$("$ROOT/scripts/ci/resolve-selfhost-bin.sh")}"
TMP_BASE="${TMPDIR:-/tmp}"
WORK_DIR="$(mktemp -d "$TMP_BASE/vox-cache-matrix.XXXXXX")"
KEEP_WORK="${VOX_PROFILE_KEEP_WORKDIR:-0}"
QUERY_SHADOW="${VOX_QUERY_SHADOW:-0}"

cleanup() {
  if [[ "$KEEP_WORK" == "1" ]]; then
    echo "[cache-matrix] kept workdir: $WORK_DIR"
    return
  fi
  rm -rf "$WORK_DIR"
}
trap cleanup EXIT

if [[ ! -x "$COMPILER_BIN" ]]; then
  echo "[cache-matrix] invalid compiler bin: $COMPILER_BIN" >&2
  exit 1
fi

mk_project() {
  local dir="$1"
  mkdir -p "$dir/src" "$dir/tests"
  cat > "$dir/vox.toml" <<'TOML'
[package]
name = "cache_matrix"
version = "0.1.0"
edition = "2026"

[dependencies]
TOML
  cat > "$dir/src/main.vox" <<'VOX'
fn helper() -> i32 { return 41; }

fn main() -> i32 { return helper() + 1; }
VOX
  cat > "$dir/tests/smoke_test.vox" <<'VOX'
fn test_smoke() -> () {}
VOX
}

cache_file_count() {
  local dir="$1"
  if [[ ! -d "$dir" ]]; then
    printf '0'
    return
  fi
  find "$dir" -type f | wc -l | awk '{print $1}'
}

print_cache_counts() {
  local dir="$1"
  local sem_count obj_count link_count
  sem_count="$(cache_file_count "$dir/target/cache/pkg-sem-v1")"
  obj_count="$(cache_file_count "$dir/target/cache/pkg-obj-v1")"
  link_count="$(cache_file_count "$dir/target/cache/link-v1")"
  printf '[cache-matrix] cache-files sem=%s obj=%s link=%s\n' "$sem_count" "$obj_count" "$link_count"
}

run_profile() {
  local label="$1"
  local cmd="$2"
  local dir="$3"
  local log="$dir/${label}.log"
  (
    cd "$dir"
    env \
      VOX_INCREMENTAL=1 \
      VOX_PROFILE=1 \
      VOX_QUERY_SHADOW="$QUERY_SHADOW" \
      "$COMPILER_BIN" $cmd > "$log" 2>&1
  )
  printf '[cache-matrix] %s\n' "$label"
  rg '^\[profile\] cache(|-prep) ' "$log" || true
  print_cache_counts "$dir"
}

PROJECT_DIR="$WORK_DIR/project"
mk_project "$PROJECT_DIR"

echo "[cache-matrix] compiler: $COMPILER_BIN"
echo "[cache-matrix] query-shadow: $QUERY_SHADOW"

run_profile "cold-build" "build --driver=tool" "$PROJECT_DIR"
run_profile "warm-build" "build --driver=tool" "$PROJECT_DIR"
run_profile "cold-list" "list" "$PROJECT_DIR"
run_profile "warm-list" "list" "$PROJECT_DIR"
run_profile "cold-test-list" "test --list" "$PROJECT_DIR"
run_profile "warm-test-list" "test --list" "$PROJECT_DIR"
