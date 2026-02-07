.PHONY: test test-go test-stage1 test-stage1-c test-stage1-interp test-examples \
	test-examples-c test-examples-interp

# Run all tests (stage0 Go unit tests + stage1 Vox tests via stage0 + example package tests).
test: test-go test-stage1 test-examples

test-go:
	cd compiler/stage0 && go test ./...

test-stage1: test-stage1-c test-stage1-interp

test-stage1-c:
	cd compiler/stage0 && go run ./cmd/vox test --engine=c ../stage1

test-stage1-interp:
	cd compiler/stage0 && go run ./cmd/vox test --engine=interp ../stage1

test-examples: test-examples-c test-examples-interp

test-examples-c:
	cd compiler/stage0 && go run ./cmd/vox test --engine=c ../../examples/c_demo

test-examples-interp:
	cd compiler/stage0 && go run ./cmd/vox test --engine=interp ../../examples/c_demo

