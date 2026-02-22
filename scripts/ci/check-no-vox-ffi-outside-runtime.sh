#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# Policy:
# - std/runtime/runtime.vox is the only production source file allowed to bind
#   C symbols prefixed with vox_*. This keeps compatibility shims contained.
# - tests are excluded because they intentionally validate compiler behavior.
output=$(find "$ROOT/src" -type f -name '*.vox' ! -name '*_test.vox' \
  ! -path "$ROOT/src/std/runtime/runtime.vox" -print0 \
  | xargs -0 grep -nHE '@ffi_import[[:space:]]*\([[:space:]]*"c"[[:space:]]*,[[:space:]]*"vox_[[:alnum:]_]*"' || true)

if [[ -n "$output" ]]; then
  echo "[vox-ffi-gate] @ffi_import(\"c\", \"vox_*\") is disallowed outside src/std/runtime/runtime.vox." >&2
  echo "[vox-ffi-gate] use platform sys/ffi bindings in std/* instead of expanding compatibility shims." >&2
  echo "$output" | sed 's/^/  - /' >&2
  exit 1
fi

echo "[vox-ffi-gate] ok: no @ffi_import(\"c\", \"vox_*\") outside src/std/runtime/runtime.vox"
