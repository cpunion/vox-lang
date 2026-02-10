# Vox Stage0 (Go)

This is the stage0 bootstrap compiler for Vox.

Supported (initial subset):

- Basic types: `i32`, `i64`, `bool`, `String`, `()`
- Nominal types: `struct`, `enum` (tagged-union lowering in C backend)
- Functions: `fn`, params, calls, `let`/`let mut`, assignment, `return`
- Control flow: `if/else`, `while`, `match` (minimal patterns)
- Modules/packages: `import "path" as x` (minimal), multi-file packages
- Generics (minimal): generic functions + monomorphization
- Package manifest: `vox.toml` (minimal parsing; path deps validated)
- Testing: `vox test` discovers and runs `test_*` functions
- Engines: `--engine=c` (C codegen + cc) and `--engine=interp` (interpreter)

Commands:

```bash
go test ./...

go run ./cmd/vox init <dir>
go run ./cmd/vox ir <dir>
go run ./cmd/vox build <dir>
go run ./cmd/vox run <dir>
go run ./cmd/vox test <dir> [--engine=c|interp]
```

Notes:

- The language spec lives in `../../docs/`.
- Stage0 implements a small but growing subset; comptime/macro/trait/effects are out of scope.
- Stage1 is the frozen self-hosting bootstrap compiler line (implemented in Vox).
- Stage2 is the active Vox compiler evolution line (language/compiler features continue here).
- Stage3 is the developer toolchain phase (fmt/lint/doc/LSP).
- Daily mainline gate: run `make test-active` from repo root.
