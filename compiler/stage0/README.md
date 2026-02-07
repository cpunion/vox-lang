# Vox Stage0 (Go)

This is the stage0 bootstrap compiler for Vox.

Supported (initial subset):

- Basic types: `i32`, `i64`, `bool`, `String`, `()`
- Functions: `fn`, params, calls, `let`/`let mut`, assignment, `return`, `if/else`
- Package manifest: `vox.toml` (minimal parsing; path deps validated)

Commands:

```bash
go test ./...

go run ./cmd/vox init <dir>
go run ./cmd/vox ir <dir>
go run ./cmd/vox build <dir>
go run ./cmd/vox run <dir>
go run ./cmd/vox test <dir>
```

Notes:

- The language spec lives in `../../docs/`.
- Generics, comptime, macros, imports, and stdlib are not implemented in stage0 yet.
- Stage1 is the self-hosting Vox compiler and includes IR/backends + build/package work.
- Stage2 is the developer toolchain phase (fmt/lint/doc/LSP).
