set shell := ["bash", "-eu", "-o", "pipefail", "-c"]

default:
  @just --list

fmt:
  @echo "[fmt] no formatter configured for .vox yet"

test-active:
  ./scripts/ci/rolling-selfhost.sh build
  ./scripts/ci/rolling-selfhost.sh test

test:
  just test-active
  @start=$$(date +%s); \
  COMPILER_BIN=$$(./scripts/ci/rolling-selfhost.sh print-bin | tail -n 1); \
  cd examples/c_demo; \
  "$$COMPILER_BIN" test-pkg target/compiler_examples.test; \
  end=$$(date +%s); \
  echo "[time] compiler test-pkg examples/c_demo: $$((end-start))s"

test-p0p1:
  ./scripts/ci/verify-p0p1.sh

audit-vox-lines max="140":
  @files=$$(find src examples -name '*.vox' -type f 2>/dev/null); \
  if [ -z "$$files" ]; then \
    echo "[audit] no .vox files found"; \
    exit 0; \
  fi; \
  awk -v max="{{max}}" 'length($$0) > max { printf "%s:%d:%d\\n", FILENAME, FNR, length($$0); count++ } END { printf "[audit] %d line(s) longer than %d chars\\n", count + 0, max }' $$files

release-bundle version:
  ./scripts/release/build-release-bundle.sh {{version}}

release-verify version:
  ./scripts/release/verify-release-bundle.sh {{version}}

release-dry-run version:
  ./scripts/release/dry-run-rolling.sh {{version}}
