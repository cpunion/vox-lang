#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

output=$(find "$ROOT/src" -type f -name '*.vox' -print0 \
  | xargs -0 grep -nH "vox_rt_" || true)

if [[ -n "$output" ]]; then
  echo "[runtime-alias] vox_rt_* symbols are disallowed in src/**/*.vox." >&2
  echo "[runtime-alias] bind std/runtime directly to vox_host_* instead." >&2
  echo "$output" | sed 's/^/  - /' >&2
  exit 1
fi

echo "[runtime-alias] ok: no vox_rt_* in src/**/*.vox"
