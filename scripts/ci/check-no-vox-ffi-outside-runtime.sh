#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# Policy:
# - During std layering convergence, vox_* FFI imports are allowed only in a
#   small, audited set of modules:
#   - src/std/runtime/runtime.vox (runtime基础能力: wake/atomic)
#   - src/std/os/os.vox (os语义封装: args/env/process/fs-walk bridge)
#   - src/std/time/time.vox (time语义封装)
#   - src/std/sys/sys_common.vox (sys网络兼容垫片，后续逐步下沉到平台FFI)
# - tests are excluded because they intentionally validate compiler behavior.
output=$(find "$ROOT/src" -type f -name '*.vox' ! -name '*_test.vox' \
  ! -path "$ROOT/src/std/runtime/runtime.vox" \
  ! -path "$ROOT/src/std/os/os.vox" \
  ! -path "$ROOT/src/std/time/time.vox" \
  ! -path "$ROOT/src/std/sys/sys_common.vox" \
  -print0 \
  | xargs -0 grep -nHE '@ffi_import[[:space:]]*\([[:space:]]*"c"[[:space:]]*,[[:space:]]*"vox_[[:alnum:]_]*"' || true)

if [[ -n "$output" ]]; then
  echo "[vox-ffi-gate] @ffi_import(\"c\", \"vox_*\") is disallowed outside audited std modules." >&2
  echo "[vox-ffi-gate] allowed: std/runtime, std/os, std/time, std/sys/sys_common." >&2
  echo "$output" | sed 's/^/  - /' >&2
  exit 1
fi

echo "[vox-ffi-gate] ok: vox_* imports are confined to audited std modules"
