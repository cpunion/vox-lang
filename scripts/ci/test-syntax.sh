#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

JOBS="${VOX_SYNTAX_TEST_JOBS:-4}"
OUT="${VOX_SYNTAX_TEST_OUT:-target/syntax.acceptance.test}"

if ! COMPILER_BIN_RAW="$(./scripts/ci/rolling-selfhost.sh print-bin)"; then
  exit $?
fi
COMPILER_BIN="$(printf '%s\n' "$COMPILER_BIN_RAW" | tail -n 1)"
if [[ ! -x "$COMPILER_BIN" ]]; then
  echo "[syntax] invalid compiler bin: $COMPILER_BIN" >&2
  exit 1
fi
echo "[syntax] compiler: $COMPILER_BIN"
echo "[syntax] jobs: $JOBS"

(
  cd tests/syntax
  VOX_ROOT=../.. "$COMPILER_BIN" test "--jobs=$JOBS" "$OUT"
)
