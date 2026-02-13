# Vox Spec Index

Vox is a Rust-like systems language with strong compile-time execution and a simplified memory model:

- Rust naming (`T`, `&T`, `Box[T]`, `Arc[T]`, `Weak[T]`)
- No user lifetimes (except `&'static T`)
- Borrowed references are ephemeral (cannot be stored/returned/captured long-term)

Note: Vox uses square brackets for generics: `Vec[T]`, `Result[T, E]`, and explicit type args at call site: `f[T](...)`.

Documents:

- `docs/00-overview.md`
- `docs/01-type-system.md`
- `docs/02-error-handling.md`
- `docs/03-module-package.md`
- `docs/04-generics-comptime.md`
- `docs/05-comptime-detailed.md`
- `docs/06-advanced-generics.md`
- `docs/07-memory-management.md`
- `docs/10-macro-system.md`
- `docs/11-package-management.md`
- `docs/12-testing-framework.md`
- `docs/13-standard-library.md`
- `docs/14-syntax-details.md`
- `docs/15-toolchain.md`
- `docs/16-platform-support.md`
- `docs/17-ffi-interop.md`
- `docs/18-diagnostics.md`
- `docs/19-ir-spec.md`
- `docs/20-bootstrap.md`
- `docs/24-release-process.md`
- `docs/27-active-backlog.md`

Archive:

- `docs/archive/21-stage1-compiler.md`
- `docs/archive/22-backlog.md`
- `docs/archive/23-backlog-next.md`
- `docs/archive/25-p0p1-closure.md`
- `docs/archive/26-closure-1-4-7-9.md`

Current policy:

- Mainline uses a single compiler implementation with rolling selfhost.
- Historical bootstrap (`stage0/stage1`) is archived in branch `archive/stage0-stage1`.
