#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# Gate scope is the language-facing surface:
# - std library sources
# - top-level CLI package sources (src/main*.vox)
# Internal compiler/backend sources may legitimately mention compatibility bridge
# symbol text while generating C runtime code.
output=$(
  (
    find "$ROOT/src/std" -type f -name '*.vox' -print0
    find "$ROOT/src" -maxdepth 1 -type f -name 'main*.vox' -print0
  ) | xargs -0 grep -nH "vox_builtin_" || true
)

if [[ -n "$output" ]]; then
  echo "[builtin-alias] vox_builtin_* symbols are disallowed in std/cli source." >&2
  echo "[builtin-alias] use @ffi_import + @build in std/* instead." >&2
  echo "$output" | sed 's/^/  - /' >&2
  exit 1
fi

echo "[builtin-alias] ok: no vox_builtin_* in std/cli source"
