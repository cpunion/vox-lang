.PHONY: fmt test test-go test-stage1 test-stage1-c test-stage1-interp test-examples \
	test-examples-c test-examples-interp test-stage1-toolchain test-selfhost audit-vox-lines

fmt:
	cd compiler/stage0 && gofmt -w $$(find . -name '*.go' -type f)

# Run all tests (stage0 Go unit tests + stage1 Vox tests via stage0 + example package tests).
test: test-go test-stage1 test-examples

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
