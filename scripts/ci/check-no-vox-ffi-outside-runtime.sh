#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# Policy:
# - During std layering convergence, vox_* FFI imports are allowed only in a
#   small, audited set of std modules where runtime bridge symbols are still
#   consumed.
# - tests are excluded because they intentionally validate compiler behavior.
output=$(find "$ROOT/src" -type f -name '*.vox' ! -name '*_test.vox' \
  ! -path "$ROOT/src/std/runtime/runtime.vox" \
  ! -path "$ROOT/src/std/fs/file_common.vox" \
  ! -path "$ROOT/src/std/sys/sys_common.vox" \
  ! -path "$ROOT/src/std/sys/sys_linux.vox" \
  ! -path "$ROOT/src/std/sys/sys_darwin.vox" \
  ! -path "$ROOT/src/std/sys/sys_windows.vox" \
  -print0 \
  | xargs -0 grep -nHE '@ffi_import[[:space:]]*\([[:space:]]*"c"[[:space:]]*,[[:space:]]*"vox_[[:alnum:]_]*"' || true)

if [[ -n "$output" ]]; then
  echo "[vox-ffi-gate] @ffi_import(\"c\", \"vox_*\") is disallowed outside audited std modules." >&2
  echo "[vox-ffi-gate] allowed: std/runtime, std/fs/file_common, std/sys platform bridges." >&2
  echo "$output" | sed 's/^/  - /' >&2
  exit 1
fi

echo "[vox-ffi-gate] ok: vox_* imports are confined to audited std modules"
