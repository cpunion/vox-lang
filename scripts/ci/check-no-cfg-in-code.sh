#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# Policy:
# - compiler keeps @cfg(...) support
# - production sources should use file-level @build(...) for platform partitioning
# - @cfg(...) is still allowed in tests validating compiler behavior

output=$(find "$ROOT/src" -type f -name '*.vox' ! -name '*_test.vox' -print0 \
  | xargs -0 grep -nH "@cfg(" || true)

if [[ -n "$output" ]]; then
  echo "[build-style] @cfg(...) is disallowed in non-test source files." >&2
  echo "[build-style] use file-level @build(...) for platform partitioning." >&2
  echo "$output" | sed 's/^/  - /' >&2
  exit 1
fi

echo "[build-style] ok: no @cfg(...) in non-test source files"
