.PHONY: fmt test test-go test-stage1 test-stage1-c test-stage1-interp test-stage2 test-stage2-tests \
	test-examples test-examples-c test-examples-interp test-stage1-toolchain test-selfhost test-stage2-selfhost \
	test-active audit-vox-lines

fmt:
	cd compiler/stage0 && gofmt -w $$(find . -name '*.go' -type f)

# Run all tests (stage0 Go unit tests + stage1 Vox tests via stage0 + example package tests).
test: test-go test-stage1 test-examples

# Active development gate (stage2-first): keep stage0/unit stable, validate stage2, and keep bootstrap chain green.
test-active: test-go test-stage2

test-go:
	cd compiler/stage0 && go test ./...

test-stage1: test-stage1-c test-stage1-interp

test-stage1-c:
	@set -e; \
	start=$$(date +%s); \
	cd compiler/stage0; \
	go run ./cmd/vox test --engine=c ../stage1; \
	end=$$(date +%s); \
	echo "[time] vox test --engine=c ../stage1: $$((end-start))s"

test-stage1-interp:
	@set -e; \
	start=$$(date +%s); \
	cd compiler/stage0; \
	go run ./cmd/vox test --engine=interp ../stage1; \
	end=$$(date +%s); \
	echo "[time] vox test --engine=interp ../stage1: $$((end-start))s"

# Stage2 gate (frozen stage1 + active stage2):
# 1) bootstrap E2E: stage1 -> stage2 -> sample package
# 2) stage2 self-test suite via stage2 test-pkg
test-stage2: test-stage2-selfhost test-stage2-tests

test-examples: test-examples-c test-examples-interp

test-examples-c:
	@set -e; \
	start=$$(date +%s); \
	cd compiler/stage0; \
	go run ./cmd/vox test --engine=c ../../examples/c_demo; \
	end=$$(date +%s); \
	echo "[time] vox test --engine=c ../../examples/c_demo: $$((end-start))s"

test-examples-interp:
	@set -e; \
	start=$$(date +%s); \
	cd compiler/stage0; \
	go run ./cmd/vox test --engine=interp ../../examples/c_demo; \
	end=$$(date +%s); \
	echo "[time] vox test --engine=interp ../../examples/c_demo: $$((end-start))s"

# Stage1 CLI/toolchain integration tests (run without test cache).
test-stage1-toolchain:
	cd compiler/stage0 && go test ./cmd/vox -run TestStage1 -count=1

# Dedicated self-hosting regression gate.
test-selfhost:
	cd compiler/stage0 && VOX_RUN_SELFHOST_TESTS=1 go test ./cmd/vox -run 'TestStage1ToolchainSelfBuildsStage1AndBuildsPackage|TestStage1SelfBuiltCompilerIsQuietOnSuccess' -count=1

# Dedicated stage2 bootstrap gate: stage1 -> stage2 -> sample package.
test-stage2-selfhost:
	cd compiler/stage0 && VOX_RUN_SELFHOST_TESTS=1 go test ./cmd/vox -run 'TestStage1BuildsStage2AndBuildsPackage' -count=1

# Dedicated stage2 suite gate: stage1 -> stage2 -> stage2 test-pkg.
test-stage2-tests:
	cd compiler/stage0 && VOX_RUN_SELFHOST_TESTS=1 go test ./cmd/vox -run 'TestStage1BuildsStage2AndRunsStage2Tests' -count=1

# Audit long lines in Vox sources (default max width: 140, override with MAX=<n>).
audit-vox-lines:
	@set -e; \
	max=$${MAX:-140}; \
	files=$$(find compiler/stage1/src examples -name '*.vox' -type f 2>/dev/null); \
	if [ -z "$$files" ]; then \
		echo "[audit] no .vox files found"; \
		exit 0; \
	fi; \
	awk -v max="$$max" 'length($$0) > max { printf "%s:%d:%d\n", FILENAME, FNR, length($$0); count++ } END { printf "[audit] %d line(s) longer than %d chars\n", count + 0, max }' $$files
