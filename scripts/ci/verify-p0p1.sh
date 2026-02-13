#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

echo "[verify] p0/p1: rolling selfhost smoke"
"$ROOT/scripts/ci/rolling-selfhost.sh" test

echo "[verify] p0/p1: full suite"
COMPILER_BIN="$($ROOT/scripts/ci/rolling-selfhost.sh print-bin | tail -n 1)"
(
  cd "$ROOT"
  "$COMPILER_BIN" test-pkg target/debug/vox.p0p1
)

echo "[verify] p0/p1 closure gate passed"
