#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
WORK_DIR="$ROOT"
OUT_REL="${VOX_SELFHOST_OUT:-target/debug/vox_rolling}"
FORCE_REBUILD="${VOX_SELFHOST_FORCE_REBUILD:-0}"
INCREMENTAL_RAW="${VOX_INCREMENTAL:-1}"

# Some generated stage binaries can exceed default thread stack limits on
# macOS during rolling bootstrap. Best-effort raise stack limit.
if ! ulimit -s unlimited >/dev/null 2>&1; then
  ulimit -s 65520 >/dev/null 2>&1 || true
fi

usage() {
  cat >&2 <<USAGE
usage: $(basename "$0") <build|test|print-bin>

modes:
  build      build compiler via rolling bootstrap
  test       build compiler, then run test smoke
  print-bin  build compiler and print its absolute path
USAGE
}

incremental_enabled() {
  case "$INCREMENTAL_RAW" in
    0|false|FALSE|False|off|OFF|Off|no|NO|No)
      return 1
      ;;
    *)
      return 0
      ;;
  esac
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

sha256_text() {
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 | awk '{print $1}'
    return 0
  fi
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum | awk '{print $1}'
    return 0
  fi
  cksum | awk '{print $1}'
}

abs_path() {
  local p="$1"
  if [[ "$p" == /* ]]; then
    printf '%s\n' "$p"
    return 0
  fi
  local dir
  local base
  dir="$(dirname "$p")"
  base="$(basename "$p")"
  local resolved
  resolved="$(
    cd "$dir" >/dev/null 2>&1 && printf '%s/%s\n' "$(pwd -P)" "$base"
  )"
  if [[ -n "$resolved" ]]; then
    printf '%s\n' "$resolved"
    return 0
  fi
  printf '%s\n' "$p"
}

is_self_bootstrap_path() {
  local b_abs
  local out_abs
  local out_exe_abs
  b_abs="$(abs_path "$1")"
  out_abs="$(abs_path "$WORK_DIR/$OUT_REL")"
  out_exe_abs="$(abs_path "$WORK_DIR/${OUT_REL}.exe")"
  [[ "$b_abs" == "$out_abs" || "$b_abs" == "$out_exe_abs" ]]
}

bootstrap_fingerprint() {
  local bootstrap_bin="$1"
  if is_self_bootstrap_path "$bootstrap_bin"; then
    # Self-bootstrap output path itself changes mtime/size on rebuild.
    # Keep a stable key component so no-op source trees can hit cache.
    printf 'self:%s\n' "$OUT_REL"
    return 0
  fi
  local size=0
  local mtime=0
  if [[ "$(uname -s)" == "Darwin" ]]; then
    size="$(stat -f '%z' "$bootstrap_bin" 2>/dev/null || echo 0)"
    mtime="$(stat -f '%m' "$bootstrap_bin" 2>/dev/null || echo 0)"
  else
    size="$(stat -c '%s' "$bootstrap_bin" 2>/dev/null || echo 0)"
    mtime="$(stat -c '%Y' "$bootstrap_bin" 2>/dev/null || echo 0)"
  fi
  printf '%s|%s|%s\n' "$bootstrap_bin" "$size" "$mtime"
}

selfhost_cache_key() {
  local bootstrap_bin="$1"
  (
    cd "$WORK_DIR"
    {
      printf 'bootstrap=%s\n' "$(bootstrap_fingerprint "$bootstrap_bin")"
      printf 'out=%s\n' "$OUT_REL"
      if [[ -f "vox.toml" ]]; then
        printf 'vox.toml\n'
        cat "vox.toml"
      fi
      find src -type f -name '*.vox' | LC_ALL=C sort | while IFS= read -r f; do
        printf 'file=%s\n' "$f"
        cat "$f"
      done
    } | sha256_text
  )
}

has_newer_selfhost_inputs() {
  local cache_key_file="$1"
  local bootstrap_bin="$2"
  (
    cd "$WORK_DIR"
    if [[ -f "vox.toml" && "vox.toml" -nt "$cache_key_file" ]]; then return 0; fi
    if find src -type f -name '*.vox' -newer "$cache_key_file" -print -quit | grep -q .; then return 0; fi
    return 1
  )
  local newer_inputs_status=$?
  if [[ "$newer_inputs_status" == "0" ]]; then return 0; fi
  if ! is_self_bootstrap_path "$bootstrap_bin" && [[ "$bootstrap_bin" -nt "$cache_key_file" ]]; then return 0; fi
  return 1
}

has_dirty_selfhost_inputs() {
  if ! command -v git >/dev/null 2>&1; then return 1; fi
  if ! git -C "$WORK_DIR" rev-parse --is-inside-work-tree >/dev/null 2>&1; then return 1; fi
  if ! git -C "$WORK_DIR" diff --quiet -- src vox.toml; then return 0; fi
  if ! git -C "$WORK_DIR" diff --cached --quiet -- src vox.toml; then return 0; fi
  return 1
}

should_rebuild_selfhost() {
  local bootstrap_bin="$1"
  local out_path="$WORK_DIR/$OUT_REL"
  local cache_key_file="${out_path}.cache.key"
  if [[ "$FORCE_REBUILD" == "1" ]]; then return 0; fi
  if ! incremental_enabled; then return 0; fi
  if [[ ! -f "$out_path" && ! -f "${out_path}.exe" ]]; then return 0; fi
  if [[ ! -f "$cache_key_file" ]]; then return 0; fi

  # Fast path: when none of the key inputs are newer than the key stamp,
  # the hash cannot change and we can skip the full-tree hash walk.
  if ! has_newer_selfhost_inputs "$cache_key_file" "$bootstrap_bin"; then
    # On dirty trees, mtime-only checks can miss same-timestamp edits.
    # Fall back to full hash validation instead of taking the fast path.
    if ! has_dirty_selfhost_inputs; then return 1; fi
  fi

  local new_key
  local old_key=""
  new_key="$(selfhost_cache_key "$bootstrap_bin")"
  old_key="$(cat "$cache_key_file")"
  if [[ "$new_key" != "$old_key" ]]; then
    return 0
  fi
  return 1
}

write_selfhost_cache_key() {
  local bootstrap_bin="$1"
  if ! incremental_enabled; then return 0; fi
  local out_path="$WORK_DIR/$OUT_REL"
  local cache_key_file="${out_path}.cache.key"
  local cache_dir
  local tmp_file
  local new_key
  cache_dir="$(dirname "$cache_key_file")"
  mkdir -p "$cache_dir"
  tmp_file="$(mktemp "${cache_key_file}.tmp.XXXXXX")"
  new_key="$(selfhost_cache_key "$bootstrap_bin")"
  printf '%s\n' "$new_key" > "$tmp_file"
  mv "$tmp_file" "$cache_key_file"
}

bootstrap_candidates() {
  local explicit="${VOX_BOOTSTRAP:-}"
  if [[ -n "$explicit" ]]; then
    local ep=""
    if ep="$(resolve_bin "$explicit" 2>/dev/null)"; then
      printf '%s\n' "$ep"
    fi
  fi

  local candidates=(
    "$WORK_DIR/target/debug/vox_rolling"
    "$WORK_DIR/target/debug/vox_tool"
    "$WORK_DIR/target/bootstrap/vox_prev"
    "$WORK_DIR/target/debug/vox"
    "$WORK_DIR/target/release/vox"
  )

  local c
  local p
  for c in "${candidates[@]}"; do
    if p="$(resolve_bin "$c" 2>/dev/null)"; then
      printf '%s\n' "$p"
    fi
  done
}

pick_legacy_bootstrap() {
  local p
  if p="$(resolve_bin "$WORK_DIR/target/bootstrap/vox_prev" 2>/dev/null)"; then
    printf '%s\n' "$p"
    return 0
  fi
  return 1
}

build_from_bootstrap() {
  local bootstrap_bin="$1"
  local bootstrap_base=""
  bootstrap_base="$(basename "$bootstrap_bin")"
  local is_legacy_bootstrap=0
  case "$bootstrap_base" in
    vox_rolling|vox_rolling.exe|vox_tool|vox_tool.exe|vox|vox.exe)
      is_legacy_bootstrap=0
      ;;
    *)
      local help_text=""
      help_text="$("$bootstrap_bin" 2>&1 || true)"
      if [[ "$help_text" == *"build-pkg <out.bin>"* ]]; then
        is_legacy_bootstrap=1
      fi
      ;;
  esac
  local legacy_runtime_c="${VOX_LEGACY_C_RUNTIME:-}"
  if [[ -z "$legacy_runtime_c" ]]; then
    # Locked bootstrap binaries named vox_prev still require runtime C bridge.
    if [[ "$bootstrap_base" == "vox_prev" || "$bootstrap_base" == "vox_prev.exe" ]]; then
      legacy_runtime_c=1
    else
      legacy_runtime_c=0
    fi
  fi
  local bootstrap_cc="${CC:-}"
  (
    cd "$WORK_DIR"
    if ! ulimit -s unlimited >/dev/null 2>&1; then
      ulimit -s 65520 >/dev/null 2>&1 || true
    fi
    # Bootstrap compatibility:
    # - old CLI: build-pkg <out.bin>
    # - new CLI: build [out.bin]
    if [[ "$is_legacy_bootstrap" == "1" ]]; then
      if [[ -n "$bootstrap_cc" ]]; then
        if VOX_LEGACY_C_RUNTIME="$legacy_runtime_c" CC="$bootstrap_cc" "$bootstrap_bin" build-pkg --driver=tool "$OUT_REL"; then
          :
        else
          VOX_LEGACY_C_RUNTIME="$legacy_runtime_c" CC="$bootstrap_cc" "$bootstrap_bin" build-pkg "$OUT_REL"
        fi
      elif VOX_LEGACY_C_RUNTIME="$legacy_runtime_c" "$bootstrap_bin" build-pkg --driver=tool "$OUT_REL"; then
        :
      else
        VOX_LEGACY_C_RUNTIME="$legacy_runtime_c" "$bootstrap_bin" build-pkg "$OUT_REL"
      fi
    else
      if [[ -n "$bootstrap_cc" ]]; then
        VOX_LEGACY_C_RUNTIME="$legacy_runtime_c" CC="$bootstrap_cc" "$bootstrap_bin" build --driver=tool "$OUT_REL"
      else
        VOX_LEGACY_C_RUNTIME="$legacy_runtime_c" "$bootstrap_bin" build --driver=tool "$OUT_REL"
      fi
    fi
  )
}

compiler_smoke_ok() {
  local bin="$1"
  if "$bin" --version >/dev/null 2>&1; then
    return 0
  fi
  return 1
}

rebuild_with_fallback() {
  local preferred_bootstrap="$1"
  local build_ok=0
  local active_bootstrap="$preferred_bootstrap"

  if build_from_bootstrap "$active_bootstrap"; then
    build_ok=1
  else
    while IFS= read -r ALT_BOOTSTRAP; do
      [[ -z "$ALT_BOOTSTRAP" ]] && continue
      [[ "$ALT_BOOTSTRAP" == "$active_bootstrap" ]] && continue
      echo "[selfhost] retry with bootstrap candidate: $ALT_BOOTSTRAP"
      if build_from_bootstrap "$ALT_BOOTSTRAP"; then
        active_bootstrap="$ALT_BOOTSTRAP"
        build_ok=1
        break
      fi
    done <<< "$BOOTSTRAP_CANDIDATES"
  fi

  if [[ "$build_ok" != "1" ]]; then
    local legacy_bootstrap=""
    if legacy_bootstrap="$(pick_legacy_bootstrap)" && [[ "$legacy_bootstrap" != "$active_bootstrap" ]]; then
      echo "[selfhost] retry with legacy bootstrap: $legacy_bootstrap"
      if build_from_bootstrap "$legacy_bootstrap"; then
        active_bootstrap="$legacy_bootstrap"
        build_ok=1
      fi
    fi
  fi

  if [[ "$build_ok" != "1" ]]; then
    return 1
  fi

  BOOTSTRAP_BIN="$active_bootstrap"
  write_selfhost_cache_key "$BOOTSTRAP_BIN"
  return 0
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

BOOTSTRAP_CANDIDATES_RAW="$(bootstrap_candidates)"
BOOTSTRAP_CANDIDATES="$(printf '%s\n' "$BOOTSTRAP_CANDIDATES_RAW" | awk '!seen[$0]++ { print; }')"
BOOTSTRAP_BIN=""
while IFS= read -r CANDIDATE; do
  [[ -z "$CANDIDATE" ]] && continue
  BOOTSTRAP_BIN="$CANDIDATE"
  break
done <<< "$BOOTSTRAP_CANDIDATES"
if [[ -z "$BOOTSTRAP_BIN" ]]; then
  echo "[selfhost] no bootstrap compiler binary found" >&2
  echo "[selfhost] set VOX_BOOTSTRAP or prepare target/bootstrap/vox_prev" >&2
  exit 1
fi

echo "[selfhost] bootstrap: $BOOTSTRAP_BIN"
if should_rebuild_selfhost "$BOOTSTRAP_BIN"; then
  echo "[selfhost] rebuild: yes"
  rebuild_with_fallback "$BOOTSTRAP_BIN"
else
  echo "[selfhost] rebuild: no (cache hit)"
fi

SELF_BIN="$(resolve_bin "$WORK_DIR/$OUT_REL")"
if ! compiler_smoke_ok "$SELF_BIN"; then
  echo "[selfhost] cached output failed smoke check; rebuilding"
  rebuild_with_fallback "$BOOTSTRAP_BIN"
  SELF_BIN="$(resolve_bin "$WORK_DIR/$OUT_REL")"
  if ! compiler_smoke_ok "$SELF_BIN"; then
    echo "[selfhost] rebuilt output still failed smoke check: $SELF_BIN" >&2
    exit 1
  fi
fi
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
TEST_OUT_REL="${VOX_TEST_OUT:-}"
if [[ -z "$TEST_OUT_REL" ]]; then
  # Default output path is derived from the run glob to avoid cache races when
  # multiple test invocations run concurrently.
  TEST_HASH="$(printf '%s' "$RUN_GLOB" | sha256_text | cut -c1-12)"
  TEST_OUT_REL="target/debug/vox.test.${TEST_HASH}"
fi

echo "[selfhost] test: run=$RUN_GLOB jobs=$JOBS"
(
  cd "$WORK_DIR"
  if ! ulimit -s unlimited >/dev/null 2>&1; then
    ulimit -s 65520 >/dev/null 2>&1 || true
  fi
  "$SELF_BIN" test "--jobs=$JOBS" "--run=$RUN_GLOB" "$TEST_OUT_REL"
)

echo "[selfhost] ok"
