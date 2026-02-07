# Vox Language

This repository contains the evolving specification and the stage0 bootstrap compiler for the Vox programming language.

Start here:

- `docs/00-overview.md`
- `docs/README.md`

Stage0 compiler (Go):

```bash
cd compiler/stage0
go test ./...
go run ./cmd/vox init ../../examples/hello
go run ./cmd/vox build ../../examples/hello
go run ./cmd/vox run ../../examples/hello
```
