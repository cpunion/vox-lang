#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# Policy:
# - All vox_* FFI imports have been eliminated from Vox source files.
# - tests are excluded because they intentionally validate compiler behavior.
output=$(find "$ROOT/src" -type f -name '*.vox' ! -name '*_test.vox' \
  -print0 \
  | xargs -0 grep -nHE '@ffi_import[[:space:]]*\([[:space:]]*"c"[[:space:]]*,[[:space:]]*"vox_[[:alnum:]_]*"' || true)

if [[ -n "$output" ]]; then
  echo "[vox-ffi-gate] @ffi_import(\"c\", \"vox_*\") is disallowed in Vox source files." >&2
  echo "$output" | sed 's/^/  - /' >&2
  exit 1
fi

echo "[vox-ffi-gate] ok: no vox_* imports found in source files"
