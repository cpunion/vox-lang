#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
ALLOW_FILE="$ROOT/scripts/release/bootstrap-intrinsics.allow"

if [[ ! -f "$ALLOW_FILE" ]]; then
  echo "[intrinsics] missing allowlist: $ALLOW_FILE" >&2
  exit 1
fi

extract_allow() {
  awk '
    {
      line=$0
      sub(/#.*/, "", line)
      gsub(/[[:space:]]/, "", line)
      if (line != "") print line
    }
  ' "$ALLOW_FILE" | LC_ALL=C sort -u
}

extract_used() {
  (
    cd "$ROOT"
    rg --no-filename -o "__[A-Za-z0-9_]+" src/std -g '*.vox' 2>/dev/null \
      | LC_ALL=C sort -u
  )
}

ALLOW_TMP="$(mktemp)"
USED_TMP="$(mktemp)"
ONLY_USED_TMP="$(mktemp)"
ONLY_ALLOW_TMP="$(mktemp)"
trap 'rm -f "$ALLOW_TMP" "$USED_TMP" "$ONLY_USED_TMP" "$ONLY_ALLOW_TMP"' EXIT

extract_allow > "$ALLOW_TMP"
extract_used > "$USED_TMP"

comm -23 "$USED_TMP" "$ALLOW_TMP" > "$ONLY_USED_TMP"
comm -13 "$USED_TMP" "$ALLOW_TMP" > "$ONLY_ALLOW_TMP"

if [[ -s "$ONLY_USED_TMP" ]]; then
  echo "[intrinsics] unsupported reserved intrinsics used by src/std:" >&2
  sed 's/^/  - /' "$ONLY_USED_TMP" >&2
  echo "[intrinsics] update compiler+bootstrap first, then allowlist if intended." >&2
  exit 1
fi

echo "[intrinsics] ok: $(wc -l < "$USED_TMP" | tr -d ' ') reserved intrinsic(s) used by src/std"
if [[ -s "$ONLY_ALLOW_TMP" ]]; then
  echo "[intrinsics] note: allowlist has unused entries (kept for compatibility):"
  sed 's/^/  - /' "$ONLY_ALLOW_TMP"
fi
