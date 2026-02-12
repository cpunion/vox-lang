#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

echo "[verify] stage2 p0/p1: self-host stage2 suite (items 1-11 baseline)"
(
  cd "$ROOT/compiler/stage0"
  VOX_RUN_SELFHOST_TESTS=1 go test ./cmd/vox -run 'TestStage1BuildsStage2AndRunsStage2Tests' -count=1
)

echo "[verify] stage2 p0/p1: lock/dep integration (item 12 + lock UX)"
(
  cd "$ROOT/compiler/stage0"
  VOX_RUN_SELFHOST_TESTS=1 go test ./cmd/vox -run 'TestStage1BuildPkgWritesLockfile|TestStage1BuildPkgFailsWhenLockDigestMismatch|TestStage1BuildPkgSupportsVersionDependencyFromRegistryCache|TestStage1BuildPkgSupportsGitDependency|TestStage1BuildsStage2LockMismatchDiagnosticsConsistentAcrossBuildAndTest' -count=1
)

echo "[verify] stage2 p0/p1: test framework metadata/rerun coverage (item 9)"
(
  cd "$ROOT/compiler/stage0"
  go test ./cmd/vox -run 'TestParseTestOptionsAndDir_RunAndRerun|TestParseTestOptionsAndDir_Jobs|TestParseTestOptionsAndDir_JSON|TestWriteFailedTests_JSONCacheFormat|TestBuildJSONTestReport|TestBuildJSONTestReport_ListOnlyIncludesModuleDetails|TestInterpTestRerunFailed|TestPrintSelectionSummary|TestModuleTestWorkers' -count=1
)

echo "[verify] stage2 p0/p1 closure gate passed"
