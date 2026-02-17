#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
WORK_DIR="$ROOT"
OUT_REL="${VOX_SELFHOST_OUT:-target/debug/vox_rolling}"
FORCE_REBUILD="${VOX_SELFHOST_FORCE_REBUILD:-0}"
BUILDINFO_FILE="$ROOT/src/vox/buildinfo/buildinfo.vox"
BUILDINFO_BACKUP=""

usage() {
  cat >&2 <<USAGE
usage: $(basename "$0") <build|test|print-bin>

modes:
  build      build compiler via rolling bootstrap
  test       build compiler, then run test smoke
  print-bin  build compiler and print its absolute path
USAGE
}

vox_escape_for_string() {
  local s="$1"
  s="${s//\\/\\\\}"
  s="${s//\"/\\\"}"
  printf '%s' "$s"
}

prepare_buildinfo() {
  if [[ ! -f "$BUILDINFO_FILE" ]]; then
    return 0
  fi

  local version="0.0.0"
  local channel="dev"
  local line=""

  line="$(grep -E '^pub const VERSION: String = "' "$BUILDINFO_FILE" | head -n 1 || true)"
  if [[ -n "$line" ]]; then
    version="${line#*\"}"
    version="${version%%\"*}"
  fi
  line="$(grep -E '^pub const CHANNEL: String = "' "$BUILDINFO_FILE" | head -n 1 || true)"
  if [[ -n "$line" ]]; then
    channel="${line#*\"}"
    channel="${channel%%\"*}"
  fi

  BUILDINFO_BACKUP="$(mktemp "${TMPDIR:-/tmp}/vox-buildinfo-selfhost.XXXXXX")"
  cp "$BUILDINFO_FILE" "$BUILDINFO_BACKUP"
  local root_escaped
  root_escaped="$(vox_escape_for_string "$ROOT")"
  cat > "$BUILDINFO_FILE" <<EOF_BUILDINFO
// Build metadata embedded in the compiler executable.
// Default channel is "dev" for repository builds; release scripts override this file.

pub const VERSION: String = "$version";
pub const CHANNEL: String = "$channel";
pub const BUILD_SOURCE_ROOT: String = "$root_escaped";

pub fn version() -> String { return VERSION; }
pub fn channel() -> String { return CHANNEL; }
pub fn build_source_root() -> String { return BUILD_SOURCE_ROOT; }
EOF_BUILDINFO
}

cleanup_buildinfo() {
  if [[ -n "$BUILDINFO_BACKUP" && -f "$BUILDINFO_BACKUP" ]]; then
    cp "$BUILDINFO_BACKUP" "$BUILDINFO_FILE"
    rm -f "$BUILDINFO_BACKUP"
  fi
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

bootstrap_fingerprint() {
  local bootstrap_bin="$1"
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

should_rebuild_selfhost() {
  local bootstrap_bin="$1"
  local out_path="$WORK_DIR/$OUT_REL"
  local cache_key_file="${out_path}.cache.key"
  local new_key
  local old_key=""

  mkdir -p "$(dirname "$cache_key_file")"
  new_key="$(selfhost_cache_key "$bootstrap_bin")"

  if [[ -f "$cache_key_file" ]]; then
    old_key="$(cat "$cache_key_file")"
  fi

  if [[ "$FORCE_REBUILD" == "1" ]]; then
    echo "$new_key" > "$cache_key_file"
    return 0
  fi
  if [[ ! -f "$out_path" && ! -f "${out_path}.exe" ]]; then
    echo "$new_key" > "$cache_key_file"
    return 0
  fi
  if [[ "$new_key" != "$old_key" ]]; then
    echo "$new_key" > "$cache_key_file"
    return 0
  fi
  return 1
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
  local help_text=""
  help_text="$("$bootstrap_bin" 2>&1 || true)"
  (
    cd "$WORK_DIR"
    # Bootstrap compatibility:
    # - old CLI: build-pkg <out.bin>
    # - new CLI: build [out.bin]
    if [[ "$help_text" == *"build-pkg <out.bin>"* ]]; then
      if "$bootstrap_bin" build-pkg --driver=tool "$OUT_REL"; then
        :
      else
        "$bootstrap_bin" build-pkg "$OUT_REL"
      fi
    else
      "$bootstrap_bin" build --driver=tool "$OUT_REL"
    fi
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

trap cleanup_buildinfo EXIT
prepare_buildinfo

echo "[selfhost] bootstrap: $BOOTSTRAP_BIN"
if should_rebuild_selfhost "$BOOTSTRAP_BIN"; then
  echo "[selfhost] rebuild: yes"
  build_from_bootstrap "$BOOTSTRAP_BIN"
else
  echo "[selfhost] rebuild: no (cache hit)"
fi

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
  "$SELF_BIN" test "--jobs=$JOBS" "--run=$RUN_GLOB" "$TEST_OUT_REL"
)

echo "[selfhost] ok"
