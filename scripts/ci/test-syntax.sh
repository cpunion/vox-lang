#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

JOBS="${VOX_SYNTAX_TEST_JOBS:-4}"
OUT="${VOX_SYNTAX_TEST_OUT:-target/syntax.acceptance.test}"

COMPILER_BIN="$(./scripts/ci/rolling-selfhost.sh print-bin | tail -n 1)"
echo "[syntax] compiler: $COMPILER_BIN"
echo "[syntax] jobs: $JOBS"

(
  cd tests/syntax
  VOX_ROOT=../.. "$COMPILER_BIN" test "--jobs=$JOBS" "$OUT"
)
