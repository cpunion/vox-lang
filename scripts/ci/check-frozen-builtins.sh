#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
LOCK_FILE="$ROOT/scripts/release/frozen-builtins.lock"
SRC_FILE="$ROOT/src/vox/typecheck/collect.vox"

if [[ ! -f "$LOCK_FILE" ]]; then
  echo "[frozen-builtins] missing lock file: $LOCK_FILE" >&2
  exit 1
fi
if [[ ! -f "$SRC_FILE" ]]; then
  echo "[frozen-builtins] missing source file: $SRC_FILE" >&2
  exit 1
fi

extract_lock() {
  awk '
    {
      line=$0
      sub(/#.*/, "", line)
      gsub(/[[:space:]]/, "", line)
      if (line != "") print line
    }
  ' "$LOCK_FILE" | LC_ALL=C sort -u
}

extract_current() {
  if command -v rg >/dev/null 2>&1; then
    rg -o 'builtin_func_sym\("[^\"]+"' "$SRC_FILE" \
      | sed -E 's/.*\("([^\"]+)"/\1/' \
      | LC_ALL=C sort -u
    return
  fi
  grep -oE 'builtin_func_sym\("[^"]+"' "$SRC_FILE" \
    | sed -E 's/.*\("([^"]+)"/\1/' \
    | LC_ALL=C sort -u
}

LOCK_TMP="$(mktemp)"
CUR_TMP="$(mktemp)"
ONLY_CUR="$(mktemp)"
ONLY_LOCK="$(mktemp)"
trap 'rm -f "$LOCK_TMP" "$CUR_TMP" "$ONLY_CUR" "$ONLY_LOCK"' EXIT

extract_lock > "$LOCK_TMP"
extract_current > "$CUR_TMP"

comm -23 "$CUR_TMP" "$LOCK_TMP" > "$ONLY_CUR"
comm -13 "$CUR_TMP" "$LOCK_TMP" > "$ONLY_LOCK"

if [[ -s "$ONLY_CUR" || -s "$ONLY_LOCK" ]]; then
  echo "[frozen-builtins] builtin/intrinsic set changed." >&2
  if [[ -s "$ONLY_CUR" ]]; then
    echo "[frozen-builtins] added entries:" >&2
    sed 's/^/  + /' "$ONLY_CUR" >&2
  fi
  if [[ -s "$ONLY_LOCK" ]]; then
    echo "[frozen-builtins] removed entries:" >&2
    sed 's/^/  - /' "$ONLY_LOCK" >&2
  fi
  echo "[frozen-builtins] policy: do not add builtin/intrinsic without explicit design decision." >&2
  echo "[frozen-builtins] if intentional, update $LOCK_FILE in the same PR with rationale." >&2
  exit 1
fi

echo "[frozen-builtins] ok: $(wc -l < "$CUR_TMP" | tr -d ' ') entries match lock"
