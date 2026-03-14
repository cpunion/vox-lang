#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
COMPILER_BIN="${VOX_PROFILE_COMPILER_BIN:-$("$ROOT/scripts/ci/resolve-selfhost-bin.sh")}"
TMP_BASE="${TMPDIR:-/tmp}"
WORK_DIR="$(mktemp -d "$TMP_BASE/vox-cache-repo.XXXXXX")"
KEEP_WORK="${VOX_PROFILE_KEEP_WORKDIR:-0}"
QUERY_SHADOW="${VOX_QUERY_SHADOW:-}"
PROFILE_REF="${VOX_PROFILE_REPO_REF:-HEAD}"

cleanup() {
  if [[ "$KEEP_WORK" == "1" ]]; then
    echo "[cache-repo] kept workdir: $WORK_DIR"
    return
  fi
  rm -rf "$WORK_DIR"
}
trap cleanup EXIT

if [[ ! -x "$COMPILER_BIN" ]]; then
  echo "[cache-repo] invalid compiler bin: $COMPILER_BIN" >&2
  exit 1
fi

if ! git -C "$ROOT" rev-parse --verify "$PROFILE_REF" >/dev/null 2>&1; then
  echo "[cache-repo] invalid git ref: $PROFILE_REF" >&2
  exit 1
fi

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
  printf '[cache-repo] cache-files sem=%s obj=%s link=%s\n' "$sem_count" "$obj_count" "$link_count"
}

print_profile_excerpt() {
  local log="$1"
  rg '^\[profile\] (cache(|-prep)|compile total:|compile typecheck:|compile irgen:|compile codegen:|cc-obj:|link:)' "$log" || true
}

export_snapshot() {
  local dest="$1"
  mkdir -p "$dest"
  git -C "$ROOT" archive --format=tar "$PROFILE_REF" | tar -xf - -C "$dest"
}

run_build() {
  local label="$1"
  local dir="$2"
  local query_shadow="$3"
  local log="$dir/${label}.log"
  (
    cd "$dir"
    env \
      VOX_INCREMENTAL=1 \
      VOX_PROFILE=1 \
      VOX_QUERY_SHADOW="$query_shadow" \
      "$COMPILER_BIN" build --driver=tool target/debug/vox_repo_profile > "$log" 2>&1
  )
  printf '[cache-repo] %s\n' "$label"
  print_profile_excerpt "$log"
  print_cache_counts "$dir"
}

run_mode() {
  local query_shadow="$1"
  local mode_dir="$WORK_DIR/query-shadow-$query_shadow"
  local prefix="q${query_shadow}"
  export_snapshot "$mode_dir"
  echo "[cache-repo] query-shadow: $query_shadow"
  run_build "$prefix-cold-build" "$mode_dir" "$query_shadow"
  run_build "$prefix-warm-build" "$mode_dir" "$query_shadow"
}

echo "[cache-repo] compiler: $COMPILER_BIN"
echo "[cache-repo] ref: $(git -C "$ROOT" rev-parse --short "$PROFILE_REF") ($PROFILE_REF)"

if [[ "$QUERY_SHADOW" == "" ]]; then
  run_mode 0
  run_mode 1
else
  run_mode "$QUERY_SHADOW"
fi
