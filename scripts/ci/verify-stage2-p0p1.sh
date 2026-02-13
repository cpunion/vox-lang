#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

echo "[verify] stage2 p0/p1: rolling selfhost smoke"
"$ROOT/scripts/ci/stage2-rolling-selfhost.sh" test

echo "[verify] stage2 p0/p1: full stage2 test suite"
STAGE2_BIN="$("$ROOT/scripts/ci/stage2-rolling-selfhost.sh" print-bin | tail -n 1)"
(
  cd "$ROOT/compiler/stage2"
  "$STAGE2_BIN" test-pkg target/debug/vox_stage2.p0p1
)

echo "[verify] stage2 p0/p1 closure gate passed"
