.PHONY: fmt test test-stage2 test-stage2-selfhost test-stage2-tests test-stage2-p0p1 \
	test-examples test-active audit-vox-lines release-bundle release-verify release-dry-run

fmt:
	@echo "[fmt] no formatter configured for .vox yet"

# Run core repo tests: stage2 rolling selfhost gates + example package smoke.
test: test-stage2 test-examples

# Active development gate.
test-active: test-stage2

# Stage2 rolling bootstrap gate (stage2(prev) -> stage2(new)).
test-stage2: test-stage2-selfhost test-stage2-tests

test-stage2-selfhost:
	./scripts/ci/stage2-rolling-selfhost.sh build

test-stage2-tests:
	./scripts/ci/stage2-rolling-selfhost.sh test

# Stage2 P0/P1 closure gate (docs/archive/25-stage2-p0p1-closure.md).
test-stage2-p0p1:
	./scripts/ci/verify-stage2-p0p1.sh

# Example package smoke uses rolling-built stage2 tool.
test-examples:
	@set -e; \
	start=$$(date +%s); \
	STAGE2_BIN=$$(./scripts/ci/stage2-rolling-selfhost.sh print-bin | tail -n 1); \
	cd examples/c_demo; \
	"$$STAGE2_BIN" test-pkg target/stage2_examples.test; \
	end=$$(date +%s); \
	echo "[time] stage2 test-pkg examples/c_demo: $$((end-start))s"

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

# Build local release bundle for current host platform (stage2 only).
# Usage: make release-bundle VERSION=v0.1.0
release-bundle:
	@if [ -z "$(VERSION)" ]; then \
		echo "usage: make release-bundle VERSION=v0.1.0"; \
		exit 1; \
	fi
	./scripts/release/build-release-bundle.sh $(VERSION)

# Verify a local release bundle archive and rolling bootstrap metadata.
# Usage: make release-verify VERSION=v0.1.0
release-verify:
	@if [ -z "$(VERSION)" ]; then \
		echo "usage: make release-verify VERSION=v0.1.0"; \
		exit 1; \
	fi
	./scripts/release/verify-release-bundle.sh $(VERSION)

# Local rolling bootstrap rehearsal (build bundle + smoke + verify).
# Usage: make release-dry-run VERSION=v0.1.0-rc1
release-dry-run:
	@if [ -z "$(VERSION)" ]; then \
		echo "usage: make release-dry-run VERSION=v0.1.0-rc1"; \
		exit 1; \
	fi
	./scripts/release/dry-run-rolling.sh $(VERSION)
