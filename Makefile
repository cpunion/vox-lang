.PHONY: fmt test test-rolling test-selfhost-build test-selfhost-gate test-selfhost-smoke test-public-api test-p0p1 test-intrinsics \
	test-examples test-active audit-vox-lines release-bundle release-verify release-dry-run \
	release-source-bundle release-source-verify

fmt:
	@echo "[fmt] no formatter configured for .vox yet"

# Run core repo tests: rolling selfhost gates + example package smoke.
test: test-intrinsics test-rolling test-examples

# Active development gate.
test-active: test-intrinsics test-rolling

# Guard std intrinsic usage against bootstrap compatibility drift.
test-intrinsics:
	./scripts/ci/check-std-intrinsics.sh

# Rolling bootstrap gate (previous compiler -> new compiler).
test-rolling: test-selfhost-build test-selfhost-gate

test-selfhost-build:
	./scripts/ci/rolling-selfhost.sh build

test-selfhost-gate:
	VOX_TEST_RUN='*api*' ./scripts/ci/rolling-selfhost.sh test

test-selfhost-smoke:
	./scripts/ci/rolling-selfhost.sh test

# Stable public library API contract gate.
test-public-api:
	@set -e; \
	COMPILER_BIN=$$(./scripts/ci/rolling-selfhost.sh print-bin | tail -n 1); \
	"$$COMPILER_BIN" test-pkg --run='*public_api*' target/debug/vox.public_api

# P0/P1 closure gate (docs/archive/25-p0p1-closure.md).
test-p0p1:
	./scripts/ci/verify-p0p1.sh

# Example package smoke uses rolling-built compiler.
test-examples:
	@set -e; \
	start=$$(date +%s); \
	COMPILER_BIN=$$(./scripts/ci/rolling-selfhost.sh print-bin | tail -n 1); \
	cd examples/c_demo; \
	"$$COMPILER_BIN" test-pkg target/compiler_examples.test; \
	end=$$(date +%s); \
	echo "[time] compiler test-pkg examples/c_demo: $$((end-start))s"

# Audit long lines in Vox sources (default max width: 140, override with MAX=<n>).
audit-vox-lines:
	@set -e; \
	max=$${MAX:-140}; \
	files=$$(find src examples -name '*.vox' -type f 2>/dev/null); \
	if [ -z "$$files" ]; then \
		echo "[audit] no .vox files found"; \
		exit 0; \
	fi; \
	awk -v max="$$max" 'length($$0) > max { printf "%s:%d:%d\n", FILENAME, FNR, length($$0); count++ } END { printf "[audit] %d line(s) longer than %d chars\n", count + 0, max }' $$files

# Build local release bundle for current host platform.
# Usage: make release-bundle VERSION=v0.2.0
release-bundle:
	@if [ -z "$(VERSION)" ]; then \
		echo "usage: make release-bundle VERSION=v0.2.0"; \
		exit 1; \
	fi
	./scripts/release/build-release-bundle.sh $(VERSION)

# Verify a local release bundle archive and rolling bootstrap metadata.
# Usage: make release-verify VERSION=v0.2.0
release-verify:
	@if [ -z "$(VERSION)" ]; then \
		echo "usage: make release-verify VERSION=v0.2.0"; \
		exit 1; \
	fi
	./scripts/release/verify-release-bundle.sh $(VERSION)

# Build local release source bundle.
# Usage: make release-source-bundle VERSION=v0.2.0
release-source-bundle:
	@if [ -z "$(VERSION)" ]; then \
		echo "usage: make release-source-bundle VERSION=v0.2.0"; \
		exit 1; \
	fi
	./scripts/release/build-source-bundle.sh $(VERSION)

# Verify local release source bundle.
# Usage: make release-source-verify VERSION=v0.2.0
release-source-verify:
	@if [ -z "$(VERSION)" ]; then \
		echo "usage: make release-source-verify VERSION=v0.2.0"; \
		exit 1; \
	fi
	./scripts/release/verify-source-bundle.sh $(VERSION)

# Local rolling bootstrap rehearsal (build bundle + smoke + verify).
# Usage: make release-dry-run VERSION=v0.2.0-rc1
release-dry-run:
	@if [ -z "$(VERSION)" ]; then \
		echo "usage: make release-dry-run VERSION=v0.2.0-rc1"; \
		exit 1; \
	fi
	./scripts/release/dry-run-rolling.sh $(VERSION)
