#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

if ! COMPILER_BIN_RAW="$("$ROOT/scripts/ci/rolling-selfhost.sh" print-bin)"; then
  exit $?
fi

COMPILER_BIN="$(printf '%s\n' "$COMPILER_BIN_RAW" | awk 'NF { last = $0 } END { print last }')"
if [[ -z "$COMPILER_BIN" ]]; then
  echo "[selfhost] rolling-selfhost did not print compiler path" >&2
  exit 1
fi
if [[ ! -x "$COMPILER_BIN" ]]; then
  echo "[selfhost] invalid compiler bin: $COMPILER_BIN" >&2
  exit 1
fi

printf '%s\n' "$COMPILER_BIN"
