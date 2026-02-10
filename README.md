# Vox Language

This repository contains the evolving specification and the stage0 bootstrap compiler for the Vox programming language.

Start here:

- `docs/00-overview.md`
- `docs/README.md`
- `docs/20-bootstrap.md`
- `docs/21-stage1-compiler.md`

Current development policy:

- `stage0`: frozen maintenance (regressions/stability only).
- `stage1`: frozen bootstrap line (only fixes that unblock `stage1 -> stage2`).
- `stage2`: active language/compiler evolution line.

Recommended gates:

```bash
make test-active   # stage0 unit + stage1->stage2 bootstrap + stage2 test-pkg suite
make test          # full gate (includes stage1 tests and examples)
```

Stage0 compiler (Go):

```bash
cd compiler/stage0
go test ./...
go run ./cmd/vox init ../../examples/hello
go run ./cmd/vox build ../../examples/hello
go run ./cmd/vox run ../../examples/hello
```
